package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/pkg/errors"
)

const (
	tokenTypeCustomer        = "customer"
	tokenTypeCustomerRefresh = "customer_refresh"
	tokenTypeRefresh         = "refresh"
	otpMaxAttempts           = 3
	otpCodeUpperBound        = 1000000

	// JWT standard claim keys (RFC 7519).
	claimSub  = "sub"
	claimType = "type"
	claimIat  = "iat"
	claimExp  = "exp"
	claimIss  = "iss"
	claimJti  = "jti"
)

// CustomerAuthConfig holds JWT and token configuration for customer auth.
type CustomerAuthConfig struct {
	JWTSecret            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Issuer               string
}

// CustomerAuthService implements domain.CustomerAuthService
type CustomerAuthService struct {
	otpRepo      domain.OTPRepository
	customerRepo domain.CustomerRepository
	tokenStore   domain.CustomerTokenStore
	smsGateway   domain.SMSGateway
	publisher    event.EventPublisher
	config       CustomerAuthConfig
	jwtSecret    []byte
}

// NewCustomerAuthService creates a new CustomerAuthService
func NewCustomerAuthService(
	otpRepo domain.OTPRepository,
	customerRepo domain.CustomerRepository,
	tokenStore domain.CustomerTokenStore,
	smsGateway domain.SMSGateway,
	publisher event.EventPublisher,
	cfg CustomerAuthConfig,
) *CustomerAuthService {
	return &CustomerAuthService{
		otpRepo:      otpRepo,
		customerRepo: customerRepo,
		tokenStore:   tokenStore,
		smsGateway:   smsGateway,
		publisher:    publisher,
		config:       cfg,
		jwtSecret:    []byte(cfg.JWTSecret),
	}
}

// SendOTP generates and sends an OTP to the given phone number
func (s *CustomerAuthService) SendOTP(ctx context.Context, phone string) error {
	code, err := generateOTPCode()
	if err != nil {
		return errors.Internal("Failed to generate OTP code")
	}

	otp := &domain.OTP{
		Phone:    phone,
		CodeHash: hashSHA256(code),
		Attempts: 0,
	}

	if err := s.otpRepo.Store(ctx, otp); err != nil {
		return err
	}

	if err := s.smsGateway.SendOTP(ctx, phone, code); err != nil {
		slog.ErrorContext(ctx, "Failed to send OTP SMS", "error", err)
		return errors.Internal("Failed to send OTP")
	}

	slog.InfoContext(ctx, "OTP sent", "phone", phone)
	return nil
}

// VerifyOTP verifies an OTP code and returns the customer, tokens, and whether the customer is new
func (s *CustomerAuthService) VerifyOTP(ctx context.Context, phone, code string) (*domain.Customer, *domain.TokenPair, bool, error) {
	otp, err := s.otpRepo.Get(ctx, phone)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil, false, errors.New(errors.ErrCodeInvalidToken, "OTP not found or expired")
		}
		return nil, nil, false, err
	}

	if otp.Attempts >= otpMaxAttempts {
		_ = s.otpRepo.Delete(ctx, phone)
		return nil, nil, false, errors.New(errors.ErrCodeInvalidToken, "Too many OTP attempts")
	}

	if incrErr := s.otpRepo.IncrementAttempts(ctx, phone); incrErr != nil {
		slog.WarnContext(ctx, "Failed to increment OTP attempts", "error", incrErr)
	}

	if hashSHA256(code) != otp.CodeHash {
		return nil, nil, false, errors.New(errors.ErrCodeInvalidCredentials, "Invalid OTP code")
	}

	if delErr := s.otpRepo.Delete(ctx, phone); delErr != nil {
		slog.WarnContext(ctx, "Failed to delete OTP after verification", "error", delErr)
	}

	customer, isNew, err := s.findOrCreateCustomer(ctx, phone)
	if err != nil {
		return nil, nil, false, err
	}

	if isNew {
		if pubErr := s.publisher.Publish(ctx, event.New(event.CustomerRegistered, customer)); pubErr != nil {
			slog.ErrorContext(ctx, "Failed to publish customer.registered event", "error", pubErr)
		}
	}

	tokens, err := s.generateTokenPair(customer)
	if err != nil {
		return nil, nil, false, err
	}

	if err := s.storeRefreshToken(ctx, customer.ID, tokens.RefreshToken); err != nil {
		return nil, nil, false, err
	}

	slog.InfoContext(ctx, "Customer authenticated", "customer_id", customer.ID, "is_new", isNew)
	return customer, tokens, isNew, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *CustomerAuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.Customer, *domain.TokenPair, error) {
	claims, err := s.parseJWTClaims(refreshToken, tokenTypeCustomerRefresh)
	if err != nil {
		return nil, nil, errors.New(errors.ErrCodeInvalidToken, "Invalid refresh token")
	}

	customerID, _ := claims[claimSub].(string)

	oldHash := hashSHA256(refreshToken)
	valid, err := s.tokenStore.ValidateToken(ctx, customerID, oldHash)
	if err != nil || !valid {
		return nil, nil, errors.New(errors.ErrCodeInvalidToken, "Refresh token has been revoked")
	}

	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, nil, err
	}

	if customer.Status != domain.CustomerStatusActive {
		return nil, nil, errors.New(errors.ErrCodeUserInactive, "Customer account is not active")
	}

	tokens, err := s.generateTokenPair(customer)
	if err != nil {
		return nil, nil, err
	}

	if err := s.storeRefreshToken(ctx, customer.ID, tokens.RefreshToken); err != nil {
		return nil, nil, err
	}

	if err := s.tokenStore.RevokeToken(ctx, customerID, oldHash); err != nil {
		slog.WarnContext(ctx, "Failed to revoke old customer refresh token", "error", err)
	}

	return customer, tokens, nil
}

