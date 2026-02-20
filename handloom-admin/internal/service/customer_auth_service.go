package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// customerSMSGateway defines the SMS sending interface used by CustomerAuthService.
// This matches the actual implementation signature (which includes context).
type customerSMSGateway interface {
	SendOTP(ctx context.Context, phone, code string) error
}

// CustomerAuthService implements domain.CustomerAuthService
type CustomerAuthService struct {
	otpRepo              domain.OTPRepository
	customerRepo         domain.CustomerRepository
	tokenStore           domain.CustomerTokenStore
	smsGateway           customerSMSGateway
	logger               *logger.Logger
	jwtSecret            []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
	issuer               string
}

// NewCustomerAuthService creates a new CustomerAuthService
func NewCustomerAuthService(
	otpRepo domain.OTPRepository,
	customerRepo domain.CustomerRepository,
	tokenStore domain.CustomerTokenStore,
	smsGateway customerSMSGateway,
	logger *logger.Logger,
	jwtSecret string,
	accessTokenDuration time.Duration,
	refreshTokenDuration time.Duration,
	issuer string,
) *CustomerAuthService {
	return &CustomerAuthService{
		otpRepo:              otpRepo,
		customerRepo:         customerRepo,
		tokenStore:           tokenStore,
		smsGateway:           smsGateway,
		logger:               logger,
		jwtSecret:            []byte(jwtSecret),
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
		issuer:               issuer,
	}
}

// SendOTP generates and sends an OTP to the given phone number
func (s *CustomerAuthService) SendOTP(ctx context.Context, phone string) error {
	// Generate a cryptographically secure 6-digit code
	code, err := generateOTPCode()
	if err != nil {
		return errors.Internal("Failed to generate OTP code")
	}

	// Hash the code for storage
	codeHash := hashSHA256(code)

	// Create OTP record
	otp := &domain.OTP{
		Phone:    phone,
		CodeHash: codeHash,
		Attempts: 0,
	}

	// Store OTP (TTL is set inside the repository)
	if err := s.otpRepo.Store(ctx, otp); err != nil {
		return err
	}

	// Send OTP via SMS gateway
	if err := s.smsGateway.SendOTP(ctx, phone, code); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to send OTP SMS")
		return errors.Internal("Failed to send OTP")
	}

	s.logger.WithContext(ctx).Infof("OTP sent to %s", phone)
	return nil
}

// VerifyOTP verifies an OTP code and returns the customer, tokens, and whether the customer is new
func (s *CustomerAuthService) VerifyOTP(ctx context.Context, phone, code string) (*domain.Customer, *domain.TokenPair, bool, error) {
	// Get stored OTP
	otp, err := s.otpRepo.Get(ctx, phone)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil, false, errors.New(errors.ErrCodeInvalidToken, "OTP not found or expired")
		}
		return nil, nil, false, err
	}

	// Check max attempts (3)
	if otp.Attempts >= 3 {
		// Delete OTP to prevent further attempts
		_ = s.otpRepo.Delete(ctx, phone)
		return nil, nil, false, errors.New(errors.ErrCodeInvalidToken, "Too many OTP attempts")
	}

	// Increment attempts before verification
	if err := s.otpRepo.IncrementAttempts(ctx, phone); err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to increment OTP attempts")
	}

	// Verify the code hash
	codeHash := hashSHA256(code)
	if codeHash != otp.CodeHash {
		return nil, nil, false, errors.New(errors.ErrCodeInvalidCredentials, "Invalid OTP code")
	}

	// OTP verified — delete it
	if err := s.otpRepo.Delete(ctx, phone); err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to delete OTP after verification")
	}

	// Find or create customer
	isNewCustomer := false
	customer, err := s.customerRepo.GetByPhone(ctx, phone)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new customer
			isNewCustomer = true
			customer = &domain.Customer{
				ID:            uuid.New().String(),
				Phone:         phone,
				PhoneVerified: true,
				Status:        domain.CustomerStatusActive,
			}
			now := time.Now()
			customer.CreatedAt = now
			customer.UpdatedAt = now

			if err := s.customerRepo.Create(ctx, customer); err != nil {
				return nil, nil, false, err
			}

			s.logger.WithContext(ctx).Infof("New customer created: %s", customer.ID)
		} else {
			return nil, nil, false, err
		}
	} else {
		// Mark phone as verified if not already
		if !customer.PhoneVerified {
			customer.PhoneVerified = true
			customer.UpdatedAt = time.Now()
			if err := s.customerRepo.Update(ctx, customer); err != nil {
				s.logger.WithContext(ctx).WithError(err).Warn("Failed to update customer phone_verified")
			}
		}
	}

	// Generate JWT token pair
	tokens, err := s.generateCustomerTokenPair(customer)
	if err != nil {
		return nil, nil, false, err
	}

	// Store refresh token hash
	refreshHash := hashSHA256(tokens.RefreshToken)
	ttl := time.Now().Add(s.refreshTokenDuration).Unix()
	if err := s.tokenStore.StoreToken(ctx, customer.ID, refreshHash, ttl); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to store customer refresh token")
		return nil, nil, false, errors.Internal("Failed to create session")
	}

	s.logger.WithContext(ctx).Infof("Customer authenticated: %s (new=%v)", customer.ID, isNewCustomer)
	return customer, tokens, isNewCustomer, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *CustomerAuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.Customer, *domain.TokenPair, error) {
	// Parse the JWT to get customer ID
	claims, err := s.validateCustomerRefreshToken(refreshToken)
	if err != nil {
		return nil, nil, errors.New(errors.ErrCodeInvalidToken, "Invalid refresh token")
	}

	customerID := claims["sub"].(string)

	// Hash the refresh token and validate it in the store
	oldHash := hashSHA256(refreshToken)
	valid, err := s.tokenStore.ValidateToken(ctx, customerID, oldHash)
	if err != nil || !valid {
		return nil, nil, errors.New(errors.ErrCodeInvalidToken, "Refresh token has been revoked")
	}

	// Get customer
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, nil, err
	}

	// Check customer status
	if customer.Status != domain.CustomerStatusActive {
		return nil, nil, errors.New(errors.ErrCodeUserInactive, "Customer account is not active")
	}

	// Generate new token pair
	tokens, err := s.generateCustomerTokenPair(customer)
	if err != nil {
		return nil, nil, err
	}

	// Store new refresh token hash
	newHash := hashSHA256(tokens.RefreshToken)
	ttl := time.Now().Add(s.refreshTokenDuration).Unix()
	if err := s.tokenStore.StoreToken(ctx, customer.ID, newHash, ttl); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to store new customer refresh token")
		return nil, nil, errors.Internal("Failed to create session")
	}

	// Revoke old refresh token (best-effort)
	if err := s.tokenStore.RevokeToken(ctx, customerID, oldHash); err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to revoke old customer refresh token")
	}

	return customer, tokens, nil
}

