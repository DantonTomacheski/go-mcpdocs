package github

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestGetRepository(t *testing.T) {
	// Load environment variables from .env file
	if err := godotenv.Load("../../.env"); err != nil {
		t.Skip("Skipping test: .env file not found or token not set")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" || token == "your_github_token_here" {
		t.Skip("Skipping test: GITHUB_TOKEN not set")
	}

	// Initialize client
	client := NewClient(token, 30*time.Second)

	// Test with a known repository
	repo, err := client.GetRepository(context.Background(), "google", "go-github")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	// Assert basic properties
	if repo == nil {
		t.Fatal("Repository should not be nil")
	}
	if repo.Name != "go-github" {
		t.Errorf("Expected repository name 'go-github', got '%s'", repo.Name)
	}
	if repo.FullName != "google/go-github" {
		t.Errorf("Expected full name 'google/go-github', got '%s'", repo.FullName)
	}
}

func TestGetRepositoryDocumentation(t *testing.T) {
	// Load environment variables from .env file
	if err := godotenv.Load("../../.env"); err != nil {
		t.Skip("Skipping test: .env file not found or token not set")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" || token == "your_github_token_here" {
		t.Skip("Skipping test: GITHUB_TOKEN not set")
	}

	// Initialize client
	client := NewClient(token, 30*time.Second)

	// Test with a known repository that has documentation
	docs, err := client.GetRepositoryDocumentation(context.Background(), "google", "go-github", "", 3)
	if err != nil {
		t.Fatalf("Failed to get repository documentation: %v", err)
	}

	// Assert basic properties
	if docs == nil {
		t.Fatal("Documentation should not be nil")
	}
	if len(docs) == 0 {
		t.Fatal("Documentation slice should not be empty")
	}

	// Check if README.md was found
	foundReadme := false
	for _, doc := range docs {
		if doc.Path == "README.md" {
			foundReadme = true
			break
		}
	}
	if !foundReadme {
		t.Error("README.md should be included in documentation")
	}
}
