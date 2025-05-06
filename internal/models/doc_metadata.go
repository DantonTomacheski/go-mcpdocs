package models

import (
	"time"
)

// DocumentMetadata contains lightweight metadata about a documentation file
type DocumentMetadata struct {
	Path      string    `json:"path"`
	Size      int       `json:"size"`
	SHA       string    `json:"sha"`
	CreatedAt time.Time `json:"created_at"`
}

// RepositoryDocumentationIndex represents the lightweight metadata cache
// for a repository's documentation
type RepositoryDocumentationIndex struct {
	RepositoryOwner string             `json:"repository_owner"`
	RepositoryName  string             `json:"repository_name"`
	RepositoryRef   string             `json:"repository_ref"`
	DocumentCount   int                `json:"document_count"`
	CreatedAt       time.Time          `json:"created_at"`
	Documents       []DocumentMetadata `json:"documents"`
}

// GenerateContentCacheKey generates the cache key for a specific document content
func (m *DocumentMetadata) GenerateContentCacheKey(prefix, owner, repo, ref string) string {
	return prefix + ":doc_content:" + owner + ":" + repo + ":" + ref + ":" + m.SHA
}
