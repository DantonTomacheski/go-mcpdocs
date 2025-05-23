package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dtomacheski/extract-data-go/internal/auth"
	"github.com/dtomacheski/extract-data-go/internal/cache"
	"github.com/dtomacheski/extract-data-go/internal/github"
	"github.com/dtomacheski/extract-data-go/internal/models"
	"github.com/dtomacheski/extract-data-go/internal/repository"
	"github.com/gin-gonic/gin"
)

// Handler contains the handlers for the API
type Handler struct {
	GitHubClient       *github.Client
	WorkerPoolSize     int
	DocumentRepository *repository.DocumentRepository
	Logger             *log.Logger
	Cache              cache.Cache
	KeyBuilder         *cache.KeyBuilder
	MinDaysBetweenRefreshes int // Minimum days required between documentation refreshes
	
	// Authentication services
	userStore          *auth.UserStore
	jwtService         *auth.JWTService
}

// NewHandler creates a new API handler
func NewHandler(client *github.Client, docRepo *repository.DocumentRepository, cacheClient cache.Cache, logger *log.Logger, workerPoolSize int, userStore *auth.UserStore, jwtService *auth.JWTService) *Handler {
	// Create a key builder with a prefix for the application
	keyBuilder := cache.NewKeyBuilder("mcpdocs")

	// Create demo admin user if running in development mode
	// In production, you would handle user creation differently
	// This logic should ideally be outside the handler initialization or managed by the injected UserStore.
	// For now, let's keep it but use the injected userStore.
	_, err := userStore.CreateUser("admin", "admin@example.com", "admin123", "admin")
	if err != nil && err != auth.ErrUserAlreadyExists {
		logger.Printf("Warning: Failed to create admin user: %v", err)
	}

	return &Handler{
		GitHubClient:       client,
		WorkerPoolSize:     workerPoolSize,
		DocumentRepository: docRepo,
		Logger:             logger,
		Cache:              cacheClient,
		KeyBuilder:         keyBuilder,
		MinDaysBetweenRefreshes: 3, // Default: minimum 3 days between refreshes
		userStore:          userStore,  // Use injected userStore
		jwtService:         jwtService, // Use injected jwtService
	}
}

