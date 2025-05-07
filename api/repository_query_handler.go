package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dtomacheski/extract-data-go/internal/models"
	"github.com/gin-gonic/gin"
)

// QueryRepositories é o novo endpoint para consultar repositórios de maneira semântica
// Permite consultar todos os repositórios ou um específico, com uma resposta
// estruturada para facilitar a integração com o frontend
func (h *Handler) QueryRepositories(c *gin.Context) {
	// Obtém os parâmetros de consulta
	queryParams := make(map[string]string)
	
	// Parâmetros específicos
	repositoryQuery := c.Query("repository")
	queryParams["repository"] = repositoryQuery
	
	// Parâmetros de paginação
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	queryParams["page"] = strconv.Itoa(page)
	
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}
	queryParams["per_page"] = strconv.Itoa(perPage)
	
	// Parâmetros de ordenação
	sortBy := c.DefaultQuery("sort_by", "stars")
	queryParams["sort_by"] = sortBy
	
	sortOrder := c.DefaultQuery("sort_order", "desc")
	queryParams["sort_order"] = sortOrder
	
	// Verifica se a consulta é para um repositório específico
	var response models.RepositoryQueryResponse
	fromCache := false
	
	// Inicializa a resposta com metadados
	response.Meta = models.ResponseMeta{
		Timestamp:   time.Now(),
		FromCache:   fromCache,
		QueryParams: queryParams,
		Pagination: models.PaginationInfo{
			CurrentPage: page,
			PerPage:     perPage,
		},
	}
	
	// Cria o contexto com cancelamento para operações GitHub
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	
	// Configurar cancelamento de contexto quando o cliente desconectar
	go func() {
		<-c.Request.Context().Done()
		cancel()
	}()
	
	// Caso específico: consulta por repositório específico
	if repositoryQuery != "" {
		// Verifica se o formato da consulta é válido (owner/repo)
		parts := strings.Split(repositoryQuery, "/")
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_repository_format",
				Message: "O formato do repositório deve ser 'proprietário/nome'",
				Status:  http.StatusBadRequest,
			})
			return
		}
		
		owner := parts[0]
		repo := parts[1]
		
		// Tenta recuperar o repositório do cache primeiro
		var cacheKey string
		if h.Cache != nil && h.Cache.IsEnabled() {
			cacheKey = h.KeyBuilder.RepositoryKey(owner, repo)
			var cachedRepo *models.Repository
			
			cacheErr := h.Cache.Get(ctx, cacheKey, &cachedRepo)
			if cacheErr == nil && cachedRepo != nil {
				// Cache hit
				h.Logger.Printf("Cache hit for repository query: %s/%s", owner, repo)
				
				// Converter para o formato de resposta semântica
				repoDetails := convertToRepositoryDetails(cachedRepo)
				
				// Preencher documentação se disponível no cache
				if h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
					enrichRepositoryWithDocumentationStatus(ctx, h, repoDetails, owner, repo)
				}
				
				response.Meta.FromCache = true
				response.Data.Repository = repoDetails
				
				c.JSON(http.StatusOK, response)
				return
			}
		}
		
		// Cache miss ou cache desabilitado, busca do GitHub
		repository, err := h.GitHubClient.GetRepository(ctx, owner, repo)
		if err != nil {
			statusCode := http.StatusInternalServerError
			errorMessage := err.Error()
			
			if strings.Contains(errorMessage, "not found") {
				statusCode = http.StatusNotFound
				errorMessage = "Repositório não encontrado"
			} else if strings.Contains(errorMessage, "unauthorized") {
				statusCode = http.StatusUnauthorized
			} else if strings.Contains(errorMessage, "rate limit") {
				statusCode = http.StatusTooManyRequests
			}
			
			c.JSON(statusCode, models.ErrorResponse{
				Error:   "github_api_error",
				Message: errorMessage,
				Status:  statusCode,
			})
			return
		}
		
		// Converter para o formato de resposta semântica
		repoDetails := convertToRepositoryDetails(repository)
		
		// Enriquecer com informações de documentação
		if h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
			enrichRepositoryWithDocumentationStatus(ctx, h, repoDetails, owner, repo)
		}
		
		// Salvar no cache para consultas futuras
		if h.Cache != nil && h.Cache.IsEnabled() && cacheKey != "" {
			h.Cache.Set(ctx, cacheKey, repository)
		}
		
		// Preencher resposta
		response.Data.Repository = repoDetails
		
		c.JSON(http.StatusOK, response)
		return
	}
	
	// Caso geral: lista de repositórios
	// Construir a query para busca no GitHub
	query := buildSearchQuery(c)
	
	// Executar a busca
	repositories, nextPage, err := h.GitHubClient.SearchRepositories(ctx, query, page, perPage)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMessage := err.Error()
		
		if strings.Contains(errorMessage, "unauthorized") {
			statusCode = http.StatusUnauthorized
		} else if strings.Contains(errorMessage, "rate limit") {
			statusCode = http.StatusTooManyRequests
		}
		
		c.JSON(statusCode, models.ErrorResponse{
			Error:   "github_api_error",
			Message: errorMessage,
			Status:  statusCode,
		})
		return
	}
	
	// Converter para o formato de resposta semântica
	repoDetailsList := make([]*models.RepositoryDetails, 0, len(repositories))
	for _, repo := range repositories {
		repoDetails := convertToRepositoryDetails(repo)
		
		// Opcional: enriquecer com informações de documentação
		// Descomentado por questões de performance - muitas chamadas simultâneas
		// Se você quiser esta funcionalidade, descomente:
		/*
		if h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
			parts := strings.Split(repo.FullName, "/")
			if len(parts) == 2 {
				enrichRepositoryWithDocumentationStatus(ctx, h, repoDetails, parts[0], parts[1])
			}
		}
		*/
		
		repoDetailsList = append(repoDetailsList, repoDetails)
	}
	
	// Preencher informações de paginação
	response.Meta.Pagination.NextPage = nextPage
	if page > 1 {
		response.Meta.Pagination.PrevPage = page - 1
	}
	
	// Preencher resposta
	response.Data.Repositories = repoDetailsList
	
	c.JSON(http.StatusOK, response)
}

