package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Cache defines the interface for cache operations
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string, value interface{}) error
	
	// Set stores a value in the cache with the default TTL
	Set(ctx context.Context, key string, value interface{}) error
	
	// SetWithTTL stores a value in the cache with a specific TTL
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error
	
	// FlushAll removes all entries from the cache
	FlushAll(ctx context.Context) error
	
	// IsEnabled returns whether the cache is enabled
	IsEnabled() bool
	
	// Close closes the cache client
	Close() error
}

// KeyBuilder provides standardized methods for generating cache keys
type KeyBuilder struct {
	Prefix string
}

// NewKeyBuilder creates a new key builder with the given prefix
func NewKeyBuilder(prefix string) *KeyBuilder {
	return &KeyBuilder{
		Prefix: prefix,
	}
}

// RepositoryKey builds a cache key for repository data
func (kb *KeyBuilder) RepositoryKey(owner, repo string) string {
	return kb.Prefix + ":repo:" + owner + ":" + repo
}

// RepositoryDocumentationKey generates a unique cache key for repository documentation, including an optional ref (tag/branch).
func (kb *KeyBuilder) RepositoryDocumentationKey(owner, repo, ref string) string {
	// If ref is empty, we might cache it under a general key or a specific 'default' key.
	// For simplicity, include the ref directly. Empty ref means default branch was likely intended.
	return fmt.Sprintf("%s:repo_docs:%s:%s:%s", kb.Prefix, owner, repo, ref)
}

// RepositoryDocumentationMetadataKey generates a unique cache key for repository documentation metadata index.
func (kb *KeyBuilder) RepositoryDocumentationMetadataKey(owner, repo, ref string) string {
	return fmt.Sprintf("%s:doc_metadata:%s:%s:%s", kb.Prefix, owner, repo, ref)
}

// DocumentContentKey generates a unique cache key for an individual document content.
func (kb *KeyBuilder) DocumentContentKey(owner, repo, ref string, pathHash string) string {
	return fmt.Sprintf("%s:doc_content:%s:%s:%s:%s", kb.Prefix, owner, repo, ref, pathHash)
}

// SearchKey builds a cache key for repository search results
func (kb *KeyBuilder) SearchKey(query string, page, perPage int) string {
	// Sanitize query slightly for key usage
	query = strings.ReplaceAll(query, " ", "_")
	query = strings.ToLower(query)
	return fmt.Sprintf("%s:search:%s:page%s:per%s", kb.Prefix, query, strconv.Itoa(page), strconv.Itoa(perPage))
}

// CustomKey builds a custom cache key with the given parts
func (kb *KeyBuilder) CustomKey(parts ...string) string {
	key := kb.Prefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}
