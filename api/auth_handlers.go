package api

import (
	"net/http"

	"github.com/dtomacheski/extract-data-go/internal/auth"
	"github.com/gin-gonic/gin"
)

// LoginHandler handles the login request
func (h *Handler) LoginHandler(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Authenticate the user
	user, err := h.userStore.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate tokens
	accessToken, err := h.jwtService.GenerateAccessToken(user)
	if err != nil {
		h.Logger.Printf("Error generating access token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user)
	if err != nil {
		h.Logger.Printf("Error generating refresh token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Return the tokens
	c.JSON(http.StatusOK, auth.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.jwtService.GetTokenExpiration(),
	})
}

// RegisterHandler handles the registration request
func (h *Handler) RegisterHandler(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Create the user
	user, err := h.userStore.CreateUser(req.Username, req.Email, req.Password, "user") // Default role is "user"
	if err != nil {
		var statusCode int
		var message string

		switch err {
		case auth.ErrUserAlreadyExists:
			statusCode = http.StatusConflict
			message = "Username already exists"
		default:
			statusCode = http.StatusInternalServerError
			message = "Failed to create user"
			h.Logger.Printf("Error creating user: %v", err)
		}

		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	// Generate tokens
	accessToken, err := h.jwtService.GenerateAccessToken(user)
	if err != nil {
		h.Logger.Printf("Error generating access token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user)
	if err != nil {
		h.Logger.Printf("Error generating refresh token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Return the tokens
	c.JSON(http.StatusCreated, auth.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.jwtService.GetTokenExpiration(),
	})
}

// RefreshTokenHandler handles the token refresh request
func (h *Handler) RefreshTokenHandler(c *gin.Context) {
	var req auth.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Validate the refresh token
	userID, err := h.jwtService.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		var statusCode int
		var message string

		switch err {
		case auth.ErrExpiredToken:
			statusCode = http.StatusUnauthorized
			message = "Refresh token has expired"
		case auth.ErrInvalidToken:
			statusCode = http.StatusUnauthorized
			message = "Invalid refresh token"
		default:
			statusCode = http.StatusInternalServerError
			message = "Failed to validate refresh token"
			h.Logger.Printf("Error validating refresh token: %v", err)
		}

		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	// Get the user from the store (in a real application, use the user ID to fetch from database)
	// This is a simplification for the example
	user, err := h.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Generate a new access token
	accessToken, err := h.jwtService.GenerateAccessToken(user)
	if err != nil {
		h.Logger.Printf("Error generating access token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	// Return the new access token
	c.JSON(http.StatusOK, auth.TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   h.jwtService.GetTokenExpiration(),
	})
}

// GetUserByID finds a user by ID (simplified implementation)
func (h *Handler) GetUserByID(userID string) (*auth.User, error) {
	// In a real application, this would query a database
	// For this example, we'll iterate through all users to find one with the matching ID
	for _, username := range h.userStore.GetAllUsernames() {
		user, err := h.userStore.GetUser(username)
		if err != nil {
			continue
		}
		if user.ID == userID {
			return user, nil
		}
	}
	return nil, auth.ErrUserNotFound
}
