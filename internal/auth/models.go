package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents a registered user in the system
type User struct {
	ID       string    `json:"id" bson:"_id"`
	Username string    `json:"username" bson:"username"`
	Email    string    `json:"email" bson:"email"`
	Password string    `json:"-" bson:"password"` // Password is not exposed in JSON responses
	Role     string    `json:"role" bson:"role"`
	Created  time.Time `json:"created" bson:"created"`
}

// Claims represents the JWT claims structure
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents the registration request payload
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// TokenResponse represents the JWT token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // Seconds until expiration
}

// RefreshRequest is the payload for refreshing an access token
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
