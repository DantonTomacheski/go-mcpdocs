package repository

import (
	"context"
	"log"
	"strings"

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
