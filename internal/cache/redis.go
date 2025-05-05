package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Common errors
var (
	ErrCacheMiss  = errors.New("cache miss")
	ErrCacheDown  = errors.New("cache service unavailable")
	ErrInvalidTTL = errors.New("invalid TTL value")
)

// RedisClient is the cache implementation using Redis
type RedisClient struct {
	client  *redis.Client
	enabled bool
	defaultTTL time.Duration
	logger  *log.Logger
}

// RedisConfig holds Redis client configuration
type RedisConfig struct {
	RedisURI   string
	Enabled    bool
	DefaultTTL time.Duration
	Logger     *log.Logger
}

// NewRedisClient creates a new Redis cache client
func NewRedisClient(cfg RedisConfig) (*RedisClient, error) {
	if !cfg.Enabled {
		return &RedisClient{
			enabled: false,
			logger:  cfg.Logger,
		}, nil
	}

	// Parse Redis URI
	opt, err := redis.ParseURL(cfg.RedisURI)
	if err != nil {
		if cfg.Logger != nil {
			cfg.Logger.Printf("Failed to parse Redis URI: %v", err)
		}
		return nil, err
	}

	// Create Redis client
	client := redis.NewClient(opt)

	// Validate connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err = client.Ping(ctx).Result()
	if err != nil {
		if cfg.Logger != nil {
			cfg.Logger.Printf("Failed to connect to Redis: %v", err)
		}
		return nil, err
	}

	// Set default TTL if not provided
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 1 * time.Hour
	}

	return &RedisClient{
		client:     client,
		enabled:    true,
		defaultTTL: cfg.DefaultTTL,
		logger:     cfg.Logger,
	}, nil
}

// IsEnabled returns whether the cache is enabled
func (c *RedisClient) IsEnabled() bool {
	return c.enabled && c.client != nil
}

// Get retrieves a value from the cache
func (c *RedisClient) Get(ctx context.Context, key string, value interface{}) error {
	if !c.IsEnabled() {
		return ErrCacheDown
	}

	// Add logging
	startTime := time.Now()
	defer func() {
		if c.logger != nil {
			c.logger.Printf("Cache GET operation for key '%s' took %v", key, time.Since(startTime))
		}
	}()

	// Try to get from cache
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		if c.logger != nil {
			c.logger.Printf("Redis GET error: %v", err)
		}
		return err
	}

	// Unmarshal data
	if err = json.Unmarshal(data, value); err != nil {
		if c.logger != nil {
			c.logger.Printf("Failed to unmarshal cached data: %v", err)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Printf("Cache HIT for key: %s", key)
	}
	return nil
}

// Set stores a value in the cache with the default TTL
func (c *RedisClient) Set(ctx context.Context, key string, value interface{}) error {
	return c.SetWithTTL(ctx, key, value, c.defaultTTL)
}

// SetWithTTL stores a value in the cache with a specific TTL
func (c *RedisClient) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.IsEnabled() {
		return ErrCacheDown
	}

	if ttl < 0 {
		return ErrInvalidTTL
	}

	// Add logging
	startTime := time.Now()
	defer func() {
		if c.logger != nil {
			c.logger.Printf("Cache SET operation for key '%s' took %v", key, time.Since(startTime))
		}
	}()

	// Marshal data
	data, err := json.Marshal(value)
	if err != nil {
		if c.logger != nil {
			c.logger.Printf("Failed to marshal data for caching: %v", err)
		}
		return err
	}

	// Set in cache
	err = c.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		if c.logger != nil {
			c.logger.Printf("Redis SET error: %v", err)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Printf("Cache SET successful for key: %s with TTL: %v", key, ttl)
	}
	return nil
}

// Delete removes a value from the cache
func (c *RedisClient) Delete(ctx context.Context, key string) error {
	if !c.IsEnabled() {
		return ErrCacheDown
	}

	// Add logging
	startTime := time.Now()
	defer func() {
		if c.logger != nil {
			c.logger.Printf("Cache DELETE operation for key '%s' took %v", key, time.Since(startTime))
		}
	}()

	// Delete from cache
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		if c.logger != nil {
			c.logger.Printf("Redis DELETE error: %v", err)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Printf("Cache DELETE successful for key: %s", key)
	}
	return nil
}

// FlushAll removes all entries from the cache
func (c *RedisClient) FlushAll(ctx context.Context) error {
	if !c.IsEnabled() {
		return ErrCacheDown
	}

	// Add logging
	startTime := time.Now()
	defer func() {
		if c.logger != nil {
			c.logger.Printf("Cache FLUSHALL operation took %v", time.Since(startTime))
		}
	}()

	// Flush cache
	err := c.client.FlushAll(ctx).Err()
	if err != nil {
		if c.logger != nil {
			c.logger.Printf("Redis FLUSHALL error: %v", err)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Printf("Cache FLUSHALL successful")
	}
	return nil
}

// Close closes the Redis client
func (c *RedisClient) Close() error {
	if !c.IsEnabled() {
		return nil
	}

	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
