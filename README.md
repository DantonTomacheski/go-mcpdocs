# go-mcpdocs - Documentation API

A high-performance API written in Go that extracts, processes and provides documentation from GitHub repositories with proper error handling, concurrency, and storage capabilities. This API is designed to serve as a reliable source of up-to-date documentation for LLMs and developers.

## Features

- Fetch comprehensive repository information from GitHub
- Extract and process documentation files (README, docs directory, etc.) from repositories
- Search for repositories with documentation
- Process and format documentation for better consumption by LLMs
- Store processed documentation in MongoDB for faster retrieval
- Retrieve documentation directly from URL paths
- Concurrent processing with configurable worker pools
- Enhanced error handling with detailed error responses
- Graceful shutdown with proper resource cleanup
- CORS protection and request timeout middleware
- Configurable worker pool size and request timeouts

## Prerequisites

- Go 1.16 or higher
- GitHub Personal Access Token
- MongoDB (optional, for document storage)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/yourusername/extract-data-go.git
cd extract-data-go
```

2. Create a `.env` file by copying the example file:

```bash
cp .env.example .env
```

3. Edit the `.env` file and add your GitHub Personal Access Token and MongoDB connection string (if using document storage):

```
GITHUB_TOKEN=your_github_token_here
PORT=8080
WORKER_POOL_SIZE=5
REQUEST_TIMEOUT=30s
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=go-mcpdocs
```

## How to Run

1. Build the application:

```bash
go build -o github-doc-api
```

2. Run the application:

```bash
./github-doc-api
```

Alternatively, you can use `go run`:

```bash
go run main.go
```

## API Endpoints

### Health Check

```
GET /health
```

Returns the status of the API.

### Get Repository Information

```
GET /api/v1/repos/:owner/:repo
```

Retrieves information about a GitHub repository.

Example: `GET /api/v1/repos/google/go-github`

### Get Repository Documentation

```
GET /api/v1/repos/:owner/:repo/docs
```

Fetches documentation files from a GitHub repository. Documentation is automatically processed and stored in MongoDB if configured.

Example: `GET /api/v1/repos/google/go-github/docs`

### Get Documentation from URL

```
GET /api/v1/docs?url=:url
```

Fetches documentation directly from a GitHub repository URL.

Example: `GET /api/v1/docs?url=https://github.com/google/go-github`

### Get Processed Documentation Snippets

```
GET /api/v1/snippets?url=:url
```

Retrieves enhanced, processed documentation snippets from a GitHub repository URL in a format optimized for LLMs.

Example: `GET /api/v1/snippets?url=https://github.com/google/go-github`

### Search Repositories

```
GET /api/v1/search/repos?q=:query&page=:page&per_page=:perPage
```

Searches for GitHub repositories with documentation.

Parameters:
- `q`: Search query
- `page`: Page number (default: 1)
- `per_page`: Number of results per page (default: 10, max: 100)

Example: `GET /api/v1/search/repos?q=golang+api&page=1&per_page=10`

## Configuration

The application can be configured using environment variables:

- `GITHUB_TOKEN`: Your GitHub Personal Access Token (required)
- `PORT`: The port on which the API server will listen (default: 8080)
- `WORKER_POOL_SIZE`: Number of concurrent workers for processing documentation (default: 5)
- `REQUEST_TIMEOUT`: Timeout for GitHub API requests (default: 30s)
- `MONGODB_URI`: MongoDB connection string (optional, for document storage)
- `MONGODB_DATABASE`: MongoDB database name (optional, default: go-mcpdocs)

## Error Handling

The API returns detailed error responses in JSON format:

```json
{
  "error": "error_code",
  "message": "Detailed error message",
  "status": 400
}
```

Common error codes:
- `invalid_request`: Invalid request parameters
- `github_api_error`: Error from GitHub API
- `not_found`: Resource not found
- `unauthorized`: Invalid GitHub token

## Development

Run tests:

```bash
go test ./...
```

Format code:

```bash
go fmt ./...
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

- [Google's go-github](https://github.com/google/go-github) - GitHub API client for Go
- [Gin Web Framework](https://github.com/gin-gonic/gin) - HTTP web framework for Go
- [MongoDB Go Driver](https://pkg.go.dev/go.mongodb.org/mongo-driver) - Official MongoDB driver for Go

## Project Status and Roadmap

This project is part of the go-mcpdocs initiative to provide high-quality, up-to-date documentation to LLMs and developers. See the `aonde-estamos.md` file for current project status details.

Planned enhancements for 2025 include:

1. Authentication and Security (JWT/OAuth, rate limiting)
2. Cache and Performance optimizations
3. Enhanced document processors for multiple formats
4. Additional API endpoints and OpenAPI documentation
5. Improved observability with structured logging and metrics
6. Enhanced user experience
7. Multi-region deployment and asynchronous processing
