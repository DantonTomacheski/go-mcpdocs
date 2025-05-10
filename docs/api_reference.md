# Referência da API go-mcpdocs

Esta seção detalha os endpoints disponíveis na API `go-mcpdocs`, como fazer requisições e os formatos de resposta esperados.

## Convenções

*   Todas as respostas da API são em formato JSON.
*   Em caso de sucesso, a API geralmente retorna o status HTTP `200 OK`.
*   Erros são indicados por códigos HTTP apropriados (e.g., `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`).

## Endpoints

### 1. Obter Documentação de um Repositório

Recupera os arquivos de documentação processados de um repositório específico do GitHub.

*   **Endpoint:** `GET /repository/:owner/:repo/docs`
*   **Método HTTP:** `GET`
*   **Parâmetros de URL:**
    *   `owner` (string, obrigatório): O nome do proprietário (usuário ou organização) do repositório no GitHub.
    *   `repo` (string, obrigatório): O nome do repositório no GitHub.
*   **Parâmetros de Query (Opcionais):**
    *   `branch` (string): O nome do branch específico do qual extrair a documentação. Se não fornecido, o branch padrão do repositório é utilizado.
*   **Resposta de Sucesso (Código `200 OK`):**
    ```json
    {
      "repository_name": "nome-do-repositorio",
      "repository_owner": "nome-do-proprietario",
      "branch": "main",
      "commit_sha": "abcdef1234567890", // SHA do commit do qual a documentação foi extraída
      "documentation_files": [
        {
          "path": "README.md",
          "content": "Conteúdo do README.md...",
          "file_type": "markdown",
          "size": 1234,
          "url": "https://github.com/owner/repo/blob/main/README.md"
        },
        {
          "path": "docs/guide.md",
          "content": "Conteúdo do guide.md...",
          "file_type": "markdown",
          "size": 5678,
          "url": "https://github.com/owner/repo/blob/main/docs/guide.md"
        }
        // ... outros arquivos de documentação
      ],
      "files_processed_count": 2,
      "cached_at": "2023-10-27T10:30:00Z", // Opcional, se cache estiver habilitado
      "cache_expires_at": "2023-10-27T11:30:00Z" // Opcional, se cache estiver habilitado
    }
    ```
    *Nota: A estrutura exata da resposta, especialmente `repository_name`, `repository_owner`, `branch`, `commit_sha`, `cached_at`, `cache_expires_at` é baseada no modelo `models.RepositoryDocsResponse` e pode incluir metadados adicionais.*

### 2. Obter Informações de um Repositório

Recupera informações básicas sobre um repositório do GitHub.

*   **Endpoint:** `GET /repository/:owner/:repo`
*   **Método HTTP:** `GET`
*   **Parâmetros de URL:**
    *   `owner` (string, obrigatório): O nome do proprietário do repositório.
    *   `repo` (string, obrigatório): O nome do repositório.
*   **Resposta de Sucesso (Código `200 OK`):**
    ```json
    {
      "name": "nome-do-repositorio",
      "owner": "nome-do-proprietario",
      "description": "Descrição do repositório.",
      "url": "https://github.com/owner/repo",
      "default_branch": "main",
      "stars": 150,
      "forks": 30,
      "last_updated": "2023-10-26T14:00:00Z"
      // ... outras informações básicas do repositório (baseado em models.Repository)
    }
    ```

### 3. Buscar Repositórios

Busca repositórios no GitHub com base em uma query.

*   **Endpoint:** `GET /search/repositories`
*   **Método HTTP:** `GET`
*   **Parâmetros de Query:**
    *   `query` (string, obrigatório): O termo de busca para os repositórios.
    *   `page` (integer, opcional, padrão: `1`): O número da página dos resultados.
    *   `per_page` (integer, opcional, padrão: `10`): O número de resultados por página.
*   **Resposta de Sucesso (Código `200 OK`):**
    ```json
    {
      "total_count": 120,
      "items": [
        {
          "name": "repo1",
          "owner": "owner1",
          "description": "Descrição do repo1",
          "url": "https://github.com/owner1/repo1",
          "stars": 50
        },
        {
          "name": "another-repo",
          "owner": "userX",
          "description": "Outro repositório interessante",
          "url": "https://github.com/userX/another-repo",
          "stars": 75
        }
        // ... outros repositórios
      ],
      "pagination": {
        "current_page": 1,
        "per_page": 10,
        "total_pages": 12,
        "next_page_url": "/search/repositories?query=...&page=2&per_page=10" // Exemplo
      }
    }
    ```

### 4. Verificação de Saúde (Health Check)

Verifica o status da aplicação.

*   **Endpoint:** `GET /health`
*   **Método HTTP:** `GET`
*   **Resposta de Sucesso (Código `200 OK`):**
    ```json
    {
      "status": "UP",
      "timestamp": "2023-10-27T10:35:00Z",
      "services": {
        "github_api": "OPERATIONAL",
        "cache": "OPERATIONAL" // Se o cache estiver habilitado
      }
    }
    ```
    *Nota: O conteúdo exato da resposta de health check pode variar conforme a implementação.*
