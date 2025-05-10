# Fluxo de Dados no go-mcpdocs

Entender como os dados fluem através do `go-mcpdocs` é crucial para compreender sua arquitetura e funcionamento. Esta seção descreve o ciclo de vida de uma requisição típica para obter a documentação de um repositório.

## Fluxo Típico: Requisição de Documentação de Repositório

Vamos considerar uma requisição `GET /repository/:owner/:repo/docs`:

1.  **Chegada da Requisição:**
    *   O servidor web (construído com Gin) recebe a requisição HTTP.
    *   O roteador (`api.SetupRouter`) direciona a requisição para o método apropriado no `api.Handler`, que neste caso seria `api.Handler.GetRepositoryDocumentation`.
    *   Middlewares configurados (como `api.corsMiddleware` e `api.timeoutMiddleware`) são executados para tratar de CORS e timeouts, respectivamente.

2.  **Processamento no `api.Handler.GetRepositoryDocumentation`:**
    *   O handler extrai os parâmetros da URL (`owner`, `repo`) e quaisquer parâmetros de query (e.g., `branch`).
    *   **Verificação de Cache (Se Habilitado):**
        *   O handler primeiro consulta o componente de cache (`cache.Cache` - e.g., `cache.RedisCache` ou `cache.MemoryCache`) para verificar se os dados da documentação para este repositório e branch já existem e são válidos.
        *   Se uma entrada válida é encontrada no cache (cache hit), os dados são recuperados do cache, e o sistema pode pular para a etapa de formatação da resposta (Etapa 6).

3.  **Busca de Dados (Cache Miss):**
    *   Se os dados não estão no cache ou estão expirados (cache miss), o `api.Handler` precisa buscar as informações.
    *   O `api.Handler.GetRepositoryDocumentation` invoca o método `github.Client.GetRepositoryDocumentation` (ou um método similar no `repository.DocumentRepository` que, por sua vez, usaria o `github.Client`).

4.  **Interação com o GitHub (`github.Client`):**
    *   O `github.Client.GetRepositoryDocumentation` é responsável por:
        *   Comunicar-se com a API do GitHub para obter a lista de arquivos do repositório (e branch especificado).
        *   Identificar quais desses arquivos são considerados arquivos de documentação (e.g., `README.md`, arquivos em pastas `docs/`, etc., usando métodos auxiliares como `github.Client.isDocumentationFile` ou `github.Client.listDocFilesInPath` que pode chamar `github.Client.listFilesRecursively`).
        *   Para cada arquivo de documentação identificado, obter seu conteúdo bruto da API do GitHub (usando `github.Client.getFileContent`). Esta etapa é frequentemente otimizada usando o `worker.WorkerPool` para buscar múltiplos arquivos concorrentemente.

5.  **Processamento e Formatação (`processor`):**
    *   O conteúdo bruto de cada arquivo de documentação obtido pelo `github.Client` é então passado para um componente `processor` (e.g., `processor.TextFormatter` ou uma implementação de `processor.Processor`).
    *   O processador pode:
        *   Limpar o texto (remover caracteres indesejados).
        *   Formatar o texto (e.g., truncar para um tamanho máximo, contar tokens).
        *   Converter o formato (e.g., Markdown para HTML, se essa funcionalidade for implementada).
        *   Extrair metadados adicionais.
    *   Os dados processados de cada arquivo são encapsulados em estruturas `models.Documentation`.

6.  **Armazenamento em Cache (Se Habilitado e Cache Miss):**
    *   Após obter e processar os dados do GitHub, o `api.Handler` (ou o `repository.DocumentRepository`) armazena o resultado (`models.RepositoryDocsResponse` contendo a lista de `models.Documentation` e metadados) no cache (`cache.Cache.Set`) com um TTL (Time To Live) definido na configuração (`config.Config.CacheTTL`).
    *   Isso garante que requisições subsequentes para o mesmo repositório/branch possam ser servidas mais rapidamente.

7.  **Formatação da Resposta:**
    *   O `api.Handler` monta a estrutura final da resposta, geralmente um objeto `models.RepositoryDocsResponse`. Este objeto inclui detalhes do repositório, a lista dos arquivos de documentação processados (`models.Documentation`), a contagem de arquivos processados, e possivelmente informações sobre o cache.

8.  **Envio da Resposta:**
    *   O `api.Handler` envia a resposta JSON com o status HTTP apropriado (e.g., `200 OK`) de volta para o cliente.

## Outros Fluxos de Dados

*   **Requisição de Informações do Repositório (`GET /repository/:owner/:repo`):** Segue um fluxo similar, mas focado apenas em buscar metadados do repositório via `github.Client`, possivelmente com cache, e retornando um `models.Repository`.
*   **Busca de Repositórios (`GET /search/repositories`):** Envolve o `api.Handler.SearchRepositories` chamando o `github.Client` para usar o endpoint de busca da API do GitHub, tratando paginação e retornando uma lista de `models.Repository` com metadados de paginação.

## Componentes de Suporte

*   **`config.Config`:** Fornece configurações (tokens, timeouts, tamanho do pool de workers, configurações de cache) que influenciam todos os estágios do fluxo.
*   **`main.main`:** Inicia a aplicação, carrega a configuração, inicializa o `api.Handler` com suas dependências (como `github.Client`, `cache.Cache`), e configura o roteador Gin (`api.SetupRouter`).

Este fluxo ilustra a interação entre os principais componentes do `go-mcpdocs`, destacando o papel do cache, do cliente GitHub, dos processadores e do pool de workers na entrega eficiente da documentação solicitada.