// buildSearchQuery constrói a query para pesquisa de repositórios
func buildSearchQuery(c *gin.Context) string {
	var queryParts []string
	
	// Adicionar termos de busca se fornecidos
	searchTerm := c.Query("q")
	if searchTerm != "" {
		queryParts = append(queryParts, searchTerm)
	}
	
	// Filtro de linguagem
	language := c.Query("language")
	if language != "" {
		queryParts = append(queryParts, "language:"+language)
	}
	
	// Filtro de tópicos
	topics := c.Query("topics")
	if topics != "" {
		for _, topic := range strings.Split(topics, ",") {
			if topic = strings.TrimSpace(topic); topic != "" {
				queryParts = append(queryParts, "topic:"+topic)
			}
		}
	}
	
	// Filtro de licença
	license := c.Query("license")
	if license != "" {
		queryParts = append(queryParts, "license:"+license)
	}
	
	// Adicionar critérios de documentação se não especificados
	hasDocTerm := false
	for _, part := range queryParts {
		if strings.Contains(strings.ToLower(part), "readme") || 
		   strings.Contains(strings.ToLower(part), "documentation") || 
		   strings.Contains(strings.ToLower(part), "docs") {
			hasDocTerm = true
			break
		}
	}
	
	if !hasDocTerm {
		queryParts = append(queryParts, "readme in:readme")
	}
	
	// Filtro de tamanho
	minStars := c.Query("min_stars")
	if minStars != "" {
		queryParts = append(queryParts, "stars:>="+minStars)
	}
	
	// Ordenação (mapeada para orderBy:direction)
	sortBy := c.DefaultQuery("sort_by", "stars")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	
	// Mapear sort_by para o parâmetro do GitHub
	githubSortBy := sortBy
	if sortBy == "stars" {
		githubSortBy = "stars"
	} else if sortBy == "updated" {
		githubSortBy = "updated"
	} else if sortBy == "forks" {
		githubSortBy = "forks"
	}
	
	queryParts = append(queryParts, "sort:"+githubSortBy+"-"+sortOrder)
	
	// Construir a query final
	return strings.Join(queryParts, " ")
}

// convertToRepositoryDetails converte um modelo de repositório para o formato de detalhes semânticos
func convertToRepositoryDetails(repo *models.Repository) *models.RepositoryDetails {
	// Extrair owner do full_name
	parts := strings.Split(repo.FullName, "/")
	owner := ""
	if len(parts) >= 1 {
		owner = parts[0]
	}
	
	return &models.RepositoryDetails{
		ID:            repo.ID,
		FullName:      repo.FullName,
		Owner:         owner,
		Name:          repo.Name,
		URL:           repo.URL,
		Description:   repo.Description,
		DefaultBranch: repo.DefaultBranch,
		Popularity: models.RepositoryPopularity{
			Stars:      repo.Stars,
			Forks:      repo.Forks,
			// Watchers e OpenIssues não estão disponíveis no modelo atual
			Watchers:   0,
			OpenIssues: 0,
		},
		UpdatedAt: repo.UpdatedAt,
		CreatedAt: repo.CreatedAt,
	}
}

// enrichRepositoryWithDocumentationStatus enriquece os detalhes do repositório com informações de documentação
func enrichRepositoryWithDocumentationStatus(ctx context.Context, h *Handler, repoDetails *models.RepositoryDetails, owner, repo string) {
	// Verificar se há documentação armazenada
	if h.DocumentRepository != nil && h.DocumentRepository.IsEnabled() {
		lastUpdate, _ := h.DocumentRepository.GetLastUpdateTime(ctx, owner, repo)
		
		if lastUpdate != nil {
			// Documentação está disponível
			repoDetails.DocumentationStatus = models.DocumentationStatus{
				Available:   true,
				LastUpdated: lastUpdate,
				Source:      "processed",
			}
			
			// Construir URL de documentação
			repoDetails.DocsURL = "/api/v1/repos/" + owner + "/" + repo + "/docs"
			
			// Nota: GetAvailableVersions não está implementado ainda no DocumentRepository
			// Implementaremos isso posteriormente
			// Por enquanto, apenas deixamos AvailableVersions vazio
			// Futuro: repoDetails.DocumentationStatus.AvailableVersions = obterVersoesDisponiveis()
		} else {
			// Sem documentação ainda
			repoDetails.DocumentationStatus = models.DocumentationStatus{
				Available: false,
			}
		}
	}
}