// GetRepository handles fetching repository information
// Also automatically fetches documentation for the repository
func (h *Handler) GetRepository(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	// Optional parameters
	ref := c.Query("ref")
	tag := c.Query("tag") // Alias para ref, mantido para compatibilidade
	if tag != "" && ref == "" {
		ref = tag // Prioriza ref, mas aceita tag para compatibilidade
	}

	// Novos parâmetros para suportar a consolidação
	docsOnly := c.Query("docs_only") == "true"

	if owner == "" || repo == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Owner and repository name are required",
			Status:  http.StatusBadRequest,
		})
		return
	}

	// Check if the client explicitly wants to skip documentation
	skipDocs := c.Query("skip_docs") == "true"

	// Skip docs não pode ser usado em conjunto com docs_only
	if docsOnly && skipDocs {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Cannot specify both 'docs_only' and 'skip_docs' parameters",
			Status:  http.StatusBadRequest,
		})
		return
	}

	// Generate cache key for repository information
	cacheKey := h.KeyBuilder.RepositoryKey(owner, repo)
	var repository *models.Repository
	var fromCache bool

	// Check if we need to respect the refresh rate limit
	forceRefresh := c.Query("force_refresh") == "true"
	// Se qualquer parâmetro que identifique versão específica for fornecido, ignoramos a validação
	// Isso inclui ref, tag ou qualquer outro parâmetro de consulta futuro
	hanySiteFilter := len(c.Request.URL.Query()) > 0 
	
	// Permitir refresh se forceRefresh for true OU 
	// se houver qualquer parâmetro de consulta (indicando uma versão específica)
	if !forceRefresh && !hanySiteFilter && h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
		// Debug para verificar os parâmetros recebidos
		h.Logger.Printf("URL query parameters: %v", c.Request.URL.Query())
		
		// Apenas aplicar a verificação de cache quando não houver parâmetros específicos
		canRefresh, nextAllowedTime, err := h.DocumentRepository.CanRefreshRepository(
			c.Request.Context(), owner, repo, h.MinDaysBetweenRefreshes)
		
		if err != nil && !canRefresh {
			// Calculate days remaining until next refresh
			daysRemaining := int(time.Until(nextAllowedTime).Hours() / 24) + 1
			
			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error:   "refresh_rate_limited",
				Message: fmt.Sprintf("Too early to refresh the project. Last update was %d days ago. Minimum %d days required between updates.", 
					h.MinDaysBetweenRefreshes - daysRemaining, h.MinDaysBetweenRefreshes),
				Status:  http.StatusTooManyRequests,
			})
			return
		}
	}

	// Try to get repository from cache first if cache is enabled
	if h.Cache != nil && h.Cache.IsEnabled() {
		h.Logger.Printf("Checking cache for repository: %s/%s", owner, repo)
		cacheErr := h.Cache.Get(c.Request.Context(), cacheKey, &repository)
		
		if cacheErr == nil {
			// Cache hit
			h.Logger.Printf("Cache hit for repository: %s/%s", owner, repo)
			fromCache = true
		} else if cacheErr != cache.ErrCacheMiss {
			// Cache error (not a miss)
			h.Logger.Printf("Cache error for repository: %v", cacheErr)
		}
	}

	// If not in cache, fetch from GitHub
	var err error
	if !fromCache {
		// Get basic repository information
		repository, err = h.GitHubClient.GetRepository(c.Request.Context(), owner, repo)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "repository not found" {
				statusCode = http.StatusNotFound
			} else if err.Error() == "unauthorized: invalid GitHub token" {
				statusCode = http.StatusUnauthorized
			} else if err.Error() == "rate limit exceeded or access denied" {
				statusCode = http.StatusTooManyRequests
			}

			c.JSON(statusCode, models.ErrorResponse{
				Error:   "github_api_error",
				Message: err.Error(),
				Status:  statusCode,
			})
			return
		}

		// Cache the repository information
		if h.Cache != nil && h.Cache.IsEnabled() {
			h.Logger.Printf("Caching repository information for %s/%s", owner, repo)
			if cacheErr := h.Cache.Set(c.Request.Context(), cacheKey, repository); cacheErr != nil {
				h.Logger.Printf("Failed to cache repository information: %v", cacheErr)
			}
		}
	}

	// If skipDocs is true, return only the repository information
	if skipDocs {
		responseMessage := "Repository retrieved successfully (without docs)"
		if fromCache {
			responseMessage += " (from cache)"
		}

		c.JSON(http.StatusOK, models.SuccessResponse{
			Status:  http.StatusOK,
			Message: responseMessage,
			Data:    repository,
		})
		return
	}

	// Set up a context that can be canceled when the client disconnects
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Configure cancellation when client disconnects
	go func() {
		<-c.Request.Context().Done()
		cancel()
	}()

	// Attempt to retrieve documentation from cache first (assuming default branch/no tag here)
	docCacheKey := h.KeyBuilder.RepositoryDocumentationKey(owner, repo, "") // Pass empty tag
	var documentationItems []models.Documentation
	var docErr error
	var docsFromCache bool

	// Try to get documentation from cache first if cache is enabled
	if h.Cache != nil && h.Cache.IsEnabled() {
		h.Logger.Printf("Checking cache for repository documentation: %s/%s", owner, repo)
		cacheErr := h.Cache.Get(ctx, docCacheKey, &documentationItems)
		
		if cacheErr == nil && len(documentationItems) > 0 {
			// Cache hit
			h.Logger.Printf("Cache hit for repository documentation: %s/%s", owner, repo)
			docsFromCache = true
		} else if cacheErr != cache.ErrCacheMiss {
			// Cache error (not a miss)
			h.Logger.Printf("Cache error for repository documentation: %v", cacheErr)
		}
	}

	// If not in cache, fetch documentation from GitHub
	if !docsFromCache {
		// Fetch documentation from the repository in the background
		h.Logger.Printf("Auto-fetching documentation for %s/%s", owner, repo)

		// First, get repository details to determine the default branch for documentation fetching
		var defaultBranchToUse string
		repoDetails, detailsErr := h.GitHubClient.GetRepository(ctx, owner, repo)
		if detailsErr != nil {
			h.Logger.Printf("Error fetching repository details for %s/%s to get default branch (needed for docs auto-fetch): %v", owner, repo, detailsErr)
			// If we can't get the default branch, GetRepositoryDocumentation will likely fail if specificRef is also empty.
			// Proceeding with empty defaultBranchToUse; client logic will handle it.
		} else if repoDetails != nil {
			defaultBranchToUse = repoDetails.DefaultBranch
		}

		documentationItems, docErr = h.GitHubClient.GetRepositoryDocumentation(ctx, owner, repo, defaultBranchToUse, "", h.WorkerPoolSize)

		// If we have documentation and no error, cache it and store in MongoDB if configured
		if docErr == nil && len(documentationItems) > 0 {
			// Cache the documentation
			if h.Cache != nil && h.Cache.IsEnabled() {
				h.Logger.Printf("Caching repository documentation for %s/%s", owner, repo)
				if cacheErr := h.Cache.Set(ctx, docCacheKey, documentationItems); cacheErr != nil {
					h.Logger.Printf("Failed to cache repository documentation: %v", cacheErr)
				}
			}

			// Store in MongoDB if enabled
			if h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
				h.Logger.Printf("Processing and storing documentation in TXT format for %s/%s", owner, repo)
				if storeErr := h.DocumentRepository.StoreDocumentation(ctx, documentationItems); storeErr != nil {
					h.Logger.Printf("Failed to store processed documentation in MongoDB: %v", storeErr)
				} else {
					h.Logger.Printf("Successfully processed and stored documentation in MongoDB for %s/%s", owner, repo)
				}
			}
		}
	}

	// Prepara a resposta com informações do repositório e documentação
	response := map[string]interface{}{
		"repository": repository,
	}

	// Adiciona a documentação à resposta se estiver disponível
	if docErr == nil && len(documentationItems) > 0 {
		response["documentation"] = map[string]interface{}{
			"message":            fmt.Sprintf("Successfully retrieved %d documentation files.", len(documentationItems)),
			"processed_files":    len(documentationItems),
			"documentation_items": documentationItems,
		}
	} else if docErr != nil {
		// Adiciona informação de erro da documentação, mas não falha a resposta principal
		response["documentation"] = map[string]interface{}{
			"error":   "Could not retrieve documentation",
			"message": docErr.Error(),
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Status:  http.StatusOK,
		Message: "Repository retrieved successfully with documentation",
		Data:    response,
	})
}

