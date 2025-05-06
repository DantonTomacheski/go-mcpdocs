package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"strings"
	"time"

	"github.com/dtomacheski/extract-data-go/internal/models"
	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

// Client represents a GitHub API client
type Client struct {
	client  *github.Client
	timeout time.Duration
}

// NewClient creates a new GitHub API client with authentication
func NewClient(token string, timeout time.Duration) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	
	return &Client{
		client:  github.NewClient(tc),
		timeout: timeout,
	}
}

// GetRepository fetches a GitHub repository by owner and repo name
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*models.Repository, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	
	repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, processGitHubError(err)
	}
	
	return convertToRepositoryModel(repository), nil
}

// GetRepositoryDocumentation fetches documentation for a repository with concurrency, targeting a specific ref (tag/branch).
// If ref is empty, it defaults to the repository's default branch.
func (c *Client) GetRepositoryDocumentation(ctx context.Context, owner, repo, ref string, concurrencyLimit int) ([]models.Documentation, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	
	// Define variable err at the function level so it's available throughout the function
	var err error
	
	// Determine the ref to use (provided ref or default branch)
	var refToUse string
	if ref == "" {
		// No ref provided, use default branch
		repository, err := c.GetRepository(ctx, owner, repo)
		if err != nil {
			log.Printf("Error getting repository %s/%s to determine default branch: %v\n", owner, repo, err)
			return nil, err
		}
		refToUse = repository.DefaultBranch
		if refToUse == "" {
			log.Printf("Error: Default branch is empty for %s/%s\n", owner, repo)
			return nil, fmt.Errorf("default branch for repository %s/%s is empty", owner, repo)
		}
		log.Printf("No specific ref provided, using default branch '%s' for %s/%s", refToUse, owner, repo)
	} else {
		// Use the provided ref
		refToUse = ref
		log.Printf("Using provided ref '%s' for %s/%s", refToUse, owner, repo)
	}

	log.Printf("Attempting to fetch documentation for %s/%s from ref '%s'", owner, repo, refToUse)

	var docPaths []string
	searchInDocs := false

	// Define caminhos comuns de documentação em diferentes repositórios
	commonDocPaths := []string{
		"docs",           // Caminho padrão (Next.js, Vue, etc)
		"src/content",    // React.dev
		"Documentation",  // Swift, alguns projetos da Apple
		"documentation", // Variação com letra minúscula
		"doc",           // Alguns projetos antigos
	}
	
	// Variável para armazenar o caminho de documentação encontrado
	var foundDocPath string
	
	// 1. Verificar cada caminho possível de documentação
	for _, path := range commonDocPaths {
		log.Printf("Verificando existência da pasta %s para %s/%s no ref '%s'...\n", path, owner, repo, refToUse)
		docsDirOpts := &github.RepositoryContentGetOptions{Ref: refToUse}
		
		// Verificamos se o diretório existe
		var dirErr error
		_, _, _, dirErr = c.client.Repositories.GetContents(ctx, owner, repo, path, docsDirOpts)
		
		if dirErr == nil {
			// Caminho encontrado
			log.Printf("Pasta de documentação %s encontrada para %s/%s no ref '%s'.\n", path, owner, repo, refToUse)
			searchInDocs = true
			foundDocPath = path
			break
		} else {
			// Verificar se o erro é 404 Not Found
			ghErr, ok := dirErr.(*github.ErrorResponse)
			if ok && ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("Pasta %s não encontrada para %s/%s no ref '%s'.\n", path, owner, repo, refToUse)
				// Continuamos verificando outros caminhos - NÃO fazemos break aqui
			} else {
				// Tratamos erros inesperados
				log.Printf("Erro inesperado verificando pasta %s para %s/%s no ref '%s': %v\n", path, owner, repo, refToUse, dirErr)
			}
		}
	}
	

	if !searchInDocs {
		log.Printf("Nenhuma pasta de documentação encontrada para %s/%s no ref '%s'.\n", owner, repo, refToUse)
	}

	// 2. Perform search based on whether a valid documentation path exists
	if searchInDocs {
		log.Printf("Listando arquivos de documentação na pasta %s de %s/%s no ref '%s'...\n", foundDocPath, owner, repo, refToUse)
		// Usar nossa função de listagem recursiva no caminho encontrado
		docPaths, err = c.listDocFilesInPath(ctx, owner, repo, foundDocPath, refToUse)
		if err != nil {
			log.Printf("Erro ao listar arquivos na pasta %s em %s/%s no ref '%s': %v\n", foundDocPath, owner, repo, refToUse, err)
			return nil, fmt.Errorf("erro ao listar arquivos de documentação na pasta %s: %w", foundDocPath, err)
		}
		if len(docPaths) == 0 {
			log.Printf("Nenhum arquivo de documentação encontrado na pasta %s em %s/%s no ref '%s'.\n", foundDocPath, owner, repo, refToUse)
			searchInDocs = false
		} else {
			log.Printf("Encontrados %d arquivos de documentação na pasta %s de %s/%s no ref '%s'\n", len(docPaths), foundDocPath, owner, repo, refToUse)
		}
	}

	// Se não existe pasta /docs/ ou não existem arquivos nela, retornamos um erro
	if !searchInDocs {
		log.Printf("A pasta /docs/ não foi encontrada em %s/%s no ref '%s' ou você configurou para usar apenas a pasta /docs/\n", owner, repo, refToUse)
		return nil, errors.New("a pasta /docs/ não foi encontrada no repositório ou está vazia")
	}

	// 4. Fetch content for the determined docPaths
	log.Printf("Fetching content for %d documentation paths for %s/%s from ref '%s' using concurrency %d...\n", len(docPaths), owner, repo, refToUse, concurrencyLimit)
	var (
		wg            sync.WaitGroup
		mu            sync.Mutex
		documentation = []models.Documentation{} // Initialize here instead of at the top
		errChan       = make(chan error, len(docPaths))
		semaphore     = make(chan struct{}, concurrencyLimit)
	)

	for _, path := range docPaths {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore slot

		go func(p string) {
			defer func() {
				<-semaphore // Release semaphore slot
				wg.Done()
			}()

			// Fetch content using the determined default branch
			doc, err := c.getFileContent(ctx, owner, repo, p, refToUse)
			if err != nil {
				// Log specific file fetch errors
				log.Printf("Error fetching content for %s/%s path %s from ref '%s': %v\n", owner, repo, p, refToUse, err)
				// Send a non-blocking error to avoid deadlock if channel buffer is full
				select {
				case errChan <- fmt.Errorf("error fetching content for %s: %w", p, err):
				default:
					log.Printf("Error channel full, discarding error for %s\n", p)
				}
				return
			}

			if doc == nil {
				log.Printf("Skipping nil content for path %s in %s/%s from ref '%s'\n", p, owner, repo, refToUse)
				return // Skip if content fetching somehow returned nil without error
			}

			// Convert to model and add to result
			content, err := doc.GetContent() // Handles base64 decoding
			if err != nil {
				log.Printf("Error getting/decoding content for %s from ref '%s': %v\n", p, refToUse, err)
				select {
				case errChan <- fmt.Errorf("error getting/decoding content for %s: %w", p, err):
				default:
					log.Printf("Error channel full, discarding content error for %s\n", p)
				}
				return
			}

			docModel := models.Documentation{
				RepoID:      0, // Repository ID is not fetched in this function
				RepoName:    fmt.Sprintf("%s/%s", owner, repo),
				Path:        p,
				Content:     content,
				ContentType: doc.GetType(),
				Size:        doc.GetSize(),
				SHA:         doc.GetSHA(),
				URL:         doc.GetHTMLURL(), // Use HTML URL for easier browser access if needed
			}

			mu.Lock()
			documentation = append(documentation, docModel)
			mu.Unlock()
		}(path)
	}

	wg.Wait()
	close(errChan)

	// Check for errors during fetch
	var fetchErrors []string
	for err := range errChan {
		if err != nil {
			fetchErrors = append(fetchErrors, err.Error())
		}
	}

	if len(fetchErrors) > 0 {
		// If we got *some* docs despite errors, return them but log the errors.
		// If we got *no* docs and there were errors, return the error.
		log.Printf("%d errors occurred during content fetch for %s/%s from ref '%s': %s\n", len(fetchErrors), owner, repo, refToUse, strings.Join(fetchErrors, "; "))
		if len(documentation) == 0 {
			return nil, fmt.Errorf("failed to fetch documentation content: %s", fetchErrors[0]) // Return first error
		}
	}

	if len(documentation) == 0 {
		// This case now means either no paths were found initially, or all fetches failed.
		log.Printf("No documentation content could be successfully retrieved for %s/%s from ref '%s'.\n", owner, repo, refToUse)
		return nil, errors.New("no documentation content could be successfully retrieved")
	}

	log.Printf("Successfully retrieved content for %d documentation files from %s/%s from ref '%s'\n", len(documentation), owner, repo, refToUse)
	return documentation, nil
}

