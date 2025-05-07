package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// The authentication middleware key in the context
const (
	UserKey = "user"
	RoleKey = "role"
)

// JWTMiddleware creates a Gin middleware to validate JWT tokens
func JWTMiddleware(jwtService *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			c.Abort()
			return
		}

		// Check if the header has the Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		// Extract the token
		tokenString := parts[1]

		// Validate the token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			var status int
			var message string

			switch err {
			case ErrExpiredToken:
				status = http.StatusUnauthorized
				message = "token has expired"
			case ErrInvalidToken:
				status = http.StatusUnauthorized
				message = "invalid token"
			default:
				status = http.StatusInternalServerError
				message = "failed to validate token"
			}

			c.JSON(status, gin.H{"error": message})
			c.Abort()
			return
		}

		// Set the user and role in the context
		c.Set(UserKey, claims.Username)
		c.Set(RoleKey, claims.Role)

		c.Next()
	}
}

// RoleMiddleware creates a middleware to check if the user has the required role
func RoleMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user role from the context
		role, exists := c.Get(RoleKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user role not found in context"})
			c.Abort()
			return
		}

		userRole := role.(string)

		// Check if the user has one of the required roles
		hasRole := false
		for _, r := range requiredRoles {
			if r == userRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUser extracts the current user from the Gin context
func GetCurrentUser(c *gin.Context) string {
	user, exists := c.Get(UserKey)
	if !exists {
		return ""
	}
	return user.(string)
}

// GetCurrentRole extracts the current user role from the Gin context
func GetCurrentRole(c *gin.Context) string {
	role, exists := c.Get(RoleKey)
	if !exists {
		return ""
	}
	return role.(string)
}
