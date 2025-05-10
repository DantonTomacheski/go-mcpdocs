package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dtomacheski/extract-data-go/internal/models"
	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
	"path/filepath"
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
// If specificRef is empty, it defaults to the defaultBranchFromHandler.
func (c *Client) GetRepositoryDocumentation(ctx context.Context, owner, repo, defaultBranchFromHandler, specificRef string, concurrencyLimit int) ([]models.Documentation, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Define variable err at the function level so it's available throughout the function
	var err error

	// Determine the ref to use (provided specificRef or defaultBranchFromHandler)
	var refToUse string
	if specificRef == "" {
		// No specificRef provided, use defaultBranchFromHandler
		if defaultBranchFromHandler == "" {
			log.Printf("Error: Default branch not provided and specific ref is empty for %s/%s\n", owner, repo)
			return nil, fmt.Errorf("default branch not provided for repository %s/%s and no specific ref given", owner, repo)
		}
		refToUse = defaultBranchFromHandler
		log.Printf("No specific ref provided, using default branch '%s' from handler for %s/%s", refToUse, owner, repo)
	} else {
		// Use the provided specificRef
		refToUse = specificRef
		log.Printf("Using provided ref '%s' for %s/%s", refToUse, owner, repo)
	}

	log.Printf("Attempting to fetch documentation for %s/%s from ref '%s'", owner, repo, refToUse)

	// Attempt to get the commit for the refToUse to get the tree SHA
	var rootTreeSHA string
	commit, _, err := c.client.Repositories.GetCommit(ctx, owner, repo, refToUse, nil)
	if err != nil {
		log.Printf("Error getting commit for ref %s in %s/%s: %v", refToUse, owner, repo, err)
		// Proceed without tree SHA if commit fetch fails, relying on search code logic
	} else if commit != nil && commit.Commit != nil && commit.Commit.Tree != nil && commit.Commit.Tree.SHA != nil {
		rootTreeSHA = *commit.Commit.Tree.SHA
		log.Printf("Successfully obtained root tree SHA: %s for ref %s", rootTreeSHA, refToUse)
	} else {
		log.Printf("Commit or tree SHA is nil for ref %s in %s/%s", refToUse, owner, repo)
	}

	var docPaths []string
	searchInDocs := false

	// Define caminhos comuns de documentação em diferentes repositórios
	commonDocPaths := []string{
		"docs",          // Caminho padrão (Next.js, Vue, etc)
		"src/content",   // React.dev
		"src/docs",      // tailwind.css
		"Documentation", // Swift, alguns projetos da Apple
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
				// Consider this a potentially transient error or a configuration issue, but allow checking other common paths.
			}
		}
	}

	// Se nenhum caminho comum foi encontrado, tente procurar na raiz por pastas de documentação padrão.
	if !searchInDocs {
		log.Printf("Nenhum caminho de documentação comum encontrado para %s/%s no ref '%s'. Tentando buscar na raiz...\n", owner, repo, refToUse)
		rootContentsOpts := &github.RepositoryContentGetOptions{Ref: refToUse}
		_, rootDirContents, _, rootErr := c.client.Repositories.GetContents(ctx, owner, repo, "", rootContentsOpts)

		if rootErr != nil {
			log.Printf("Erro ao listar conteúdo da raiz de %s/%s no ref '%s': %v\n", owner, repo, refToUse, rootErr)
			// Não retorna erro aqui, pois o comportamento final de 'pasta não encontrada' será tratado mais abaixo.
		} else {
			standardRootDocFolders := []string{"docs", "documentation", "doc"}
			for _, item := range rootDirContents {
				if item.GetType() == "dir" {
					for _, standardFolder := range standardRootDocFolders {
						if strings.EqualFold(item.GetName(), standardFolder) {
							log.Printf("Pasta de documentação '%s' encontrada na raiz de %s/%s no ref '%s'.\n", item.GetName(), owner, repo, refToUse)
							foundDocPath = item.GetName()
							searchInDocs = true
							break // Sai do loop de standardRootDocFolders
						}
					}
				}
				if searchInDocs {
					break // Sai do loop de rootDirContents, pois já encontrou uma pasta
				}
			}
			if !searchInDocs {
				log.Printf("Nenhuma pasta de documentação padrão (docs, documentation, doc) encontrada na raiz de %s/%s no ref '%s'.\n", owner, repo, refToUse)
			}
		}
	}

	// If after checking common paths and specific root folders, no documentation path is set,
	// and we have a rootTreeSHA, try to find all documentation files using the Git Tree API as a fallback.
	if !searchInDocs && rootTreeSHA != "" {
		log.Printf("No specific documentation folder found for %s/%s. Attempting to find all .md/.mdx files via Git Tree API using tree SHA %s", owner, repo, rootTreeSHA)
		pathsFromTree, treeErr := c._getDocPathsFromTree(ctx, owner, repo, rootTreeSHA)
		if treeErr != nil {
			log.Printf("Error using Git Tree API for %s/%s (tree %s): %v. Proceeding without these results.", owner, repo, rootTreeSHA, treeErr)
		} else if len(pathsFromTree) > 0 {
			log.Printf("Found %d documentation files via Git Tree API for %s/%s.", len(pathsFromTree), owner, repo)
			docPaths = pathsFromTree
			// When using Git Tree API for a global search, we assume all found .md/.mdx files are desired.
			// The 'searchInDocs' flag and 'foundDocPath' might not be relevant in the same way,
			// as we are not limiting to a sub-folder. The content fetching loop below will handle these paths.
			// We can set searchInDocs to true to indicate docs were found, even if not in a specific 'docs' folder.
			searchInDocs = true // Mark that we found docs, so the next section processes them.
		} else {
			log.Printf("No .md/.mdx files found via Git Tree API for %s/%s (tree %s).", owner, repo, rootTreeSHA)
		}
	}

	// 2. Perform search based on whether a valid documentation path exists
	if searchInDocs {
		// If docPaths is already populated by Git Tree API, this block will be skipped if foundDocPath is empty.
		// We need to ensure that if docPaths has items from Tree API, we proceed to fetch content for them.
		// The current structure uses 'foundDocPath' to list files within that specific path.
		// If Tree API was used, 'docPaths' contains full paths and 'foundDocPath' might be empty.

		// If 'foundDocPath' is set (meaning a specific 'docs' folder etc. was identified and preferred),
		// and docPaths is not already populated by a global tree search, list files in that specific path.
		if foundDocPath != "" && len(docPaths) == 0 { // Only if docPaths not already set by Tree API
			log.Printf("Listando arquivos de documentação na pasta %s de %s/%s no ref '%s'...\n", foundDocPath, owner, repo, refToUse)
			// Usar nossa função de listagem recursiva no foundDocPath
			err := c.listFilesRecursively(ctx, owner, repo, foundDocPath, refToUse, &docPaths)
			if err != nil {
				log.Printf("Erro ao listar arquivos de documentação na pasta %s para %s/%s: %v\n", foundDocPath, owner, repo, err)
				// Decidir se retorna erro ou continua com busca global se aplicável
				// return nil, fmt.Errorf("failed to list documentation files in %s: %w", foundDocPath, err)
			} else {
				log.Printf("Successfully listed files in %s, docPaths count: %d", foundDocPath, len(docPaths))
			}
		} else if len(docPaths) > 0 {
			log.Printf("Proceeding with %d paths found (possibly from Git Tree API or prior logic) for %s/%s on ref '%s'", len(docPaths), owner, repo, refToUse)
		} else {
			// This case means searchInDocs was true, but foundDocPath was empty, and docPaths is also empty.
			// This might happen if searchInDocs was set true by Tree API but it returned no paths.
			// Or if a docs folder was identified at root but listFilesRecursively failed or returned empty and Tree API also failed/empty.
			log.Printf("Warning: searchInDocs is true but no docPaths were ultimately populated for %s/%s on ref '%s'. No documentation will be fetched.", owner, repo, refToUse)
			// No specific error, but no docs will be processed. Client will get an empty list.
		}

	} else {
		// searchInDocs is false, meaning no 'docs' folder found and Tree API fallback was not used or yielded nothing.
		// Attempt a final global search for any .md/.mdx files in the repo if not done by Tree API already.
		// This is similar to what Tree API does, but using searchForMarkdownFiles.
		// Only do this if rootTreeSHA was not available (so Tree API wasn't attempted) OR if we want it as an ultimate fallback.
		if rootTreeSHA == "" { // Only if Tree API wasn't already tried
			log.Printf("Nenhuma pasta de documentação encontrada e Git Tree API not used/failed. Tentando busca global por arquivos .md/.mdx em %s/%s no ref '%s'...\n", owner, repo, refToUse)
			// Note: searchForMarkdownFiles currently searches the default branch. If refToUse is critical here,
			// searchForMarkdownFiles would need modification to include the ref in its query string.
			// For now, we use it as is, which might be acceptable for a broad fallback.
			globalPaths, err := c.searchForMarkdownFiles(ctx, owner, repo) // refToUse and extensions are implicit or handled within the method
			if err != nil {
				log.Printf("Erro na busca global por arquivos de documentação para %s/%s: %v\n", owner, repo, err)
				// Não retorna erro fatal aqui, pode ser que o repositório não tenha docs
			} else {
				docPaths = append(docPaths, globalPaths...)
			}
		}
	}

	// Check if any documentation files were found
	if len(docPaths) == 0 {
		log.Printf("No documentation files found for %s/%s on ref '%s'.\n", owner, repo, refToUse)
		return nil, errors.New("no documentation files found")
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

// _getDocPathsFromTree fetches all documentation file paths from a repository using the Git Tree API.
// It filters for .md and .mdx files.
func (c *Client) _getDocPathsFromTree(ctx context.Context, owner, repo, treeSHA string) ([]string, error) {
	if treeSHA == "" {
		return nil, fmt.Errorf("treeSHA cannot be empty for _getDocPathsFromTree")
	}

	log.Printf("Fetching Git tree for %s/%s using SHA: %s", owner, repo, treeSHA)
	tree, _, err := c.client.Git.GetTree(ctx, owner, repo, treeSHA, true) // true for recursive
	if err != nil {
		return nil, fmt.Errorf("failed to get git tree for %s/%s (SHA: %s): %w", owner, repo, treeSHA, err)
	}

	if tree.GetTruncated() {
		log.Printf("Warning: Git tree for %s/%s (SHA: %s) was truncated. Some files may be missing.", owner, repo, treeSHA)
		// Consider if we need to handle this more actively, e.g., by returning an error or specific info.
	}

	var docPaths []string
	docExtensions := map[string]bool{".md": true, ".mdx": true}

	for _, entry := range tree.Entries {
		if entry.GetType() == "blob" {
			ext := strings.ToLower(filepath.Ext(entry.GetPath()))
			if docExtensions[ext] {
				docPaths = append(docPaths, entry.GetPath())
			}
		}
	}

	log.Printf("Found %d documentation files (.md, .mdx) via Git Tree API for %s/%s (tree %s)", len(docPaths), owner, repo, treeSHA)
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



// listDocFilesInPath is defined in docs_lister.go - REMOVING DUPLICATE DEFINITION