// findDocumentation searches for documentation files and directories
func (c *Client) findDocumentation(ctx context.Context, owner, repo, branch string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	
	// Common documentation paths - Expanded to match more documentation locations
	commonPaths := []string{
		"README.md",
		"docs",
		"doc",
		"documentation",
		"wiki",
		".github",
		"CONTRIBUTING.md",
		"CHANGELOG.md",
		"API.md",
		"content/docs",
		"website/docs",
		"src/docs",
		"pages/docs",
		"examples",
		"tutorials",
		"guides",
		"reference"}
	
	var docPaths []string
	
	// Check for README.md first (most common)
	_, _, resp, err := c.client.Repositories.GetContents(ctx, owner, repo, "README.md", &github.RepositoryContentGetOptions{
		Ref: branch,
	})
	
	if err == nil {
		docPaths = append(docPaths, "README.md")
	} else if resp != nil && resp.StatusCode != http.StatusNotFound {
		return nil, processGitHubError(err)
	}
	
	// Check for other documentation files/directories
	for _, path := range commonPaths[1:] {
		_, dirContent, resp, err := c.client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
			Ref: branch,
		})
		
		if err == nil {
			if len(dirContent) > 0 {
				// If it's a directory, add all markdown files
				for _, content := range dirContent {
					if content.GetType() == "file" && (isMarkdownFile(content.GetName()) || isDocumentationFile(content.GetName())) {
						docPaths = append(docPaths, content.GetPath())
					}
				}
			} else {
				// If it's a file, just add it
				docPaths = append(docPaths, path)
			}
		} else if resp != nil && resp.StatusCode != http.StatusNotFound {
			return nil, processGitHubError(err)
		}
	}
	
	return docPaths, nil
}

