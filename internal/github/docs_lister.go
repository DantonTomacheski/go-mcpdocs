package github

import (
	"context"
	"log"

	"github.com/google/go-github/v53/github"
)



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


