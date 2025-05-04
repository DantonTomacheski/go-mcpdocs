# Onde Estamos - Progresso do Projeto API GitHub

---
**Resumo Atual (Maio/2025)**

- A API está funcional, robusta e pronta para uso básico.
- Endpoints principais implementados: busca de repositórios, extração e processamento concorrente de documentação, pesquisa e health check.
- Conexão segura com GitHub, tratamento de erros, configuração flexível e shutdown gracioso.
- Testes básicos presentes, arquitetura preparada para escalar.

**Próximos passos sugeridos:**
1. Implementar autenticação para proteger endpoints.
2. Adicionar cache para otimizar chamadas ao GitHub.
3. Implementar rate limiting para evitar abuso.
4. Expandir processadores de documentação (ex: Markdown para HTML).
5. Melhorar cobertura de testes automatizados.
6. Documentação interativa (Swagger/OpenAPI).

Veja detalhes completos abaixo:
---

## Fluxo do Sistema em 9 Etapas

1. **Carregamento da Configuração**
   - O sistema lê variáveis de ambiente (.env) para obter token do GitHub, porta do servidor, tamanho do pool de workers e timeout (`config/config.go`).
2. **Inicialização de Dependências**
   - Inicializa logger, configurações e cliente GitHub (`main.go`).
3. **Criação do Handler da API**
   - Instancia o handler central da API, que recebe o cliente GitHub e o tamanho do pool de workers (`api/handlers.go`).
4. **Configuração do Router e Middlewares**
   - Define rotas REST, middlewares de CORS e timeout, e organiza rotas em grupos (`api/router.go`).
5. **Inicialização do Servidor HTTP**
   - Configura e inicia o servidor HTTP, pronto para aceitar conexões (`main.go`).
6. **Recepção e Roteamento das Requisições**
   - O router recebe requisições e direciona para os handlers apropriados (`api/router.go`).
7. **Processamento das Requisições**
   - Os handlers processam as requisições, extraem parâmetros, validam entrada e coordenam a chamada ao cliente GitHub e ao processador de documentação (`api/handlers.go`, `api/url_handler.go`, `api/enhanced_url_handler.go`).
8. **Extração e Processamento de Documentação**
   - O processador (`internal/processor/extractor.go`) extrai snippets de código dos arquivos de documentação, formata e organiza a resposta.
9. **Resposta ao Cliente e Shutdown Gracioso**
   - O sistema retorna os dados processados ao cliente, e implementa shutdown gracioso ao receber sinais do sistema (`main.go`).


Este documento detalha o progresso atual do projeto de API em Go para buscar e processar documentação de repositórios do GitHub com tratamento de erros e concorrência adequados.

## 1. Instalação e Configuração Inicial

### Instalação do Go
- Instalamos o Go no sistema Fedora 42 usando `sudo dnf install golang`
- A instalação incluiu os pacotes golang, golang-bin, golang-src e dependências relacionadas

### Inicialização do Projeto
- Criamos um módulo Go com `go mod init github.com/dtomacheski/extract-data-go`
- Instalamos as dependências necessárias com:
  ```
  go get github.com/google/go-github/v53/github golang.org/x/oauth2 github.com/gin-gonic/gin github.com/joho/godotenv
  ```

## 2. Estrutura do Projeto

Criamos a seguinte estrutura de diretórios:
```
extract-data-go/
├── api/               # Controladores da API e roteamento
├── config/            # Configurações da aplicação
├── internal/          # Código interno da aplicação
│   ├── github/        # Cliente e funções para API do GitHub
│   └── models/        # Modelos de dados
├── .env               # Variáveis de ambiente (tokens, configurações)
├── .env.example       # Exemplo de variáveis de ambiente
├── go.mod             # Definição do módulo e dependências
├── go.sum             # Checksums das dependências
├── main.go            # Ponto de entrada da aplicação
└── README.md          # Documentação do projeto
```

## 3. Componentes Implementados

### Modelos de Dados (`internal/models/repo.go`)
- Definimos estruturas para representar:
  - `Repository`: Informações sobre repositórios GitHub
  - `Documentation`: Conteúdo de documentação de repositórios
  - `ErrorResponse`: Formato padronizado para respostas de erro
  - `SuccessResponse`: Formato padronizado para respostas de sucesso

### Cliente GitHub (`internal/github/client.go`)
- Implementamos um cliente para a API do GitHub com:
  - Autenticação via token OAuth
  - Métodos para buscar informações de repositórios
  - Métodos para buscar documentação de repositórios
  - Processamento concorrente usando goroutines e semáforos
  - Tratamento adequado de erros da API
  - Funções para buscar arquivos de documentação (README, etc.)
  - Função de pesquisa de repositórios

