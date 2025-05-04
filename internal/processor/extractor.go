package processor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dtomacheski/extract-data-go/internal/models"
)

// DocumentProcessor processes documentation content to extract code snippets
type DocumentProcessor struct {
	// Configuration options could go here in the future
	// For example: snippet limit, min/max snippet size, etc.
}

// NewDocumentProcessor creates a new document processor
func NewDocumentProcessor() *DocumentProcessor {
	return &DocumentProcessor{}
}

// ExtractSnippets extracts code snippets from documentation content
func (p *DocumentProcessor) ExtractSnippets(docs []models.Documentation, repoName, repoURL string) models.DocumentationResponse {
	var allSnippets []models.CodeSnippet
	processedFiles := 0

	for _, doc := range docs {
		// Skip empty content
		if doc.Content == "" {
			continue
		}

		// Process the document to extract snippets
		fileSnippets := p.processDocument(doc, repoName, repoURL)
		allSnippets = append(allSnippets, fileSnippets...)

		if len(fileSnippets) > 0 {
			processedFiles++
		}
	}

	// Create the response
	return models.DocumentationResponse{
		RepositoryName: repoName,
		RepositoryURL:  repoURL,
		TotalSnippets:  len(allSnippets),
		TotalFiles:     processedFiles,
		Snippets:       allSnippets,
	}
}

// processDocument processes a single document to extract code snippets
func (p *DocumentProcessor) processDocument(doc models.Documentation, repoName, repoURL string) []models.CodeSnippet {
	var snippets []models.CodeSnippet

	// Process Markdown documents
	if strings.HasSuffix(strings.ToLower(doc.Path), ".md") || strings.HasSuffix(strings.ToLower(doc.Path), ".mdx") {
		snippets = append(snippets, p.extractMarkdownSnippets(doc, repoName, repoURL)...)
	}

	// In the future, we could add support for other document types here

	return snippets
}

// extractMarkdownSnippets extracts code snippets from markdown content
func (p *DocumentProcessor) extractMarkdownSnippets(doc models.Documentation, repoName, repoURL string) []models.CodeSnippet {
	var snippets []models.CodeSnippet

	// Regex to find code blocks with language
	codeBlockRegex := regexp.MustCompile("```([a-zA-Z0-9]*)\n([\\s\\S]*?)```")
	matches := codeBlockRegex.FindAllStringSubmatch(doc.Content, -1)

	// Extract document title
	title := extractTitle(doc.Content)
	
	// Calculate source URL
	sourceURL := fmt.Sprintf("%s/blob/master/%s", repoURL, doc.Path)
	if repoURL == "https://github.com/vercel/next.js" {
		sourceURL = fmt.Sprintf("%s/blob/canary/%s", repoURL, doc.Path)
	}

	// Process each code block
	for i, match := range matches {
		if len(match) < 3 {
			continue
		}

		language := match[1]
		code := strings.TrimSpace(match[2])

		// Skip empty code blocks
		if code == "" {
			continue
		}

		// We'll use the file name and position to create more meaningful titles
		snippetNum := i+1

		// Try to find a better description from the content around the code block
		description := extractDescriptionForCodeBlock(doc.Content, match[0])

		// Create the snippet
		snippet := models.CodeSnippet{
			Title:       fmt.Sprintf("%s (%s) - Snippet %d", title, repoName, snippetNum),
			Description: description,
			Source:      sourceURL,
			Language:    language,
			Code:        code,
		}

		snippets = append(snippets, snippet)
	}

	return snippets
}

// extractTitle extracts a title from markdown content
func extractTitle(content string) string {
	// Look for h1 headers
	h1Regex := regexp.MustCompile(`(?m)^#\s+(.+)$`)
	h1Matches := h1Regex.FindStringSubmatch(content)
	if len(h1Matches) > 1 {
		return strings.TrimSpace(h1Matches[1])
	}

	// If no h1, look for h2
	h2Regex := regexp.MustCompile(`(?m)^##\s+(.+)$`)
	h2Matches := h2Regex.FindStringSubmatch(content)
	if len(h2Matches) > 1 {
		return strings.TrimSpace(h2Matches[1])
	}

	// If no headers, use the first non-empty line
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Limit length
			if len(line) > 50 {
				return line[:47] + "..."
			}
			return line
		}
	}

	return "Untitled Document"
}

// extractDescriptionForCodeBlock tries to extract a description for a code block
func extractDescriptionForCodeBlock(content, codeBlock string) string {
	// Find the position of the code block
	blockPos := strings.Index(content, codeBlock)
	if blockPos == -1 {
		return "Code snippet from documentation"
	}

	// Get content before the code block (up to 500 chars)
	startPos := blockPos - 500
	if startPos < 0 {
		startPos = 0
	}
	beforeBlock := content[startPos:blockPos]

	// Look for paragraphs before the code block
	paragraphs := strings.Split(beforeBlock, "\n\n")
	if len(paragraphs) > 0 {
		// Get the last paragraph
		lastPara := strings.TrimSpace(paragraphs[len(paragraphs)-1])
		
		// Remove any markdown formatting
		lastPara = cleanMarkdownFormatting(lastPara)
		
		// Limit length
		if len(lastPara) > 120 {
			return lastPara[:117] + "..."
		}
		
		if lastPara != "" {
			return lastPara
		}
	}

	return "Code snippet from documentation"
}

// cleanMarkdownFormatting removes common markdown formatting
func cleanMarkdownFormatting(text string) string {
	// Remove headers
	text = regexp.MustCompile(`(?m)^#+\s+`).ReplaceAllString(text, "")
	
	// Remove bold/italic
	text = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`\*(.+?)\*`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`__(.+?)__`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`_(.+?)_`).ReplaceAllString(text, "$1")
	
	// Remove links but keep text
	text = regexp.MustCompile(`\[(.+?)\]\(.+?\)`).ReplaceAllString(text, "$1")
	
	return text
}
