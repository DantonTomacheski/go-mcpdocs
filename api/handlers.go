package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/dtomacheski/extract-data-go/internal/github"
	"github.com/dtomacheski/extract-data-go/internal/models"
	"github.com/dtomacheski/extract-data-go/internal/repository"
	"github.com/gin-gonic/gin"
)

// Handler contains the handlers for the API
type Handler struct {
	GitHubClient      *github.Client
	WorkerPoolSize    int
	DocumentRepository *repository.DocumentRepository
	Logger            *log.Logger
}

// NewHandler creates a new API handler
func NewHandler(client *github.Client, docRepo *repository.DocumentRepository, logger *log.Logger, workerPoolSize int) *Handler {
	return &Handler{
		GitHubClient:      client,
		WorkerPoolSize:    workerPoolSize,
		DocumentRepository: docRepo,
		Logger:            logger,
	}
}

// GetRepository handles fetching repository information
// Agora também busca automaticamente a documentação do repositório
func (h *Handler) GetRepository(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")

	if owner == "" || repo == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Owner and repository name are required",
			Status:  http.StatusBadRequest,
		})
		return
	}

	// Verifica se o cliente deseja explicitamente não incluir documentação
	skipDocs := c.Query("skip_docs") == "true"

	// Obtém informações básicas do repositório
	repository, err := h.GitHubClient.GetRepository(c.Request.Context(), owner, repo)
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

	// Se skipDocs for verdadeiro, retorna apenas as informações do repositório
	if skipDocs {
		c.JSON(http.StatusOK, models.SuccessResponse{
			Status:  http.StatusOK,
			Message: "Repository retrieved successfully (without docs)",
			Data:    repository,
		})
		return
	}

	// Configura um contexto que pode ser cancelado quando o cliente desconecta
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Configura o cancelamento quando o cliente desconecta
	go func() {
		<-c.Request.Context().Done()
		cancel()
	}()

	// Busca documentação do repositório em segundo plano
	h.Logger.Printf("Auto-fetching documentation for %s/%s", owner, repo)
	documentationItems, docErr := h.GitHubClient.GetRepositoryDocumentation(ctx, owner, repo, h.WorkerPoolSize)

	// Se houver documentação e não houver erro, armazena-a no MongoDB se configurado
	if docErr == nil && len(documentationItems) > 0 && h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
		h.Logger.Printf("Processing and storing documentation in TXT format for %s/%s", owner, repo)
		if storeErr := h.DocumentRepository.StoreDocumentation(ctx, documentationItems); storeErr != nil {
			h.Logger.Printf("Failed to store processed documentation in MongoDB: %v", storeErr)
		} else {
			h.Logger.Printf("Successfully processed and stored documentation in MongoDB for %s/%s", owner, repo)
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

	if owner == "" || repo == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Owner and repository name are required",
			Status:  http.StatusBadRequest,
		})
		return
	}

	// Set up a context that can be canceled when the client disconnects
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Set up cancellation when client disconnects
	go func() {
		<-c.Request.Context().Done()
		cancel()
	}()

	// Call the updated client function that returns all documentation items
	documentationItems, err := h.GitHubClient.GetRepositoryDocumentation(ctx, owner, repo, h.WorkerPoolSize)
	if err != nil {
		statusCode := http.StatusInternalServerError
		// Simplify error mapping based on likely errors from the updated client function
		if strings.Contains(err.Error(), "repository not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "invalid GitHub token") {
			statusCode = http.StatusUnauthorized
		} else if strings.Contains(err.Error(), "rate limit exceeded") {
			statusCode = http.StatusTooManyRequests
		} else if strings.Contains(err.Error(), "no documentation files found") || strings.Contains(err.Error(), "no documentation content could be successfully retrieved") {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, models.ErrorResponse{
			Error:   "github_api_error",
			Message: err.Error(),
			Status:  statusCode,
		})
		return
	}

	// Processar e armazenar documentação no MongoDB no formato TXT desejado
	if h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
		h.Logger.Printf("Processing and storing documentation in TXT format for %s/%s", owner, repo)
		if err := h.DocumentRepository.StoreDocumentation(ctx, documentationItems); err != nil {
			h.Logger.Printf("Failed to store processed documentation in MongoDB: %v", err)
			// Isso não é um erro crítico, ainda podemos retornar a documentação para o cliente
		} else {
			h.Logger.Printf("Successfully processed and stored documentation in MongoDB for %s/%s", owner, repo)
		}
	}

	// Construct the new response using RepositoryDocsResponse
	response := models.RepositoryDocsResponse{
		Status:             http.StatusOK,
		Message:            fmt.Sprintf("Successfully retrieved %d documentation files.", len(documentationItems)),
		RepositoryOwner:    owner,
		RepositoryName:     repo,
		ProcessedFilesCount: len(documentationItems),
		DocumentationItems: documentationItems,
	}

	c.JSON(http.StatusOK, response)
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
