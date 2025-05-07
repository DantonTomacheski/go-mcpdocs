package auth

import (
	"errors"
	"time"

	"github.com/dtomacheski/extract-data-go/config"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

// JWTService handles JWT token operations
type JWTService struct {
	config *config.Config
}

// NewJWTService creates a new JWT service
func NewJWTService(config *config.Config) *JWTService {
	return &JWTService{
		config: config,
	}
}

// GenerateAccessToken creates a new JWT access token for the given user
func (s *JWTService) GenerateAccessToken(user *User) (string, error) {
	// Set expiration time based on configured duration
	expirationTime := time.Now().Add(s.config.JWTAccessDuration)

	// Create the claims
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.config.JWTIssuer,
			Subject:   user.ID,
		},
	}

	// Generate the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// Sign the token with the secret key
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateRefreshToken creates a new JWT refresh token
func (s *JWTService) GenerateRefreshToken(user *User) (string, error) {
	// Set expiration time for refresh token
	expirationTime := time.Now().Add(s.config.JWTRefreshDuration)

	// Create minimal claims for refresh token (just enough to identify the user)
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Issuer:    s.config.JWTIssuer,
		Subject:   user.ID,
	}

	// Generate the refresh token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// Sign the token with the secret key
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken validates a token and returns its claims
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate the alg is what we expect
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return []byte(s.config.JWTSecret), nil
		},
	)

	if err != nil {
		// Check if the error is because the token is expired
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// Validate the token
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Extract the claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (s *JWTService) ValidateRefreshToken(tokenString string) (string, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate the alg is what we expect
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return []byte(s.config.JWTSecret), nil
		},
	)

	if err != nil {
		// Check if the error is because the token is expired
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}

	// Validate the token
	if !token.Valid {
		return "", ErrInvalidToken
	}

	// Extract the claims
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return "", ErrInvalidToken
	}

	// Return the user ID from the subject claim
	return claims.Subject, nil
}

// GetTokenExpiration returns the expiration time in seconds
func (s *JWTService) GetTokenExpiration() int64 {
	return int64(s.config.JWTAccessDuration.Seconds())
}
