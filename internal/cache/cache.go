package cache

import (
	"context"
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

// RepositoryDocumentationKey builds a cache key for repository documentation
func (kb *KeyBuilder) RepositoryDocumentationKey(owner, repo string) string {
	return kb.Prefix + ":docs:" + owner + ":" + repo
}

// SearchKey builds a cache key for repository search results
func (kb *KeyBuilder) SearchKey(query string, page, perPage int) string {
	return kb.Prefix + ":search:" + query + ":" + string(page) + ":" + string(perPage)
}

// CustomKey builds a custom cache key with the given parts
func (kb *KeyBuilder) CustomKey(parts ...string) string {
	key := kb.Prefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}
