package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dtomacheski/extract-data-go/internal/models"
	"github.com/dtomacheski/extract-data-go/internal/utils"
	"github.com/gin-gonic/gin"
)

// GetRawDocsFromURL handles fetching raw documentation files from a GitHub repository URL
func (h *Handler) GetRawDocsFromURL(c *gin.Context) {
	repoURL := c.Query("repo")
	if repoURL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Missing 'repo' parameter",
			Status:  http.StatusBadRequest,
		})
		return
	}

	// Extract branch/tag if specified
	ref := c.Query("ref")

	// Extract owner and repo from the URL
	owner, repo, err := utils.ExtractOwnerAndRepo(repoURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: fmt.Sprintf("Invalid GitHub repository URL: %v", err),
			Status:  http.StatusBadRequest,
		})
		return
	}

	// Set up cancellation context
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Set up cancellation when client disconnects
	go func() {
		<-c.Request.Context().Done()
		cancel()
	}()

	// Get documentation
	var documentation []models.Documentation
	if ref != "" {
		// If ref is provided, we would need to extend our client to support custom refs
		// For now this is a placeholder for future implementation
		c.JSON(http.StatusNotImplemented, models.ErrorResponse{
			Error:   "not_implemented",
			Message: "Custom branch/tag reference is not yet supported",
			Status:  http.StatusNotImplemented,
		})
		return
	} else {
		var err error
		documentation, err = h.GitHubClient.GetRepositoryDocumentation(ctx, owner, repo, "", h.WorkerPoolSize)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "repository not found" {
				statusCode = http.StatusNotFound
			} else if err.Error() == "unauthorized: invalid GitHub token" {
				statusCode = http.StatusUnauthorized
			} else if err.Error() == "rate limit exceeded or access denied" {
				statusCode = http.StatusTooManyRequests
			} else if err.Error() == "no documentation found for repository" {
				statusCode = http.StatusNotFound
			}

			c.JSON(statusCode, models.ErrorResponse{
				Error:   "github_api_error",
				Message: err.Error(),
				Status:  statusCode,
			})
			return
		}
	}

	// Return the documentation
	c.JSON(http.StatusOK, models.SuccessResponse{
		Status:  http.StatusOK,
		Message: "Documentation retrieved successfully",
		Data:    documentation,
	})
}

// isDocFile checks if a file is a documentation file
func isDocFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".md" || ext == ".rst" || ext == ".txt" ||
		ext == ".mdx" || ext == ".asciidoc" || ext == ".adoc"
}
