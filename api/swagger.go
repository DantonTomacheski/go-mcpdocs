package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SetupSwagger adds Swagger documentation routes to the router
func SetupSwagger(router *gin.Engine) {
	// Check if Swagger is enabled
	if os.Getenv("SWAGGER_ENABLED") != "true" {
		return
	}
	// Load the HTML template
	router.LoadHTMLGlob("templates/*.html")
	
	// Serve the Swagger YAML file
	router.GET("/swagger.yaml", serveSwaggerFile)
	
	// Serve the custom Swagger UI
	router.GET("/swagger", func(c *gin.Context) {
		c.HTML(http.StatusOK, "swagger-ui.html", gin.H{})
	})
}

// serveSwaggerFile serves the swagger.yaml file
func serveSwaggerFile(c *gin.Context) {
	// Get the content type based on file extension
	contentType := "application/yaml"
	if strings.HasSuffix(c.Request.URL.Path, ".json") {
		contentType = "application/json"
	}
	
	// Set the content type header
	c.Header("Content-Type", contentType)
	c.Header("Access-Control-Allow-Origin", "*")
	
	// Serve the file
	c.File(filepath.Join(".", "swagger.yaml"))
}
