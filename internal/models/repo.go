package models

import "time"

// Repository represents GitHub repository data
type Repository struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Description     string    `json:"description"`
	Stars           int       `json:"stars"`
	Forks           int       `json:"forks"`
	Language        string    `json:"language"`
	Topics          []string  `json:"topics"`
	DefaultBranch   string    `json:"default_branch"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	URL             string    `json:"url"`
	HTMLURL         string    `json:"html_url"`
	DocumentationURL string   `json:"documentation_url,omitempty"`
}

// Documentation represents documentation content for a repo
type Documentation struct {
	RepoID      int64  `json:"repo_id"`
	RepoName    string `json:"repo_name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	SHA         string `json:"sha"`
	URL         string `json:"url"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// SuccessResponse - Generic success response (used by multiple handlers)
type SuccessResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// RepositoryDocsResponse represents the consolidated response for the /repos/:owner/:repo/docs endpoint
type RepositoryDocsResponse struct {
	Status             int             `json:"status"`
	Message            string          `json:"message"`
	RepositoryOwner    string          `json:"repository_owner"`
	RepositoryName     string          `json:"repository_name"`
	ProcessedFilesCount int             `json:"processed_files_count"`
	DocumentationItems []Documentation `json:"documentation_items"`
}
