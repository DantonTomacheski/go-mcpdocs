package repository

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/dtomacheski/extract-data-go/internal/database"
	"github.com/dtomacheski/extract-data-go/internal/models"
	"github.com/dtomacheski/extract-data-go/internal/processor"
)

// DocumentRepository handles document storage and retrieval operations
type DocumentRepository struct {
	mongoClient   *database.Client
	logger        *log.Logger
	enabled       bool
	textFormatter *processor.TextFormatter
}

// NewDocumentRepository creates a new document repository
func NewDocumentRepository(mongoClient *database.Client, logger *log.Logger) *DocumentRepository {
	enabled := mongoClient != nil

	return &DocumentRepository{
		mongoClient:   mongoClient,
		logger:        logger,
		enabled:       enabled,
		textFormatter: processor.NewTextFormatter(),
	}
}

// StoreDocumentation processa e armazena documentação no formato TXT no banco de dados
func (r *DocumentRepository) StoreDocumentation(ctx context.Context, docs []models.Documentation) error {
	if !r.enabled {
		r.logger.Println("MongoDB storage is disabled, skipping document storage")
		return nil
	}

	if len(docs) == 0 {
		return nil
	}

	// Extrair owner/repo do nome do primeiro documento
	if len(docs) == 0 || docs[0].RepoName == "" {
		return nil
	}

	// Obter owner/repo do RepoName (formato: owner/repo)
	repoParts := strings.Split(docs[0].RepoName, "/")
	if len(repoParts) != 2 {
		r.logger.Printf("Invalid repository name format: %s", docs[0].RepoName)
		return nil
	}

	repoOwner := repoParts[0]
	repoName := repoParts[1]

	// Processar e formatar a documentação
	filename, formattedText, snippetsCount := r.textFormatter.ProcessAndFormatDocumentation(docs, repoOwner, repoName)

	if snippetsCount == 0 {
		r.logger.Println("No snippets found in documentation, skipping storage")
		return nil
	}

	r.logger.Printf("Storing processed documentation with %d snippets in MongoDB as %s", snippetsCount, filename)

	// Armazenar no MongoDB usando a nova função
	return r.mongoClient.StoreProcessedDocumentation(ctx, repoOwner, repoName, filename, formattedText, snippetsCount)
}

// GetProcessedDocumentation recupera documentação processada pelo caminho
func (r *DocumentRepository) GetProcessedDocumentation(ctx context.Context, owner, repo string, filename string) (*database.DocStorage, error) {
	if !r.enabled {
		r.logger.Println("MongoDB storage is disabled, cannot retrieve documents")
		return nil, nil
	}

	// Construir o caminho processado
	processedPath := "/" + owner + "/" + repo + "/" + filename
	return r.mongoClient.GetDocumentationByProcessedPath(ctx, processedPath)
}

// GetDocumentationByRepoID ainda mantido para compatibilidade
func (r *DocumentRepository) GetDocumentationByRepoID(ctx context.Context, repoID int64) ([]database.DocStorage, error) {
	if !r.enabled {
		r.logger.Println("MongoDB storage is disabled, cannot retrieve documents")
		return nil, nil
	}

	return r.mongoClient.GetDocumentationByRepoID(ctx, repoID)
}

// IsEnabled returns whether MongoDB storage is enabled
func (r *DocumentRepository) IsEnabled() bool {
	return r.enabled
}

// GetLastUpdateTime returns the last update time for a repository's documentation
func (r *DocumentRepository) GetLastUpdateTime(ctx context.Context, owner, repo string) (*time.Time, error) {
	if !r.enabled {
		r.logger.Println("MongoDB storage is disabled, cannot check last update time")
		return nil, nil
	}

	// Search for documents with the repository path prefix
	path := "/" + owner + "/" + repo + "/"
	latestDoc, err := r.mongoClient.GetLatestDocumentForRepo(ctx, path)
	if err != nil {
		return nil, err
	}

	if latestDoc == nil {
		return nil, nil // No document found, never updated
	}

	return &latestDoc.UpdatedAt, nil
}

// CanRefreshRepository checks if a repository can be refreshed based on the minimum time between updates
func (r *DocumentRepository) CanRefreshRepository(ctx context.Context, owner, repo string, minDaysBetweenRefreshes int) (bool, time.Time, error) {
	if !r.enabled {
		// If MongoDB is disabled, always allow refresh
		return true, time.Time{}, nil
	}

	lastUpdate, err := r.GetLastUpdateTime(ctx, owner, repo)
	if err != nil {
		return false, time.Time{}, err
	}

	if lastUpdate == nil {
		// No previous update found, allow refresh
		return true, time.Time{}, nil
	}

	// Calculate the next allowed refresh time
	nextAllowedRefresh := lastUpdate.Add(time.Duration(minDaysBetweenRefreshes) * 24 * time.Hour)
	now := time.Now()

	if now.Before(nextAllowedRefresh) {
		// Too early to refresh
		return false, nextAllowedRefresh, errors.New("too early to refresh the project")
	}

	return true, time.Time{}, nil
}
