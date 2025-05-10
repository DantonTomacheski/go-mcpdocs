# Componentes Principais do go-mcpdocs

O `go-mcpdocs` é composto por diversos módulos e estruturas que trabalham em conjunto para fornecer sua funcionalidade principal. Esta seção descreve os componentes mais importantes.

## 1. Configuração (`config.Config`)

A estrutura `config.Config` é fundamental para o funcionamento da aplicação, pois armazena todas as configurações necessárias carregadas do ambiente ou de um arquivo `.env`. 

*   **Responsabilidades:**
    *   Armazenar tokens de API (e.g., `GitHubToken`).
    *   Definir parâmetros de servidor (e.g., `Port`, `RequestTimeout`).
    *   Configurar o pool de workers (e.g., `WorkerPoolSize`).
    *   Gerenciar configurações de cache (e.g., `RedisURI`, `EnableCache`, `CacheTTL`).
*   **Observações:**
    *   Inclui valores padrão para garantir que a aplicação possa rodar mesmo com configurações mínimas.
    *   Realiza validações para garantir a integridade dos dados de configuração (e.g., portas válidas, timeouts positivos).

## 2. Cliente GitHub (`github.Client`)

O `github.Client` é o componente responsável por toda a comunicação com a API do GitHub.

*   **Responsabilidades:**
    *   Autenticar requisições para a API do GitHub.
    *   Buscar informações de repositórios.
    *   Listar arquivos e diretórios em um repositório.
    *   Obter o conteúdo de arquivos específicos.
    *   Gerenciar a paginação de resultados da API do GitHub.
    *   Implementar o método `GetRepositoryDocumentation` que orquestra a extração concorrente de arquivos de documentação.
*   **Observações:**
    *   Utiliza um cliente HTTP configurado (possivelmente com timeouts e retries).
    *   Trata erros da API do GitHub, convertendo-os para erros internos mais gerenciáveis.
    *   Pode incluir lógica para lidar com os limites de taxa (rate limits) da API do GitHub.

## 3. Cache (`cache.Cache` Interface e Implementações)

A interface `cache.Cache` define um contrato para mecanismos de caching, com o objetivo de reduzir a carga na API do GitHub e acelerar as respostas para requisições repetidas.

*   **Interface `cache.Cache`:**
    *   Define métodos como `Get(key string)` e `Set(key string, value interface{}, ttl time.Duration)`.
*   **Implementações Comuns (exemplos baseados no `memory.json`):**
    *   `cache.MemoryCache`: Uma implementação de cache em memória, simples e rápida para ambientes de desenvolvimento ou instâncias únicas.
    *   `cache.RedisCache`: Utiliza um servidor Redis (possivelmente via Upstash, como indicado no `memory.json`) para um cache distribuído, mais robusto para ambientes de produção e escaláveis.
*   **Observações:**
    *   A integração do cache é crucial para a performance, podendo reduzir significativamente o tempo de resposta.
    *   A chave de cache geralmente inclui o nome do repositório, owner e branch.

## 4. Modelos de Dados (`models`)

O pacote `models` contém as estruturas de dados que representam as entidades principais manipuladas pela aplicação.

*   **`models.Documentation`:**
    *   Representa um único arquivo de documentação extraído.
    *   Contém campos como `Path`, `Content`, `FileType`, `Size`, `URL`.
*   **`models.Repository`:**
    *   Representa as informações básicas de um repositório GitHub (nome, owner, descrição, URL, etc.).
*   **`models.RepositoryDocsResponse`:**
    *   Estrutura a resposta da API para o endpoint de obtenção de documentação.
    *   Agrega metadados do repositório (`RepositoryName`, `RepositoryOwner`, `Branch`, `CommitSHA`) e uma lista de `models.Documentation`, além da contagem de arquivos processados e informações de cache.

## 5. Processadores (`processor.Processor` Interface)

A interface `processor.Processor` (e suas implementações como `processor.MarkdownProcessor`, `processor.TextFormatter`) define como o conteúdo bruto dos arquivos de documentação é tratado e transformado.

*   **Responsabilidades:**
    *   Identificar o tipo de arquivo (e.g., Markdown, texto plano).
    *   Limpar, formatar ou converter o conteúdo (e.g., remover metadados indesejados, converter Markdown para HTML se necessário).
    *   Extrair informações relevantes, como exemplos de código.
*   **Observações:**
    *   O `processor.TextFormatter` é especificamente mencionado para formatar e possivelmente truncar o conteúdo, além de contar tokens.

## 6. Pool de Workers (`worker.WorkerPool`)

O `worker.WorkerPool` gerencia a execução concorrente de tarefas, especialmente a coleta de arquivos de documentação do GitHub.

*   **Responsabilidades:**
    *   Criar e gerenciar um número configurável de goroutines (workers).
    *   Distribuir tarefas (e.g., download de um arquivo) entre os workers disponíveis.
    *   Coletar os resultados das tarefas.
*   **Observações:**
    *   Essencial para a performance da aplicação, permitindo que múltiplos arquivos sejam processados em paralelo.
    *   Utiliza canais Go para comunicação e sincronização.

## 7. Manipulador da API (`api.Handler`)

A estrutura `api.Handler` é o coração da camada de API, responsável por lidar com as requisições HTTP recebidas.

*   **Responsabilidades:**
    *   Definir os métodos que correspondem aos endpoints da API (e.g., `GetRepositoryDocumentation`, `GetRepository`, `SearchRepositories`).
    *   Validar os parâmetros de entrada das requisições.
    *   Orquestrar a lógica de negócio, chamando outros componentes como `github.Client`, `cache.Cache`, e `repository.DocumentRepository`.
    *   Formatar as respostas HTTP em JSON.
    *   Gerenciar o ciclo de vida das requisições, incluindo o tratamento de timeouts e cancelamento de contexto.

Esses componentes, juntamente com outros módulos auxiliares (como o `database.Client` para persistência, se aplicável, e o `repository.DocumentRepository` para interagir com o armazenamento), formam a espinha dorsal do `go-mcpdocs`.