// getFileContent fetches the content of a file from GitHub
func (c *Client) getFileContent(ctx context.Context, owner, repo, path, branch string) (*github.RepositoryContent, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	
	fileContent, _, resp, err := c.client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: branch,
	})
	
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil // Return nil without error for not found
		}
		return nil, processGitHubError(err)
	}
	
	return fileContent, nil
}

// SearchRepositories searches for repositories with documentation
func (c *Client) SearchRepositories(ctx context.Context, query string, page, perPage int) ([]*models.Repository, int, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	
	result, resp, err := c.client.Search.Repositories(ctx, query, opts)
	if err != nil {
		return nil, 0, processGitHubError(err)
	}
	
	repositories := make([]*models.Repository, 0, len(result.Repositories))
	for _, repo := range result.Repositories {
		repositories = append(repositories, convertToRepositoryModel(repo))
	}
	
	return repositories, resp.NextPage, nil
}

// Helper functions

// convertToRepositoryModel converts a GitHub repository to our model
func convertToRepositoryModel(repo *github.Repository) *models.Repository {
	if repo == nil {
		return nil
	}
	
	r := &models.Repository{
		ID:            repo.GetID(),
		Name:          repo.GetName(),
		FullName:      repo.GetFullName(),
		Description:   repo.GetDescription(),
		Stars:         repo.GetStargazersCount(),
		Forks:         repo.GetForksCount(),
		Language:      repo.GetLanguage(),
		DefaultBranch: repo.GetDefaultBranch(),
		URL:           repo.GetURL(),
		HTMLURL:       repo.GetHTMLURL(),
	}
	
	if repo.GetCreatedAt().Time != (time.Time{}) {
		r.CreatedAt = repo.GetCreatedAt().Time
	}
	
	if repo.GetUpdatedAt().Time != (time.Time{}) {
		r.UpdatedAt = repo.GetUpdatedAt().Time
	}
	
	if repo.Topics != nil {
		r.Topics = repo.Topics
	}
	
	// Extract documentation URL if available
	if repo.GetHasWiki() {
		r.DocumentationURL = repo.GetHTMLURL() + "/wiki"
	}
	
	return r
}

