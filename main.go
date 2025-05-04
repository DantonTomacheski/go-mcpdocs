package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dtomacheski/extract-data-go/api"
	"github.com/dtomacheski/extract-data-go/config"
	"github.com/dtomacheski/extract-data-go/internal/database"
	"github.com/dtomacheski/extract-data-go/internal/github"
	"github.com/dtomacheski/extract-data-go/internal/repository"
)

func main() {
	// Initialize logger
	logger := log.New(os.Stdout, "[GITHUB-DOC-API] ", log.LstdFlags)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize GitHub client
	githubClient := github.NewClient(cfg.GitHubToken, cfg.RequestTimeout)

	// Initialize MongoDB client if enabled
	var mongoClient *database.Client
	if cfg.EnableMongoDB {
		logger.Println("Initializing MongoDB connection...")
		mongoClient, err = database.NewClient(cfg.MongoURI, logger)
		if err != nil {
			logger.Fatalf("Failed to connect to MongoDB: %v", err)
		}
		logger.Println("Successfully connected to MongoDB")
		
		// Ensure MongoDB client is closed on shutdown
		defer func() {
			if err := mongoClient.Close(context.Background()); err != nil {
				logger.Printf("Error closing MongoDB connection: %v", err)
			}
		}()
	} else {
		logger.Println("MongoDB integration disabled - no connection string provided")
	}

	// Initialize document repository
	docRepo := repository.NewDocumentRepository(mongoClient, logger)

	// Initialize API handler
	handler := api.NewHandler(githubClient, docRepo, logger, cfg.WorkerPoolSize)

	// Set up router
	router := api.SetupRouter(handler)

	// Configure HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Printf("Starting server on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Println("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server exited gracefully")
}
