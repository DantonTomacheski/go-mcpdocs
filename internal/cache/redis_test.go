package cache

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

// testStruct is a simple struct for testing cache operations
type testStruct struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// setupMockRedis creates a mock Redis server for testing
func setupMockRedis(t *testing.T) (*miniredis.Miniredis, *RedisClient) {
	// Create a mock Redis server
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create mock Redis server: %v", err)
	}

	// Create Redis URI for the mock server
	redisURI := "redis://" + s.Addr()

	// Create a logger for testing
	logger := log.New(os.Stdout, "[TEST-REDIS] ", log.LstdFlags)

	// Create a Redis client with the mock server
	client, err := NewRedisClient(RedisConfig{
		RedisURI:   redisURI,
		Enabled:    true,
		DefaultTTL: 1 * time.Second,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}

	return s, client
}

// TestRedisClient_Get_Set tests the Get and Set operations
func TestRedisClient_Get_Set(t *testing.T) {
	s, client := setupMockRedis(t)
	defer s.Close()
	defer client.Close()

	ctx := context.Background()

	// Test data
	key := "test:user:1"
	expected := testStruct{
		ID:   "1",
		Name: "John Doe",
		Age:  30,
	}

	// Set the test data in cache
	err := client.Set(ctx, key, expected)
	assert.NoError(t, err)

	// Get the test data from cache
	var actual testStruct
	err = client.Get(ctx, key, &actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

// TestRedisClient_SetWithTTL tests the SetWithTTL operation
func TestRedisClient_SetWithTTL(t *testing.T) {
	s, client := setupMockRedis(t)
	defer s.Close()
	defer client.Close()

	ctx := context.Background()

	// Test data
	key := "test:user:2"
	expected := testStruct{
		ID:   "2",
		Name: "Jane Doe",
		Age:  28,
	}

	// Set the test data in cache with TTL
	err := client.SetWithTTL(ctx, key, expected, 500*time.Millisecond)
	assert.NoError(t, err)

	// Verify the key exists
	var actual testStruct
	err = client.Get(ctx, key, &actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	// Wait for TTL to expire
	time.Sleep(600 * time.Millisecond)

	// Verify the key is gone
	err = client.Get(ctx, key, &actual)
	assert.Error(t, err)
	assert.Equal(t, ErrCacheMiss, err)
}

// TestRedisClient_Delete tests the Delete operation
func TestRedisClient_Delete(t *testing.T) {
	s, client := setupMockRedis(t)
	defer s.Close()
	defer client.Close()

	ctx := context.Background()

	// Test data
	key := "test:user:3"
	expected := testStruct{
		ID:   "3",
		Name: "Bob Smith",
		Age:  35,
	}

	// Set the test data in cache
	err := client.Set(ctx, key, expected)
	assert.NoError(t, err)

	// Delete the key
	err = client.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify the key is gone
	var actual testStruct
	err = client.Get(ctx, key, &actual)
	assert.Error(t, err)
	assert.Equal(t, ErrCacheMiss, err)
}

// TestRedisClient_FlushAll tests the FlushAll operation
func TestRedisClient_FlushAll(t *testing.T) {
	s, client := setupMockRedis(t)
	defer s.Close()
	defer client.Close()

	ctx := context.Background()

	// Set multiple test items
	for i := 1; i <= 3; i++ {
		key := "test:user:" + string(i+'0')
		data := testStruct{
			ID:   string(i + '0'),
			Name: "User " + string(i+'0'),
			Age:  20 + i,
		}

		err := client.Set(ctx, key, data)
		assert.NoError(t, err)
	}

	// Flush all keys
	err := client.FlushAll(ctx)
	assert.NoError(t, err)

	// Verify no keys exist
	for i := 1; i <= 3; i++ {
		key := "test:user:" + string(i+'0')
		var actual testStruct
		err = client.Get(ctx, key, &actual)
		assert.Error(t, err)
		assert.Equal(t, ErrCacheMiss, err)
	}
}

// TestRedisClient_Disabled tests the behavior when the client is disabled
func TestRedisClient_Disabled(t *testing.T) {
	// Create a disabled Redis client
	client := &RedisClient{
		enabled: false,
		logger:  log.New(os.Stdout, "[TEST-DISABLED] ", log.LstdFlags),
	}

	ctx := context.Background()

	// Test data
	key := "test:user:4"
	data := testStruct{
		ID:   "4",
		Name: "Alice Johnson",
		Age:  32,
	}

	// All operations should return ErrCacheDown
	assert.Equal(t, false, client.IsEnabled())

	err := client.Set(ctx, key, data)
	assert.Error(t, err)
	assert.Equal(t, ErrCacheDown, err)

	err = client.SetWithTTL(ctx, key, data, 1*time.Second)
	assert.Error(t, err)
	assert.Equal(t, ErrCacheDown, err)

	var actual testStruct
	err = client.Get(ctx, key, &actual)
	assert.Error(t, err)
	assert.Equal(t, ErrCacheDown, err)

	err = client.Delete(ctx, key)
	assert.Error(t, err)
	assert.Equal(t, ErrCacheDown, err)

	err = client.FlushAll(ctx)
	assert.Error(t, err)
	assert.Equal(t, ErrCacheDown, err)

	// Close should not error
	err = client.Close()
	assert.NoError(t, err)
}