// Logout revokes a customer's refresh token
func (s *CustomerAuthService) Logout(ctx context.Context, customerID, refreshToken string) error {
	tokenHash := hashSHA256(refreshToken)
	if err := s.tokenStore.RevokeToken(ctx, customerID, tokenHash); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Customer logged out: %s", customerID)
	return nil
}

// ValidateCustomerToken validates a customer access token and returns the claims
func (s *CustomerAuthService) ValidateCustomerToken(ctx context.Context, tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token")
	}

	if !token.Valid {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token claims")
	}

	// Verify it's a customer token
	tokenType, _ := claims["type"].(string)
	if tokenType != "customer" {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Not a customer token")
	}

	return claims, nil
}

// generateCustomerTokenPair generates access and refresh tokens for a customer
func (s *CustomerAuthService) generateCustomerTokenPair(customer *domain.Customer) (*domain.TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(s.accessTokenDuration)
	refreshExpiry := now.Add(s.refreshTokenDuration)

	// Access token claims
	accessClaims := jwt.MapClaims{
		"sub":   customer.ID,
		"phone": customer.Phone,
		"email": customer.Email,
		"type":  "customer",
		"iat":   now.Unix(),
		"exp":   accessExpiry.Unix(),
		"iss":   s.issuer,
		"jti":   uuid.New().String(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, errors.Internal("Failed to generate customer access token")
	}

	// Refresh token claims
	refreshClaims := jwt.MapClaims{
		"sub":  customer.ID,
		"type": "customer_refresh",
		"iat":  now.Unix(),
		"exp":  refreshExpiry.Unix(),
		"iss":  s.issuer,
		"jti":  uuid.New().String(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, errors.Internal("Failed to generate customer refresh token")
	}

	return &domain.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
	}, nil
}

// validateCustomerRefreshToken validates a customer refresh token and returns claims
func (s *CustomerAuthService) validateCustomerRefreshToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token claims")
	}

	// Verify it's a customer refresh token
	tokenType, _ := claims["type"].(string)
	if tokenType != "customer_refresh" {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Not a customer refresh token")
	}

	return claims, nil
}

// hashSHA256 computes a SHA256 hash of the input and returns a hex string
func hashSHA256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// generateOTPCode generates a cryptographically secure 6-digit OTP code
func generateOTPCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// Ensure interface compliance
var _ domain.CustomerAuthService = (*CustomerAuthService)(nil)
