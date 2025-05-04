package models

// CodeSnippet represents a code snippet extracted from documentation
type CodeSnippet struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Language    string `json:"language"`
	Code        string `json:"code"`
}

// DocumentationResponse represents the full response with extracted snippets
type DocumentationResponse struct {
	RepositoryName string        `json:"repository_name"`
	RepositoryURL  string        `json:"repository_url"`
	TotalSnippets  int           `json:"total_snippets"`
	TotalFiles     int           `json:"total_files"`
	Snippets       []CodeSnippet `json:"snippets"`
}
