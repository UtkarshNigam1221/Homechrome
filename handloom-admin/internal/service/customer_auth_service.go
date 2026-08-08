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
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
	"github.com/handloom/admin/pkg/telemetry"
)

const (
	tokenTypeCustomer        = "customer"
	tokenTypeCustomerRefresh = "customer_refresh"
	tokenTypeRefresh         = "refresh"
	otpMaxAttempts           = 3
	otpCodeUpperBound        = 1000000

	// How long a rotated refresh token stays usable. Refreshes overlap: a
	// second browser tab, or a retry, presents the same token while the
	// first rotation is still in flight. Deleting it outright answers the
	// straggler with a 401, and the handler clears both auth cookies on
	// that path — logging the customer out mid-session. Observed refresh
	// handler durations reach ~3.2s on a cold Lambda, so the window has to
	// clear that comfortably.
	//
	// The window cannot extend a token past its natural life: RefreshToken
	// validates the JWT's exp claim before it ever consults the store, so
	// bumping the DB TTL on a nearly-expired token is inert.
	//
	// Note for anyone adding token-reuse detection later: this makes it
	// harder. Distinguishing a benign grace-window replay from a genuine
	// stolen-token replay needs the rotated row to carry its successor's
	// hash, so only a post-grace presentation raises the alarm.
	refreshGracePeriod = 30 * time.Second

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
	config       CustomerAuthConfig
	jwtSecret    []byte
}

// NewCustomerAuthService creates a new CustomerAuthService
func NewCustomerAuthService(
	otpRepo domain.OTPRepository,
	customerRepo domain.CustomerRepository,
	tokenStore domain.CustomerTokenStore,
	smsGateway domain.SMSGateway,
	cfg CustomerAuthConfig,
) *CustomerAuthService {
	return &CustomerAuthService{
		otpRepo:      otpRepo,
		customerRepo: customerRepo,
		tokenStore:   tokenStore,
		smsGateway:   smsGateway,
		config:       cfg,
		jwtSecret:    []byte(cfg.JWTSecret),
	}
}

// SendOTP generates and sends an OTP to the given phone number
func (s *CustomerAuthService) SendOTP(ctx context.Context, phone string) error {
	ctx, span := telemetry.StartServiceSpan(ctx, "customer_auth", "send_otp")

	code, err := generateOTPCode()
	if err != nil {
		genErr := errors.Internal("Failed to generate OTP code")
		span.EndWithError(genErr)
		return genErr
	}

	otp := &domain.OTP{
		Phone:    phone,
		CodeHash: hashSHA256(code),
		Attempts: 0,
	}

	if err := s.otpRepo.Store(ctx, otp); err != nil {
		span.EndWithError(err)
		return err
	}

	if err := s.smsGateway.SendOTP(ctx, phone, code); err != nil {
		slog.ErrorContext(ctx, "Failed to send OTP SMS", "error", err)
		metrics.Record(ctx, "otp_outcome", metrics.L{metrics.LabelOutcome: "send_failed"})
		smsErr := errors.Internal("Failed to send OTP")
		span.EndWithError(smsErr)
		return smsErr
	}

	metrics.Record(ctx, "otp_outcome", metrics.L{metrics.LabelOutcome: "sent"})
	slog.InfoContext(ctx, "OTP sent", "phone", phone)
	span.End()
	return nil
}

