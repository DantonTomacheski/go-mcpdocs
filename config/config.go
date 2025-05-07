package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	GitHubToken    string
	Port           string
	WorkerPoolSize int
	RequestTimeout time.Duration
	MongoURI       string
	EnableMongoDB  bool
	RedisURI       string
	EnableCache    bool
	CacheTTL       time.Duration
	MinDaysBetweenRefreshes int // Minimum days required between documentation refreshes
	
	// JWT Authentication settings
	JWTSecret           string
	JWTAccessDuration   time.Duration
	JWTRefreshDuration  time.Duration
	JWTIssuer           string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Get GitHub token
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	// Get port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	// Get worker pool size
	workerPoolSizeStr := os.Getenv("WORKER_POOL_SIZE")
	workerPoolSize := 5 // Default value
	if workerPoolSizeStr != "" {
		var err error
		workerPoolSize, err = strconv.Atoi(workerPoolSizeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid WORKER_POOL_SIZE: %v", err)
		}
		if workerPoolSize <= 0 {
			workerPoolSize = 5 // Ensure a positive value
		}
	}

	// Get request timeout
	timeoutStr := os.Getenv("REQUEST_TIMEOUT")
	timeout := 30 * time.Second // Default value
	if timeoutStr != "" {
		var err error
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REQUEST_TIMEOUT: %v", err)
		}
		if timeout <= 0 {
			timeout = 30 * time.Second // Ensure a positive value
		}
	}

	// Get MongoDB URI
	mongoURI := os.Getenv("MONGO_URI")
	enableMongoDB := mongoURI != ""

	// Get Redis URI
	redisURI := os.Getenv("REDIS_URI")
	enableCache := redisURI != ""

	// Get cache TTL
	cacheTTLStr := os.Getenv("CACHE_TTL")
	cacheTTL := 1 * time.Hour // Default value
	if cacheTTLStr != "" {
		var err error
		cacheTTL, err = time.ParseDuration(cacheTTLStr)
		if err != nil {
			return nil, fmt.Errorf("invalid CACHE_TTL: %v", err)
		}
		if cacheTTL <= 0 {
			cacheTTL = 1 * time.Hour // Ensure a positive value
		}
	}

	// Get minimum days between refreshes
	minDaysBetweenRefreshesStr := os.Getenv("MIN_DAYS_BETWEEN_REFRESHES")
	minDaysBetweenRefreshes := 3 // Default value: 3 days
	if minDaysBetweenRefreshesStr != "" {
		var err error
		minDaysBetweenRefreshes, err = strconv.Atoi(minDaysBetweenRefreshesStr)
		if err != nil {
			return nil, fmt.Errorf("invalid MIN_DAYS_BETWEEN_REFRESHES: %v", err)
		}
		if minDaysBetweenRefreshes < 0 {
			minDaysBetweenRefreshes = 3 // Ensure a positive value
		}
	}

	// JWT settings
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default_jwt_secret_key_change_me_in_production" // Default secret (not secure for production)
	}

	// JWT access token duration
	accessDurStr := os.Getenv("JWT_ACCESS_DURATION")
	accessDuration := 15 * time.Minute // Default value: 15 minutes
	if accessDurStr != "" {
		var err error
		accessDuration, err = time.ParseDuration(accessDurStr)
		if err != nil {
			return nil, fmt.Errorf("invalid JWT_ACCESS_DURATION: %v", err)
		}
		if accessDuration <= 0 {
			accessDuration = 15 * time.Minute // Ensure a positive value
		}
	}

	// JWT refresh token duration
	refreshDurStr := os.Getenv("JWT_REFRESH_DURATION")
	refreshDuration := 7 * 24 * time.Hour // Default value: 7 days
	if refreshDurStr != "" {
		var err error
		refreshDuration, err = time.ParseDuration(refreshDurStr)
		if err != nil {
			return nil, fmt.Errorf("invalid JWT_REFRESH_DURATION: %v", err)
		}
		if refreshDuration <= 0 {
			refreshDuration = 7 * 24 * time.Hour // Ensure a positive value
		}
	}

	// JWT issuer
	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		jwtIssuer = "github-docs-api" // Default issuer
	}

	return &Config{
		GitHubToken:    token,
		Port:           port,
		WorkerPoolSize: workerPoolSize,
		RequestTimeout: timeout,
		MongoURI:       mongoURI,
		EnableMongoDB:  enableMongoDB,
		RedisURI:       redisURI,
		EnableCache:    enableCache,
		CacheTTL:       cacheTTL,
		MinDaysBetweenRefreshes: minDaysBetweenRefreshes,
		JWTSecret:          jwtSecret,
		JWTAccessDuration:  accessDuration,
		JWTRefreshDuration: refreshDuration,
		JWTIssuer:          jwtIssuer,
	}, nil
}
