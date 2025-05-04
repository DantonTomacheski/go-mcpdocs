package github

import (
	"context"
	"log"
	"path"
	"strings"

	"github.com/google/go-github/v53/github"
)

// listDocFilesInPath lista recursivamente todos os arquivos markdown na pasta especificada
func (c *Client) listDocFilesInPath(ctx context.Context, owner, repo, docPath, branch string) ([]string, error) {
	var allFiles []string
	
	// Inicia a busca recursiva a partir do caminho de documentação fornecido
	err := c.listFilesRecursively(ctx, owner, repo, docPath, branch, &allFiles)
	if err != nil {
		return nil, err
	}
	
	// Filtrar apenas arquivos markdown e outros arquivos de documentação
	var docFiles []string
	for _, file := range allFiles {
		if isMarkdownFile(file) || isDocumentationFile(file) {
			docFiles = append(docFiles, file)
		}
	}
	
	return docFiles, nil
}

// listDocsDirectoryFiles é mantido para compatibilidade (chama listDocFilesInPath com "docs")
func (c *Client) listDocsDirectoryFiles(ctx context.Context, owner, repo, branch string) ([]string, error) {
	return c.listDocFilesInPath(ctx, owner, repo, "docs", branch)
}

// listFilesRecursively busca recursivamente todos os arquivos em um diretório e seus subdiretórios
func (c *Client) listFilesRecursively(ctx context.Context, owner, repo, dirPath, branch string, files *[]string) error {
	opts := &github.RepositoryContentGetOptions{Ref: branch}
	
	// Obtém o conteúdo do diretório
	_, contents, resp, err := c.client.Repositories.GetContents(ctx, owner, repo, dirPath, opts)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("Directory not found: %s\n", dirPath)
			return nil
		}
		return err
	}
	
	// Processa cada item no diretório
	for _, content := range contents {
		contentPath := *content.Path
		
		// Se for um diretório, busca recursivamente
		if *content.Type == "dir" {
			err := c.listFilesRecursively(ctx, owner, repo, contentPath, branch, files)
			if err != nil {
				log.Printf("Error listing directory %s: %v\n", contentPath, err)
				// Continua mesmo com erro para tentar obter o máximo de arquivos possível
				continue
			}
		} else if *content.Type == "file" {
			// Adiciona à lista de arquivos
			*files = append(*files, contentPath)
		}
	}
	
	return nil
}

// isDirectDocumentationFile verifica se o arquivo é de documentação direta (README, docs, guias)
func isDirectDocumentationFile(name string) bool {
	fileName := strings.ToLower(path.Base(name))
	return strings.HasPrefix(fileName, "readme") || 
		   strings.HasPrefix(fileName, "guide") || 
		   strings.HasPrefix(fileName, "documentation") || 
		   strings.HasPrefix(fileName, "docs") ||
		   strings.HasPrefix(fileName, "tutorial")
}
