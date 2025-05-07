package models

import "time"

// RepositoryQueryResponse representa a resposta para consultas de repositórios
type RepositoryQueryResponse struct {
	// Metadados da resposta
	Meta ResponseMeta `json:"meta"`
	
	// Dados dos repositórios
	Data RepositoryData `json:"data"`
}

// ResponseMeta contém metadados sobre a resposta
type ResponseMeta struct {
	// Timestamp da resposta
	Timestamp time.Time `json:"timestamp"`
	
	// Informações de paginação
	Pagination PaginationInfo `json:"pagination,omitempty"`
	
	// Se a resposta veio do cache
	FromCache bool `json:"from_cache"`
	
	// Parâmetros da consulta que geraram esta resposta
	QueryParams map[string]string `json:"query_params,omitempty"`
}

// PaginationInfo contém informações sobre a paginação
type PaginationInfo struct {
	// Página atual
	CurrentPage int `json:"current_page"`
	
	// Tamanho da página
	PerPage int `json:"per_page"`
	
	// Total de itens
	TotalItems int `json:"total_items,omitempty"`
	
	// Total de páginas
	TotalPages int `json:"total_pages,omitempty"`
	
	// Número da próxima página (se disponível)
	NextPage int `json:"next_page,omitempty"`
	
	// Número da página anterior (se disponível)
	PrevPage int `json:"prev_page,omitempty"`
}

// RepositoryData contém os dados dos repositórios
type RepositoryData struct {
	// Lista de repositórios (quando múltiplos)
	Repositories []*RepositoryDetails `json:"repositories,omitempty"`
	
	// Detalhes de um único repositório (quando específico)
	Repository *RepositoryDetails `json:"repository,omitempty"`
}

// RepositoryDetails representa detalhes semânticos de um repositório
type RepositoryDetails struct {
	// Identificação do repositório
	ID int64 `json:"id"`
	
	// Nome completo (owner/repo)
	FullName string `json:"full_name"`
	
	// Nome do proprietário
	Owner string `json:"owner"`
	
	// Nome do repositório
	Name string `json:"name"`
	
	// URL do repositório
	URL string `json:"url"`
	
	// Descrição do repositório
	Description string `json:"description"`
	
	// URL da documentação do repositório (se disponível)
	DocsURL string `json:"docs_url,omitempty"`
	
	// Dados de popularidade
	Popularity RepositoryPopularity `json:"popularity"`
	
	// Ramo padrão
	DefaultBranch string `json:"default_branch"`
	
	// Tags disponíveis
	Tags []string `json:"tags,omitempty"`
	
	// Status do cache de documentação
	DocumentationStatus DocumentationStatus `json:"documentation_status,omitempty"`
	
	// Data da última atualização do repositório
	UpdatedAt time.Time `json:"updated_at"`
	
	// Data da criação do repositório
	CreatedAt time.Time `json:"created_at"`
}

// RepositoryPopularity contém métricas de popularidade de um repositório
type RepositoryPopularity struct {
	// Número de estrelas
	Stars int `json:"stars"`
	
	// Número de forks
	Forks int `json:"forks"`
	
	// Número de watchers
	Watchers int `json:"watchers"`
	
	// Problemas abertos
	OpenIssues int `json:"open_issues"`
}

// DocumentationStatus representa o status da documentação de um repositório
type DocumentationStatus struct {
	// Se a documentação está disponível
	Available bool `json:"available"`
	
	// Data da última atualização da documentação
	LastUpdated *time.Time `json:"last_updated,omitempty"`
	
	// Fonte da documentação (ex: "README", "docs/", "wiki")
	Source string `json:"source,omitempty"`
	
	// Versões disponíveis da documentação
	AvailableVersions []string `json:"available_versions,omitempty"`
}