// GetRepositoryDocumentation handles fetching documentation for a repository
func (h *Handler) GetRepositoryDocumentation(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	tag := c.Query("tag") // Optional tag or branch name
	forceRefresh := c.Query("force_refresh") == "true" // Optional force refresh parameter

	if owner == "" || repo == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Owner and repository name are required",
			Status:  http.StatusBadRequest,
		})
		return
	}
	
	// Check if we need to respect the refresh rate limit
	// Se qualquer parâmetro que identifique versão específica for fornecido, ignoramos a validação
	// Isso inclui tag ou qualquer outro parâmetro de consulta futuro
	hanySiteFilter := len(c.Request.URL.Query()) > 0
	
	// Permitir refresh se forceRefresh for true OU
	// se houver qualquer parâmetro de consulta (indicando uma versão específica)
	if !forceRefresh && !hanySiteFilter && h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
		// Debug para verificar os parâmetros recebidos
		h.Logger.Printf("URL query parameters: %v", c.Request.URL.Query())
		
		// Apenas aplicar a verificação de cache quando não houver parâmetros específicos
		canRefresh, nextAllowedTime, err := h.DocumentRepository.CanRefreshRepository(
			c.Request.Context(), owner, repo, h.MinDaysBetweenRefreshes)
		
		if err != nil && !canRefresh {
			// Calculate days remaining until next refresh
			daysRemaining := int(time.Until(nextAllowedTime).Hours() / 24) + 1
			
			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error:   "refresh_rate_limited",
				Message: fmt.Sprintf("Too early to refresh the project. Last update was %d days ago. Minimum %d days required between updates.", 
					h.MinDaysBetweenRefreshes - daysRemaining, h.MinDaysBetweenRefreshes),
				Status:  http.StatusTooManyRequests,
			})
			return
		}
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Configure context cancellation when the client disconnects
	go func() {
		<-c.Request.Context().Done()
		cancel()
	}()

	// IMPLEMENTATION OF TWO-LAYER CACHING STRATEGY
	// 1. First, check for metadata index in cache
	var metadataIndex models.RepositoryDocumentationIndex
	var documentationItems []models.Documentation
	var fromCache bool
	var fromFragmentedCache bool
	var metadataCacheKey string
	
	if h.Cache != nil && h.Cache.IsEnabled() {
		// Generate cache key for repository documentation metadata
		metadataCacheKey = h.KeyBuilder.RepositoryDocumentationMetadataKey(owner, repo, tag)
		
		h.Logger.Printf("Checking metadata cache for repository documentation: %s/%s (ref: %s)", owner, repo, tag)
		cacheErr := h.Cache.Get(ctx, metadataCacheKey, &metadataIndex)
		
		if cacheErr == nil {
			// Metadata cache hit - now we need to fetch document contents
			h.Logger.Printf("Metadata cache hit for repository documentation: %s/%s (ref: %s)", owner, repo, tag)
			
			// Prepare to fetch all document contents
			documentationItems = make([]models.Documentation, 0, metadataIndex.DocumentCount)
			missingDocuments := make([]models.DocumentMetadata, 0)
			
			// For each document in the metadata index, try to fetch it from content cache
			for _, docMeta := range metadataIndex.Documents {
				// Generate content cache key
				contentCacheKey := h.KeyBuilder.DocumentContentKey(owner, repo, tag, docMeta.SHA)
				
				// Try to get document content from cache
				var docContent models.Documentation
				contentCacheErr := h.Cache.Get(ctx, contentCacheKey, &docContent)
				
				if contentCacheErr == nil {
					// Cache hit for this document
					documentationItems = append(documentationItems, docContent)
				} else {
					// Cache miss for this document - add to missing documents list
					missingDocuments = append(missingDocuments, docMeta)
				}
			}
			
			// If we have all documents from cache, mark as complete cache hit
			if len(missingDocuments) == 0 {
				h.Logger.Printf("Complete content cache hit for all %d documents", len(documentationItems))
				fromCache = true
				fromFragmentedCache = true
			} else {
				// Partial cache hit - need to fetch missing documents from GitHub
				h.Logger.Printf("Partial cache hit: %d/%d documents in cache, fetching %d missing documents", 
					len(documentationItems), metadataIndex.DocumentCount, len(missingDocuments))
				
				// Fetch missing documents from GitHub
				missingPaths := make([]string, 0, len(missingDocuments))
				for _, doc := range missingDocuments {
					missingPaths = append(missingPaths, doc.Path)
				}
				
				// Here we need to fetch specific documents by path
				// If the GitHub client doesn't have a method to fetch specific documents,
				// we'll fetch all documents and filter them
				var defaultBranchForGHClient string
				if tag == "" { // If no specific tag/ref is requested, we need the default branch
					repoInfo, repoErr := h.GitHubClient.GetRepository(ctx, owner, repo)
					if repoErr != nil {
						h.Logger.Printf("Failed to fetch repository info for %s/%s to get default branch: %v", owner, repo, repoErr)
						c.JSON(http.StatusInternalServerError, models.ErrorResponse{
							Error:   "github_error",
							Message: fmt.Sprintf("Failed to retrieve repository details for default branch: %s", repoErr.Error()),
							Status:  http.StatusInternalServerError,
						})
						return
					}
					defaultBranchForGHClient = repoInfo.DefaultBranch
					if defaultBranchForGHClient == "" {
						h.Logger.Printf("Repository %s/%s has an empty default branch string", owner, repo)
						c.JSON(http.StatusInternalServerError, models.ErrorResponse{
							Error:   "repository_error",
							Message: "Repository's default branch is not set or is empty.",
							Status:  http.StatusInternalServerError,
						})
						return
					}
				}
		
				allDocs, err := h.GitHubClient.GetRepositoryDocumentation(ctx, owner, repo, defaultBranchForGHClient, tag, h.WorkerPoolSize)
				if err != nil {
					h.Logger.Printf("Failed to fetch missing documents: %v", err)
					// Continue with partial results if we have some
					if len(documentationItems) == 0 {
						// No documents at all, return error
						statusCode := getStatusCodeFromError(err)
						c.JSON(statusCode, models.ErrorResponse{
							Error:   "github_api_error",
							Message: err.Error(),
							Status:  statusCode,
						})
						return
					}
				} else {
					// Filter only the documents we need
					for _, doc := range allDocs {
						for _, path := range missingPaths {
							if doc.Path == path {
								documentationItems = append(documentationItems, doc)
								
								// Cache this document content
								if h.Cache.IsEnabled() {
									contentCacheKey := h.KeyBuilder.DocumentContentKey(owner, repo, tag, doc.SHA)
									h.Cache.Set(ctx, contentCacheKey, doc)
								}
								break
							}
						}
					}
				}
				fromFragmentedCache = true
			}
		} else if cacheErr != cache.ErrCacheMiss {
			// Cache error (not a miss)
			h.Logger.Printf("Cache error for repository documentation metadata: %v", cacheErr)
		}
	}

	// If nothing in metadata cache, fall back to fetching everything from GitHub
	var err error
	if !fromCache && !fromFragmentedCache { // This condition might need refinement based on how fromFragmentedCache is set for full misses
		h.Logger.Printf("Fetching repository documentation from GitHub for: %s/%s (ref: %s, forceRefresh: %t)", owner, repo, tag, forceRefresh)

		var defaultBranchForGHClient string
		if tag == "" { // If no specific tag/ref is requested, we need the default branch
			repoInfo, repoErr := h.GitHubClient.GetRepository(ctx, owner, repo)
			if repoErr != nil {
				h.Logger.Printf("Failed to fetch repository info for %s/%s to get default branch: %v", owner, repo, repoErr)
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:   "github_error",
					Message: fmt.Sprintf("Failed to retrieve repository details for default branch: %s", repoErr.Error()),
					Status:  http.StatusInternalServerError,
				})
				return
			}
			defaultBranchForGHClient = repoInfo.DefaultBranch
			if defaultBranchForGHClient == "" {
				h.Logger.Printf("Repository %s/%s has an empty default branch string", owner, repo)
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error:   "repository_error",
					Message: "Repository's default branch is not set or is empty.",
					Status:  http.StatusInternalServerError,
				})
				return
			}
		}
		
		documentationItems, err = h.GitHubClient.GetRepositoryDocumentation(ctx, owner, repo, defaultBranchForGHClient, tag, h.WorkerPoolSize)
		if err != nil {
			h.Logger.Printf("Error fetching repository documentation for %s/%s (ref: %s): %v", owner, repo, tag, err)
			statusCode := getStatusCodeFromError(err)
			c.JSON(statusCode, models.ErrorResponse{
				Error:   "github_api_error",
				Message: err.Error(),
				Status:  statusCode,
			})
			return
		}
		
		// If we have valid results, create and cache metadata index and document contents
		if h.Cache != nil && h.Cache.IsEnabled() && len(documentationItems) > 0 {
			h.Logger.Printf("Caching metadata and content for %d documents", len(documentationItems))
			
			// Create metadata index
			metadataIndex = models.RepositoryDocumentationIndex{
				RepositoryOwner: owner,
				RepositoryName:  repo,
				RepositoryRef:   tag,
				DocumentCount:   len(documentationItems),
				CreatedAt:       time.Now(),
				Documents:       make([]models.DocumentMetadata, 0, len(documentationItems)),
			}
			
			// Cache each document content separately
			for _, doc := range documentationItems {
				// Add to metadata index
				metadataIndex.Documents = append(metadataIndex.Documents, models.DocumentMetadata{
					Path:      doc.Path,
					Size:      doc.Size,
					SHA:       doc.SHA,
					CreatedAt: time.Now(),
				})
				
				// Cache document content
				contentCacheKey := h.KeyBuilder.DocumentContentKey(owner, repo, tag, doc.SHA)
				if cacheErr := h.Cache.Set(ctx, contentCacheKey, doc); cacheErr != nil {
					h.Logger.Printf("Failed to cache document content for %s: %v", doc.Path, cacheErr)
				}
			}
			
			// Cache metadata index
			metadataCacheKey = h.KeyBuilder.RepositoryDocumentationMetadataKey(owner, repo, tag)
			if cacheErr := h.Cache.Set(ctx, metadataCacheKey, metadataIndex); cacheErr != nil {
				h.Logger.Printf("Failed to cache repository documentation metadata: %v", cacheErr)
			}
		}
	}

	// Process and store documentation in MongoDB in TXT format
	if (!fromCache || !fromFragmentedCache) && h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
		h.Logger.Printf("Processing and storing documentation in TXT format for %s/%s", owner, repo)
		if err := h.DocumentRepository.StoreDocumentation(ctx, documentationItems); err != nil {
			h.Logger.Printf("Failed to store processed documentation in MongoDB: %v", err)
			// This is not a critical error, we can still return the documentation to the client
		} else {
			h.Logger.Printf("Successfully processed and stored documentation in MongoDB for %s/%s", owner, repo)
		}
	}

	// Construct the response using RepositoryDocsResponse
	response := models.RepositoryDocsResponse{
		Status:             http.StatusOK,
		Message:            fmt.Sprintf("Successfully retrieved %d documentation files.", len(documentationItems)),
		RepositoryOwner:    owner,
		RepositoryName:     repo,
		RepositoryRef:      tag,
		ProcessedFilesCount: len(documentationItems),
		DocumentationItems: documentationItems,
	}

	// Add cache information to the response
	if fromCache {
		response.Message += " (from cache)"
	} else if fromFragmentedCache {
		response.Message += " (from fragmented cache)"
	}

	c.JSON(http.StatusOK, response)
}