// Logout revokes a customer's refresh token
func (s *CustomerAuthService) Logout(ctx context.Context, customerID, refreshToken string) error {
	if err := s.tokenStore.RevokeToken(ctx, customerID, hashSHA256(refreshToken)); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Customer logged out", "customer_id", customerID)
	return nil
}

// ValidateCustomerToken validates a customer access token and returns typed claims
func (s *CustomerAuthService) ValidateCustomerToken(ctx context.Context, tokenString string) (*domain.CustomerTokenClaims, error) {
	claims, err := s.parseJWTClaims(tokenString, tokenTypeCustomer)
	if err != nil {
		return nil, err
	}

	customerID, _ := claims[claimSub].(string)
	phone, _ := claims["phone"].(string)
	email, _ := claims["email"].(string)

	return &domain.CustomerTokenClaims{
		CustomerID: customerID,
		Phone:      phone,
		Email:      email,
	}, nil
}

// --- private helpers ---

// findOrCreateCustomer retrieves an existing customer by phone or creates a new one.
func (s *CustomerAuthService) findOrCreateCustomer(ctx context.Context, phone string) (*domain.Customer, bool, error) {
	customer, err := s.customerRepo.GetByPhone(ctx, phone)
	if err == nil {
		if !customer.PhoneVerified {
			customer.PhoneVerified = true
			customer.UpdatedAt = time.Now()
			if updateErr := s.customerRepo.Update(ctx, customer); updateErr != nil {
				slog.WarnContext(ctx, "Failed to update customer phone_verified", "error", updateErr)
			}
		}
		return customer, false, nil
	}

	if !errors.IsNotFound(err) {
		return nil, false, err
	}

	now := time.Now()
	customer = &domain.Customer{
		ID:            uuid.New().String(),
		Phone:         phone,
		PhoneVerified: true,
		Status:        domain.CustomerStatusActive,
	}
	customer.CreatedAt = now
	customer.UpdatedAt = now

	if err := s.customerRepo.Create(ctx, customer); err != nil {
		return nil, false, err
	}

	slog.InfoContext(ctx, "New customer created", "customer_id", customer.ID)
	return customer, true, nil
}

// parseJWTClaims parses a JWT, validates HMAC signing, and verifies the token type claim.
func (s *CustomerAuthService) parseJWTClaims(tokenString, expectedType string) (jwt.MapClaims, error) {
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

	tokenType, _ := claims[claimType].(string)
	if tokenType != expectedType {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token type")
	}

	return claims, nil
}

// generateTokenPair generates access and refresh JWT tokens for a customer.
func (s *CustomerAuthService) generateTokenPair(customer *domain.Customer) (*domain.TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(s.config.AccessTokenDuration)
	refreshExpiry := now.Add(s.config.RefreshTokenDuration)

	accessClaims := jwt.MapClaims{
		claimSub:  customer.ID,
		"phone":   customer.Phone,
		"email":   customer.Email,
		claimType: tokenTypeCustomer,
		claimIat:  now.Unix(),
		claimExp:  accessExpiry.Unix(),
		claimIss:  s.config.Issuer,
		claimJti:  uuid.New().String(),
	}

	accessTokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, errors.Internal("Failed to generate customer access token")
	}

	refreshClaims := jwt.MapClaims{
		claimSub:  customer.ID,
		claimType: tokenTypeCustomerRefresh,
		claimIat:  now.Unix(),
		claimExp:  refreshExpiry.Unix(),
		claimIss:  s.config.Issuer,
		claimJti:  uuid.New().String(),
	}

	refreshTokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, errors.Internal("Failed to generate customer refresh token")
	}

	return &domain.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
	}, nil
}

// storeRefreshToken hashes and stores a refresh token in the token store.
func (s *CustomerAuthService) storeRefreshToken(ctx context.Context, customerID, refreshToken string) error {
	ttl := time.Now().Add(s.config.RefreshTokenDuration).Unix()
	if err := s.tokenStore.StoreToken(ctx, customerID, hashSHA256(refreshToken), ttl); err != nil {
		slog.ErrorContext(ctx, "Failed to store customer refresh token", "error", err)
		return errors.Internal("Failed to create session")
	}
	return nil
}

// hashSHA256 computes a SHA256 hash of the input and returns a hex string.
func hashSHA256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// generateOTPCode generates a cryptographically secure 6-digit OTP code.
func generateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(otpCodeUpperBound))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// Ensure interface compliance
var _ domain.CustomerAuthService = (*CustomerAuthService)(nil)
