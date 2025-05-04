package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dtomacheski/extract-data-go/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// MongoDB connection and collection constants
const (
	DatabaseName     = "deepwiki"
	DocsCollectionName = "documentation"
	DefaultTimeout   = 10 * time.Second
)

// DocStorage é uma versão leve da Documentation para armazenamento no DB
type DocStorage struct {
	RepoID        int64     `bson:"repo_id"`
	RepoName      string    `bson:"repo_name"`
	Filename      string    `bson:"filename"`      // Ex: llms.txt
	ProcessedPath string    `bson:"processed_path"` // Ex: /vercel/next.js/llms.txt
	ContentType   string    `bson:"content_type"`
	Size          int       `bson:"size"`
	SnippetsCount int       `bson:"snippets_count"`
	Content       string    `bson:"content"`       // Conteúdo processado em formato TXT
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`
}

// Client handles MongoDB operations
type Client struct {
	client     *mongo.Client
	database   *mongo.Database
	docs       *mongo.Collection
	timeout    time.Duration
	logger     *log.Logger
}

// NewClient creates a new MongoDB client
func NewClient(uri string, logger *log.Logger) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	// Configure the client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().
		ApplyURI(uri).
		SetServerAPIOptions(serverAPI)

	// Create a new client and connect to the server
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}

	// Verify the connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	database := client.Database(DatabaseName)
	docs := database.Collection(DocsCollectionName)

	return &Client{
		client:   client,
		database: database,
		docs:     docs,
		timeout:  DefaultTimeout,
		logger:   logger,
	}, nil
}

// Close disconnects the client
func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// StoreProcessedDocumentation armazena documentação processada no MongoDB
func (c *Client) StoreProcessedDocumentation(ctx context.Context, repoOwner, repoName string, filename string, content string, snippetsCount int) error {
	if content == "" {
		return errors.New("no content to store")
	}

	// Create the full path in the format /owner/repo/filename.txt
	processedPath := fmt.Sprintf("/%s/%s/%s", repoOwner, repoName, filename)

	// Create storage document
	storeDoc := DocStorage{
		RepoID:        0, // This could be set if needed
		RepoName:      fmt.Sprintf("%s/%s", repoOwner, repoName),
		Filename:      filename,
		ProcessedPath: processedPath,
		ContentType:   "text/plain",
		Size:          len(content),
		SnippetsCount: snippetsCount,
		Content:       content,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Insert the document
	_, err := c.docs.InsertOne(ctx, storeDoc)
	return err
}

// StoreDocumentation is kept for backward compatibility
func (c *Client) StoreDocumentation(ctx context.Context, docs []models.Documentation) error {
	c.logger.Println("Warning: Using deprecated StoreDocumentation method. Use StoreProcessedDocumentation instead.")
	return errors.New("method deprecated, use StoreProcessedDocumentation instead")
}

// GetDocumentationByRepoID retrieves documentation for a repository
func (c *Client) GetDocumentationByRepoID(ctx context.Context, repoID int64) ([]DocStorage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	filter := bson.D{{Key: "repo_id", Value: repoID}}
	cursor, err := c.docs.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []DocStorage
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// GetDocumentationByPath retrieves a specific document by path
func (c *Client) GetDocumentationByProcessedPath(ctx context.Context, processedPath string) (*DocStorage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	filter := bson.D{{Key: "processed_path", Value: processedPath}}
	var result DocStorage
	err := c.docs.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateDocumentation updates an existing document
func (c *Client) UpdateDocumentation(ctx context.Context, doc *DocStorage) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	filter := bson.D{{Key: "processed_path", Value: doc.ProcessedPath}}
	doc.UpdatedAt = time.Now()
	
	update := bson.D{{Key: "$set", Value: doc}}
	_, err := c.docs.UpdateOne(ctx, filter, update)
	return err
}

// DeleteDocumentation deletes a document
func (c *Client) DeleteDocumentation(ctx context.Context, processedPath string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	filter := bson.D{{Key: "processed_path", Value: processedPath}}
	_, err := c.docs.DeleteOne(ctx, filter)
	return err
}