// Helper function to get HTTP status code from error
func getStatusCodeFromError(err error) int {
	statusCode := http.StatusInternalServerError
	if strings.Contains(err.Error(), "repository not found") {
		statusCode = http.StatusNotFound
	} else if strings.Contains(err.Error(), "invalid GitHub token") {
		statusCode = http.StatusUnauthorized
	} else if strings.Contains(err.Error(), "rate limit exceeded") {
		statusCode = http.StatusTooManyRequests
	} else if strings.Contains(err.Error(), "no documentation files found") || strings.Contains(err.Error(), "no documentation content could be successfully retrieved") {
		statusCode = http.StatusNotFound
	}
	return statusCode
}

// SearchRepositories handles searching for repositories
func (h *Handler) SearchRepositories(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Query parameter 'q' is required",
			Status:  http.StatusBadRequest,
		})
		return
	}

	// Add documentation-related terms to the search query if not already present
	docTerms := []string{"documentation", "docs", "readme", "wiki"}
	hasDocTerm := false
	for _, term := range docTerms {
		if strings.Contains(strings.ToLower(query), term) {
			hasDocTerm = true
			break
		}
	}

	if !hasDocTerm {
		query = query + " readme in:readme"
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	// Execute search
	repositories, nextPage, err := h.GitHubClient.SearchRepositories(c.Request.Context(), query, page, perPage)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "unauthorized: invalid GitHub token" {
			statusCode = http.StatusUnauthorized
		} else if err.Error() == "rate limit exceeded or access denied" {
			statusCode = http.StatusTooManyRequests
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:   "github_api_error",
			Message: err.Error(),
			Status:  statusCode,
		})
		return
	}

	// Prepare pagination links
	pagination := map[string]interface{}{
		"current_page": page,
		"per_page":     perPage,
		"next_page":    nextPage,
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Status:  http.StatusOK,
		Message: "Repositories retrieved successfully",
		Data: map[string]interface{}{
			"repositories": repositories,
			"pagination":   pagination,
		},
	})
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"message": "API is running",
	})
}
