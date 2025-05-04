package api

import (
	"context"
	"time"

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

	// Health check endpoint
	router.GET("/health", handler.HealthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Repository endpoints
		v1.GET("/repos/:owner/:repo", handler.GetRepository)
		v1.GET("/repos/:owner/:repo/docs", handler.GetRepositoryDocumentation)
		
		// Search endpoint
		v1.GET("/search/repos", handler.SearchRepositories)
		
		// Direct URL endpoint (new approach)
		v1.GET("/docs", handler.GetDocsFromURL)
		
		// Enhanced documentation endpoint (Context7-like)
		v1.GET("/snippets", handler.GetProcessedDocsFromURL)
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
