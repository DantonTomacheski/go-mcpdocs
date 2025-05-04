package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/dtomacheski/extract-data-go/internal/models"
	"github.com/dtomacheski/extract-data-go/internal/processor"
	"github.com/dtomacheski/extract-data-go/internal/utils"
	"github.com/gin-gonic/gin"
)

// GetProcessedDocsFromURL handles fetching and processing documentation from a GitHub repository URL
// This is an enhanced version that extracts code snippets
func (h *Handler) GetProcessedDocsFromURL(c *gin.Context) {
	// Get repository URL from query parameters
	repoURL := c.Query("repo")
	if repoURL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Missing 'repo' parameter",
			Status:  http.StatusBadRequest,
		})
		return
	}

	// Get limit parameter (maximum number of snippets to return)
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // Maximum 100 snippets per request
	}

	// Get page parameter for pagination
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	// Extract branch/tag if specified
	ref := c.Query("ref")

	// Extract owner and repo from the URL
	owner, repo, err := utils.ExtractOwnerAndRepo(repoURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid GitHub repository URL: " + err.Error(),
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
		documentation, err = h.GitHubClient.GetRepositoryDocumentation(ctx, owner, repo, h.WorkerPoolSize)
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

	// Get repository info to build URLs
	repoInfo, err := h.GitHubClient.GetRepository(ctx, owner, repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "github_api_error",
			Message: "Failed to get repository information: " + err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	// Create document processor
	docProcessor := processor.NewDocumentProcessor()

	// Process the documentation to extract code snippets
	processedResponse := docProcessor.ExtractSnippets(documentation, repoInfo.FullName, repoInfo.HTMLURL)

	// Format response in enhanced style
	if c.Query("format") == "enhanced" {
		formattedString, err := formatEnhancedStyle(processedResponse)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to format response: " + err.Error(),
				Status:  http.StatusInternalServerError,
			})
			return
		}

		// Check if output should be saved to file for testing
		if c.Query("output") == "file" {
			outputFilename := "output_enhanced.txt"
			err := os.WriteFile(outputFilename, []byte(formattedString), 0644)
			if err != nil {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:   "file_error",
					Message: "Failed to write output file: " + err.Error(),
					Status:  http.StatusInternalServerError,
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"status":  http.StatusOK,
				"message": "Output successfully saved to " + outputFilename,
			})
		} else {
			// Default behavior: return as text/plain
			c.Header("Content-Type", "text/plain; charset=utf-8")
			c.String(http.StatusOK, formattedString)
		}
		return // Exit after handling context7 format
	}

	// Return the processed documentation
	c.JSON(http.StatusOK, models.SuccessResponse{
		Status:  http.StatusOK,
		Message: "Documentation processed successfully",
		Data:    processedResponse,
	})
}

// formatEnhancedStyle formats snippets into a structured text format.
func formatEnhancedStyle(response models.DocumentationResponse) (string, error) {
	var sb strings.Builder

	sb.WriteString("Repository: " + response.RepositoryName + "\n")
	sb.WriteString("Total Files: " + fmt.Sprintf("%d", response.TotalFiles) + "\n")
	sb.WriteString("Total Snippets: " + fmt.Sprintf("%d", response.TotalSnippets) + "\n\n")

	separator := "----------------------------------------\n\n"

	for i, snippet := range response.Snippets {
		sb.WriteString("TITLE: " + snippet.Title + "\n")
		sb.WriteString("DESCRIPTION: " + snippet.Description + "\n")
		sb.WriteString("SOURCE: " + snippet.Source + "\n")
		sb.WriteString("LANGUAGE: " + snippet.Language + "\n")
		sb.WriteString("CODE:\n```\n" + snippet.Code + "\n```\n")
		
		// Add separator only if it's not the last snippet
		if i < len(response.Snippets)-1 {
			sb.WriteString("\n" + separator) // Add separator
		}
	}
	
	return sb.String(), nil
}