// processGitHubError processes GitHub API errors
func processGitHubError(err error) error {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		if ghErr.Response.StatusCode == http.StatusNotFound {
			return errors.New("repository not found")
		}
		if ghErr.Response.StatusCode == http.StatusUnauthorized {
			return errors.New("unauthorized: invalid GitHub token")
		}
		if ghErr.Response.StatusCode == http.StatusForbidden {
			return errors.New("rate limit exceeded or access denied")
		}
	}
	return err
}

// isMarkdownFile checks for markdown file extensions
func isMarkdownFile(filename string) bool {
	lcFilename := strings.ToLower(filename)
	return strings.HasSuffix(lcFilename, ".md") || strings.HasSuffix(lcFilename, ".mdx")
}

// isDocumentationFile checks common documentation filenames
func isDocumentationFile(filename string) bool {
	lcFilename := strings.ToLower(filename)
	return lcFilename == "readme.md" ||
		lcFilename == "readme.mdx" ||
		lcFilename == "contributing.md" ||
		lcFilename == "license.md" || // Often contains useful info
		lcFilename == "code_of_conduct.md"
}

// searchForMarkdownFiles uses GitHub Search API to find markdown files in the repository
func (c *Client) searchForMarkdownFiles(ctx context.Context, owner, repo string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	
	query := fmt.Sprintf("repo:%s/%s extension:md", owner, repo)
	var allPaths []string
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
		TextMatch: false,
	}

	for {
		result, resp, err := c.client.Search.Code(ctx, query, opts)
		if err != nil {
			// Handle rate limit specifically
			if _, ok := err.(*github.RateLimitError); ok {
				log.Printf("Rate limit hit during search, trying again after reset: %v\n", resp.Rate.Reset)
				time.Sleep(time.Until(resp.Rate.Reset.Time))
				continue // Retry the same page
			}
			log.Printf("Error searching code: %v\n", err)
			return nil, processGitHubError(err) // Use centralized error processing
		}
	
		for _, item := range result.CodeResults {
			if item.Path != nil {
				if isMarkdownFile(*item.Path) || isDocumentationFile(*item.Name) {
					allPaths = append(allPaths, *item.Path)
				}
			}
		}
		
		if resp.NextPage == 0 {
			break // No more pages
		}
		opts.Page = resp.NextPage // Set the next page number
	}
	
	log.Printf("Found %d potential documentation files via search for %s/%s\n", len(allPaths), owner, repo)
	return allPaths, nil
}

// searchForMarkdownFilesInPath uses GitHub Search API to find markdown files in a specific path of the repository
func (c *Client) searchForMarkdownFilesInPath(ctx context.Context, owner, repo, path string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	
	query := fmt.Sprintf("repo:%s/%s path:%s extension:md", owner, repo, path)
	var allPaths []string
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
		TextMatch: false,
	}

	for {
		result, resp, err := c.client.Search.Code(ctx, query, opts)
		if err != nil {
			// Handle rate limit specifically
			if _, ok := err.(*github.RateLimitError); ok {
				log.Printf("Rate limit hit during search, trying again after reset: %v\n", resp.Rate.Reset)
				time.Sleep(time.Until(resp.Rate.Reset.Time))
				continue // Retry the same page
			}
			log.Printf("Error searching code: %v\n", err)
			return nil, processGitHubError(err) // Use centralized error processing
		}
	
		for _, item := range result.CodeResults {
			if item.Path != nil {
				if isMarkdownFile(*item.Path) || isDocumentationFile(*item.Name) {
					allPaths = append(allPaths, *item.Path)
				}
			}
		}
		
		if resp.NextPage == 0 {
			break // No more pages
		}
		opts.Page = resp.NextPage // Set the next page number
	}
	
	log.Printf("Found %d potential documentation files via search in path '%s' for %s/%s\n", len(allPaths), path, owner, repo)
	return allPaths, nil
}

// listDocFilesInPath is defined in docs_lister.go - REMOVING DUPLICATE DEFINITION
