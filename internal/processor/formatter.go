package processor

import (
	"fmt"
	"strings"

	"github.com/dtomacheski/extract-data-go/internal/models"
)

// TextFormatter é responsável por formatar snippets de código no formato TXT desejado
type TextFormatter struct {
	// Configurações poderiam ser adicionadas aqui
}

// NewTextFormatter cria um novo formatador de texto
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{}
}

// FormatSnippetsToText formata uma lista de snippets de código para o formato TXT desejado
func (f *TextFormatter) FormatSnippetsToText(snippets []models.CodeSnippet) string {
	var sb strings.Builder
	
	for i, snippet := range snippets {
		// Adiciona o título
		sb.WriteString(fmt.Sprintf("TITLE: %s\n", snippet.Title))
		
		// Adiciona a descrição
		sb.WriteString(fmt.Sprintf("DESCRIPTION: %s\n", snippet.Description))
		
		// Adiciona a fonte
		sb.WriteString(fmt.Sprintf("SOURCE: %s\n", snippet.Source))
		sb.WriteString("\n")
		
		// Adiciona o código com a linguagem
		sb.WriteString(fmt.Sprintf("LANGUAGE: %s\n", snippet.Language))
		sb.WriteString("CODE:\n```\n")
		sb.WriteString(snippet.Code)
		sb.WriteString("\n```\n")
		sb.WriteString("\n")
		
		// Adiciona separador entre snippets, exceto para o último
		if i < len(snippets)-1 {
			sb.WriteString("----------------------------------------\n\n")
		}
	}
	
	return sb.String()
}

// GenerateFilename gera um nome de arquivo para o documento TXT baseado no repositório
func (f *TextFormatter) GenerateFilename(repoOwner, repoName string) string {
	// Simplifica o nome do repositório para uso em nome de arquivo
	simplifiedName := strings.ToLower(repoName)
	simplifiedName = strings.ReplaceAll(simplifiedName, ".", "-")
	simplifiedName = strings.ReplaceAll(simplifiedName, " ", "-")
	
	return fmt.Sprintf("%s-docs.txt", simplifiedName)
}

// ProcessAndFormatDocumentation processa a documentação e formata como TXT
func (f *TextFormatter) ProcessAndFormatDocumentation(docs []models.Documentation, repoOwner, repoName string) (string, string, int) {
	// Customizar o processador de documentos para usar URLs simplificados
	docProcessor := NewDocumentProcessor()
	
	// Reestruturar os documentos para usar apenas os arquivos da pasta docs
	var filteredDocs []models.Documentation
	for _, doc := range docs {
		// Verificar se o caminho parece ser da pasta docs
		if strings.HasPrefix(doc.Path, "docs/") {
			filteredDocs = append(filteredDocs, doc)
		}
	}
	
	// Usar documentos filtrados se houver, caso contrário usar todos
	docsToProcess := docs
	if len(filteredDocs) > 0 {
		docsToProcess = filteredDocs
	}
	
	// Extrai snippets da documentação
	baseRepoName := fmt.Sprintf("%s/%s", repoOwner, repoName)
	repoURL := fmt.Sprintf("https://github.com/%s", baseRepoName)
	docsResponse := docProcessor.ExtractSnippets(docsToProcess, baseRepoName, repoURL)
	
	// Simplificar os URLs de SOURCE
	for i := range docsResponse.Snippets {
		// Extrair apenas o caminho relativo ao repositório
		fullPath := docsResponse.Snippets[i].Source
		// Encontrar a posição após /blob/branch/ no URL
		parts := strings.Split(fullPath, "/blob/")
		if len(parts) > 1 {
			branchAndPath := parts[1]
			branchParts := strings.SplitN(branchAndPath, "/", 2)
			if len(branchParts) > 1 {
				// Formatar como /owner/repo/path
				docsResponse.Snippets[i].Source = fmt.Sprintf("/%s/%s/%s", repoOwner, repoName, branchParts[1])
			}
		}
	}
	
	// Formatar o texto
	formattedText := f.FormatSnippetsToText(docsResponse.Snippets)
	
	// Gerar o nome do arquivo
	filename := f.GenerateFilename(repoOwner, repoName)
	
	return filename, formattedText, len(docsResponse.Snippets)
}
