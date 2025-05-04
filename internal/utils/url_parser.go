package utils

import (
	"fmt"
	"strings"
)

// ExtractOwnerAndRepo extracts the owner and repository name from a GitHub URL.
func ExtractOwnerAndRepo(repoURL string) (string, string, error) {
	// Assume github.com URL for now
	cleanedURL := strings.TrimPrefix(repoURL, "https://")
	cleanedURL = strings.TrimPrefix(cleanedURL, "http://")
	parts := strings.Split(cleanedURL, "/")
	if len(parts) < 3 || parts[0] != "github.com" {
		return "", "", fmt.Errorf("invalid or unsupported GitHub URL format: %s", repoURL)
	}
	return parts[1], parts[2], nil
}