### Configuração (`config/config.go`)
- Sistema de configuração baseado em variáveis de ambiente:
  - Leitura de tokens do GitHub
  - Configuração da porta do servidor
  - Tamanho do pool de workers para concorrência
  - Timeout para requisições à API do GitHub
- Suporte para arquivo `.env` usando godotenv

### API Handlers (`api/handlers.go`)
- Implementamos handlers HTTP para:
  - Buscar informações de repositórios (`GET /api/v1/repos/:owner/:repo`)
  - Buscar documentação de repositórios (`GET /api/v1/repos/:owner/:repo/docs`)
  - Pesquisar repositórios (`GET /api/v1/search/repos?q=:query`)
  - Verificação de saúde (`GET /health`)
- Tratamento adequado de erros e códigos de status HTTP

### Roteamento da API (`api/router.go`)
- Configuração de rotas usando o framework Gin
- Middleware CORS para permitir solicitações cross-origin
- Middleware de timeout para limitar tempo de resposta
- Organização de rotas em grupos (v1)

### Aplicação Principal (`main.go`)
- Ponto de entrada da aplicação
- Carregamento de configurações
- Inicialização do cliente GitHub
- Configuração do servidor HTTP
- Implementação de desligamento gracioso (graceful shutdown)
- Tratamento de sinais do sistema (SIGINT, SIGTERM)

### Testes (`internal/github/client_test.go`)
- Testes para o cliente GitHub:
  - Teste para busca de repositórios
  - Teste para busca de documentação

## 4. Configuração e Variáveis de Ambiente

Configuramos os seguintes parâmetros via `.env`:
- `GITHUB_TOKEN`: Token de acesso pessoal do GitHub
- `PORT`: Porta do servidor (padrão: 8080)
- `WORKER_POOL_SIZE`: Tamanho do pool de workers para concorrência (padrão: 5)
- `REQUEST_TIMEOUT`: Timeout para requisições à API do GitHub (padrão: 30s)

## 5. Endpoints da API

A API REST expõe os seguintes endpoints:

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/health` | Verificação de saúde da API |
| GET | `/api/v1/repos/:owner/:repo` | Busca informações de um repositório |
| GET | `/api/v1/repos/:owner/:repo/docs` | Busca documentação de um repositório |
| GET | `/api/v1/search/repos?q=:query` | Pesquisa repositórios |

## 6. Construção e Execução

- Construímos a aplicação com `go build -o github-doc-api`
- Executamos o servidor com `./github-doc-api`
- O servidor está configurado para escutar na porta 8080 (ou configurada)

## 7. Funcionalidades Implementadas

- ✅ Busca de repositórios do GitHub
- ✅ Extração de arquivos de documentação (README, docs, etc.)
- ✅ Processamento concorrente dos arquivos de documentação
- ✅ Tratamento adequado de erros com respostas detalhadas
- ✅ Paginação para resultados de pesquisa
- ✅ Desligamento gracioso do servidor
- ✅ Configuração flexível via variáveis de ambiente

## 8. Estado Atual

- O servidor está funcional e respondendo a requisições
- As chamadas à API do GitHub estão funcionando corretamente
- Os endpoints estão retornando dados no formato JSON esperado
- A concorrência está implementada para melhorar a performance
- Testes básicos foram implementados

## 9. Próximos Passos Prioritários

1. **Autenticação:**
   - Proteger endpoints sensíveis com autenticação JWT ou OAuth.
2. **Cache:**
   - Implementar cache em memória (ex: Redis ou Go cache) para reduzir requisições repetidas ao GitHub.
3. **Rate Limiting:**
   - Adicionar middleware para limitar requisições por IP/usuário.
4. **Processadores de Documentação:**
   - Converter Markdown para HTML e permitir outros formatos.
5. **Cobertura de Testes:**
   - Expandir testes unitários e integração, cobrindo casos de erro e concorrência.
6. **Documentação Interativa:**
   - Gerar documentação Swagger/OpenAPI acessível pela API.
7. **(Opcional) WebSockets:**
   - Permitir notificações em tempo real para clientes que acompanham processamento de grandes repositórios.

## 10. Comandos Úteis

- Iniciar o servidor: `./github-doc-api`
- Testar endpoints:
  ```
  curl http://localhost:8080/health
  curl http://localhost:8080/api/v1/repos/:owner/:repo
  curl http://localhost:8080/api/v1/repos/:owner/:repo/docs
  curl "http://localhost:8080/api/v1/search/repos?q=golang+api"
  ```
- Executar testes: `go test ./...`
- Formatar código: `go fmt ./...`
