# Redis Cache Implementation

This document outlines the Redis caching implementation with Upstash in the go-mcpdocs project.

## Overview

We've implemented Redis caching to improve API performance, particularly for GitHub API calls that retrieve repository and documentation data. The cache layer stores responses from GitHub API calls to reduce API rate limits and improve response times.

## Architecture

The cache implementation follows these key principles:

1. **Separation of concerns**: The cache is implemented in a separate package in `internal/cache`
2. **Interface-based design**: The cache can be switched between implementations (in-memory, Redis, etc.)
3. **Graceful fallback**: If the cache is unavailable, the system falls back to direct API calls
4. **Configurable TTL**: Cache expiration is configurable at both global and per-item levels

## Package Structure

- `internal/cache/cache.go`: Defines the Cache interface and key builder utilities
- `internal/cache/redis.go`: Redis implementation of the Cache interface using go-redis/v9
- `internal/cache/redis_test.go`: Unit tests for the Redis cache implementation

## Configuration

### Environment Variables

Configure the Redis cache in the `.env` file:

```
# Redis Cache Configuration with Upstash
REDIS_URI=rediss://default:<PASSWORD>@<HOSTNAME>.upstash.io:6379
CACHE_TTL=1h
```

- `REDIS_URI`: Connection string for Upstash Redis (or any Redis server)
- `CACHE_TTL`: Default time-to-live for cached items (using Go duration format, e.g., "1h", "30m")

## Cache Keys

Cache keys are structured using the `KeyBuilder` to ensure consistency:

- Repository data: `mcpdocs:repo:<owner>:<repo>`
- Repository documentation: `mcpdocs:docs:<owner>:<repo>`
- Search results: `mcpdocs:search:<query>:<page>:<perPage>`

## Integration Points

The cache is integrated in:

1. `api.Handler.GetRepository`: Caches repository metadata
2. `api.Handler.GetRepositoryDocumentation`: Caches repository documentation

## Error Handling

The cache implementation includes robust error handling:

- Cache errors don't cause API failures (graceful fallback to GitHub API)
- Connection errors are logged but non-fatal
- Cached responses are marked with "(from cache)" in the response message

## Monitoring and Performance

The cache client logs key metrics:

- Cache hits and misses
- Operation times for Get/Set/Delete operations
- Connection status and errors

## Testing

We've implemented unit tests for the Redis cache implementation using miniredis for mock testing. Run tests with:

```
go test -v ./internal/cache
```

## Future Improvements

Potential enhancements for the caching system:

1. **Metrics Collection**: Add Prometheus metrics for cache hit/miss rates and latency
2. **Cache Invalidation**: Add explicit cache invalidation after repository updates
3. **Advanced Key Patterns**: Extend key patterns for more API endpoints
4. **Cache Compression**: Add compression for large documentation responses
5. **Cluster Support**: Support for Redis Cluster for higher availability

## Troubleshooting

Common issues:

- **Connection Issues**: Verify Upstash credentials and network connectivity
- **Unexpected Cache Misses**: Check TTL settings and key construction
- **Memory Usage**: Monitor Redis memory usage and adjust TTL values if needed
