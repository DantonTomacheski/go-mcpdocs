package api

import (
	"context"
	"time"

	"github.com/dtomacheski/extract-data-go/internal/auth"
	"github.com/gin-gonic/gin"
)

// SetupRouter configures the API routes
func SetupRouter(handler *Handler) *gin.Engine {
	// Create a default gin router with Logger and Recovery middleware
	router := gin.Default()

	// Add CORS middleware
	router.Use(corsMiddleware())
	
	// Add request timeout middleware
	router.Use(timeoutMiddleware(time.Minute))

	// Set up Swagger documentation
	SetupSwagger(router)

	// Health check endpoint
	router.GET("/health", handler.HealthCheck)

	// Authentication routes
	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/login", handler.LoginHandler)
		authRoutes.POST("/register", handler.RegisterHandler)
		authRoutes.POST("/refresh", handler.RefreshTokenHandler)
	}

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Repository endpoints (public)
		v1.GET("/repos/:owner/:repo", handler.GetRepository)
		
		// Repositories query endpoint - novo endpoint semântico (public)
		v1.GET("/repositories", handler.QueryRepositories)
		
		// Search endpoint (legado) (public)
		v1.GET("/search/repos", handler.SearchRepositories)
		
		// Documentation endpoints (hierarchical organization)
		// These endpoints will be protected by JWT authentication
		docs := v1.Group("/docs")
		docs.Use(auth.JWTMiddleware(handler.jwtService)) // Apply JWT middleware to this group
		{
			// Raw documentation files endpoint
			docs.GET("/raw", handler.GetRawDocsFromURL)
			
			// Code snippets endpoint
			docs.GET("/snippets", handler.GetCodeSnippetsFromURL)
			
			// Endpoint to get documentation for a specific repository (now protected)
			// This was previously v1.GET("/repos/:owner/:repo/docs", handler.GetRepositoryDocumentation)
			// Moving it here to be under the authenticated /docs group
			docs.GET("/repos/:owner/:repo", handler.GetRepositoryDocumentation)
		}
		
		// Legacy endpoints (for backward compatibility)
		// These are now also protected if they map to protected new endpoints
		// Note: The middleware is applied to the group, so these might need separate handling 
		// if some should remain public and others protected based on the old path.
		// For simplicity, making them protected for now if they use handlers now under /docs.

		// Redirects to /api/v1/docs/raw (now protected)
		v1.GET("/docs", auth.JWTMiddleware(handler.jwtService), handler.GetRawDocsFromURL)      
		// Redirects to /api/v1/docs/snippets (now protected)
		v1.GET("/snippets", auth.JWTMiddleware(handler.jwtService), handler.GetCodeSnippetsFromURL) 
	}

	return router
}

// corsMiddleware handles CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// timeoutMiddleware adds a timeout to the request context
func timeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for streaming requests
		if c.GetHeader("Accept") == "text/event-stream" {
			c.Next()
			return
		}

		// Create a new context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()
	}
}
