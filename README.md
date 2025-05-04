# GitHub Documentation API

A high-performance API written in Go that fetches and processes GitHub repository documentation with proper error handling and concurrency.

## Features

- Fetch repository information from GitHub
- Extract documentation files (README, docs directory, etc.) from repositories
- Search for repositories with documentation
- Concurrent processing of documentation files
- Proper error handling with detailed error responses
- Graceful shutdown
- Configurable worker pool size and request timeouts

## Prerequisites

- Go 1.16 or higher
- GitHub Personal Access Token

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

3. Edit the `.env` file and add your GitHub Personal Access Token:

```
GITHUB_TOKEN=your_github_token_here
PORT=8080
WORKER_POOL_SIZE=5
REQUEST_TIMEOUT=30s
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

Fetches documentation files from a GitHub repository.

Example: `GET /api/v1/repos/google/go-github/docs`

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