// VerifyOTP verifies an OTP code and returns the customer, tokens, and whether the customer is new
func (s *CustomerAuthService) VerifyOTP(ctx context.Context, phone, code string) (*domain.Customer, *domain.TokenPair, bool, error) {
	ctx, span := telemetry.StartServiceSpan(ctx, "customer_auth", "verify_otp")

	otp, err := s.otpRepo.Get(ctx, phone)
	if err != nil {
		if errors.IsNotFound(err) {
			notFoundErr := errors.New(errors.ErrCodeInvalidToken, "OTP not found or expired")
			span.EndWithError(notFoundErr)
			return nil, nil, false, notFoundErr
		}
		span.EndWithError(err)
		return nil, nil, false, err
	}

	if otp.Attempts >= otpMaxAttempts {
		_ = s.otpRepo.Delete(ctx, phone)
		attemptsErr := errors.New(errors.ErrCodeInvalidToken, "Too many OTP attempts")
		span.EndWithError(attemptsErr)
		return nil, nil, false, attemptsErr
	}

	if incrErr := s.otpRepo.IncrementAttempts(ctx, phone); incrErr != nil {
		slog.WarnContext(ctx, "Failed to increment OTP attempts", "error", incrErr)
	}

	if hashSHA256(code) != otp.CodeHash {
		metrics.Record(ctx, "otp_outcome", metrics.L{metrics.LabelOutcome: "verify_failed"})
		invalidErr := errors.New(errors.ErrCodeInvalidCredentials, "Invalid OTP code")
		span.EndWithError(invalidErr)
		return nil, nil, false, invalidErr
	}

	if delErr := s.otpRepo.Delete(ctx, phone); delErr != nil {
		slog.WarnContext(ctx, "Failed to delete OTP after verification", "error", delErr)
	}

	customer, isNew, err := s.findOrCreateCustomer(ctx, phone)
	if err != nil {
		span.EndWithError(err)
		return nil, nil, false, err
	}

	tokens, err := s.generateTokenPair(customer)
	if err != nil {
		span.EndWithError(err)
		return nil, nil, false, err
	}

	if err := s.storeRefreshToken(ctx, customer.ID, tokens.RefreshToken); err != nil {
		span.EndWithError(err)
		return nil, nil, false, err
	}

	span.SetAttribute("entity.id", customer.ID)
	span.SetAttribute("customer.is_new", isNew)
	metrics.Record(ctx, "otp_outcome", metrics.L{metrics.LabelOutcome: "verified"})
	// session_started fires on OTP verify — i.e. an already-on-site visitor
	// completed authentication. Device + UTM attribution belong on
	// visitor-level signals (site_visitor) and conversion events
	// (payment_completed, customer_first_purchase), not here.
	metrics.Record(ctx, "session_started", metrics.L{
		metrics.LabelCountry:   middleware.GetCountry(ctx),
		metrics.LabelCity:      middleware.GetCity(ctx),
		metrics.LabelIsNewUser: fmt.Sprintf("%t", isNew),
	})
	slog.InfoContext(ctx, "Customer authenticated", "customer_id", customer.ID, "is_new", isNew)
	span.End()
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
	if err != nil {
		// A throttled or timed-out lookup says nothing about the token. Folding
		// it into ErrCodeInvalidToken would answer 401, and the handler clears
		// both auth cookies on that path — a transient blip would end the
		// session outright.
		return nil, nil, err
	}
	if !valid {
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

	// Expire the old token on a grace window rather than deleting it, so an
	// overlapping refresh still succeeds. ValidateToken treats ttl < now as
	// invalid, so expiry is exact regardless of when DynamoDB's TTL sweeper
	// gets to the row.
	graceTTL := time.Now().Add(refreshGracePeriod).Unix()
	if err := s.tokenStore.StoreToken(ctx, customerID, oldHash, graceTTL); err != nil {
		slog.WarnContext(ctx, "Failed to expire old customer refresh token", "error", err)
	}

	return customer, tokens, nil
}

// Logout revokes the presented refresh token and sweeps any grace-window
// predecessor left behind by a recent rotation — but nothing else.
//
// Rotation keeps the pre-rotation token alive for refreshGracePeriod so an
// overlapping refresh still succeeds, and the caller here only ever holds the
// successor. Revoking just that would leave the predecessor usable for up to
// 30 seconds after the customer asked to be logged out. RevokeTokensExpiringBefore
// with a cutoff of "now + refreshGracePeriod" reaches exactly that predecessor
// and nothing on another device: every live session's TTL is the full 7-day
// refresh lifetime, far past this cutoff, so logging out on one device leaves
// the customer's other sessions untouched.
func (s *CustomerAuthService) Logout(ctx context.Context, customerID, refreshToken string) error {
	if err := s.tokenStore.RevokeToken(ctx, customerID, hashSHA256(refreshToken)); err != nil {
		return err
	}

	cutoff := time.Now().Add(refreshGracePeriod).Unix()
	if err := s.tokenStore.RevokeTokensExpiringBefore(ctx, customerID, cutoff); err != nil {
		slog.WarnContext(ctx, "Failed to sweep grace-window predecessor on logout", "error", err)
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
