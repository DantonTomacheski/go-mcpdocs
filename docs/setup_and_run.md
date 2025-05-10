# Configuração e Execução do go-mcpdocs

Este guia fornece instruções passo a passo para configurar o ambiente de desenvolvimento, instalar dependências, configurar e executar a aplicação `go-mcpdocs` localmente.

## Pré-requisitos

Antes de começar, certifique-se de que você tem os seguintes softwares instalados:

*   **Go:** Versão 1.18 ou superior (verifique a versão no arquivo `go.mod`).
*   **Git:** Para clonar o repositório.
*   **(Opcional) Docker:** Se você planeja usar Redis via Docker para caching.
*   **(Opcional) Make:** Se você for utilizar os comandos do `Makefile`.

## 1. Clonar o Repositório

Clone o repositório do projeto para a sua máquina local:

```bash
git clone <URL_DO_REPOSITORIO_GO-MCPDOCS>
cd go-mcpdocs
```

Substitua `<URL_DO_REPOSITORIO_GO-MCPDOCS>` pela URL correta do repositório.

## 2. Instalar Dependências

O projeto usa Go Modules para gerenciamento de dependências. Para instalar as dependências necessárias, execute:

```bash
go mod tidy
```

Ou, se houver um `Makefile` com um alvo para isso (comum em projetos Go):

```bash
make deps
```

## 3. Configurar Variáveis de Ambiente

A aplicação `go-mcpdocs` é configurada principalmente através de variáveis de ambiente. Existe um arquivo de exemplo ` .env.example` no projeto. Copie este arquivo para `.env` e edite-o com suas configurações.

```bash
cp .env.example .env
```

Abra o arquivo `.env` em um editor de texto e configure as seguintes variáveis (conforme a estrutura `config.Config`):

*   `GITHUB_TOKEN` (Obrigatório): Seu token de acesso pessoal do GitHub com as permissões necessárias para ler repositórios. Este é crucial para que a aplicação possa interagir com a API do GitHub.
    *   _Como gerar um token:_ Vá para GitHub -> Settings -> Developer settings -> Personal access tokens -> Generate new token. Certifique-se de que o token tenha escopo `repo` (para acesso a repositórios públicos e privados) ou `public_repo` (apenas para repositórios públicos).
*   `PORT` (Opcional, Padrão: `8080`): A porta na qual o servidor HTTP da API irá escutar.
*   `WORKER_POOL_SIZE` (Opcional, Padrão: `10`): O número de workers concorrentes para processar arquivos de documentação.
*   `REQUEST_TIMEOUT_SECONDS` (Opcional, Padrão: `30`): O tempo máximo em segundos para uma requisição ser processada.
*   `ENABLE_CACHE` (Opcional, Padrão: `false`): Defina como `true` para habilitar o caching.
*   `CACHE_PROVIDER` (Opcional, Padrão: `memory`): Define o provedor de cache. Pode ser `memory` para cache em memória ou `redis` para usar Redis.
*   `REDIS_URI` (Obrigatório se `CACHE_PROVIDER=redis`): A URI de conexão para o seu servidor Redis (e.g., `redis://localhost:6379` ou uma URI do Upstash).
*   `CACHE_TTL_SECONDS` (Opcional, Padrão: `3600`): O tempo de vida (Time To Live) para os itens no cache, em segundos.

**Exemplo de arquivo `.env`:**

```
GITHUB_TOKEN=seu_github_token_aqui
PORT=8080
WORKER_POOL_SIZE=10
REQUEST_TIMEOUT_SECONDS=60
ENABLE_CACHE=true
CACHE_PROVIDER=redis
REDIS_URI=redis://localhost:6379
CACHE_TTL_SECONDS=7200
```

## 4. (Opcional) Configurar Redis para Cache

Se você habilitou o cache com Redis (`ENABLE_CACHE=true` e `CACHE_PROVIDER=redis`), você precisará de uma instância Redis rodando e acessível pela `REDIS_URI` que você configurou.

Você pode usar uma instância Redis local (instalada diretamente ou via Docker) ou um serviço de Redis na nuvem como o Upstash.

**Para rodar Redis com Docker:**

```bash
docker run -d -p 6379:6379 --name mcpdocs-redis redis
```

## 5. Executar a Aplicação

A aplicação é iniciada através do arquivo `main.go`.

**Usando `go run`:**

```bash
go run main.go
```

**Usando o `Makefile` (se disponível um comando `run` ou `start`):
**Verifique o `Makefile` por alvos como `run`, `start` ou `dev`.
Por exemplo:

```bash
make run
```

Ou, para desenvolvimento com live reload (se configurado, por exemplo, com Air, como sugere o arquivo `.air.toml` no projeto):

```bash
make dev
# ou diretamente se Air estiver instalado
# air
```

Após a inicialização, se não houver erros, você verá logs indicando que o servidor está escutando na porta configurada (e.g., `Server listening on :8080`).

## 6. Verificar se a Aplicação está Rodando

Abra seu navegador ou use uma ferramenta como `curl` para acessar o endpoint de health check:

```bash
curl http://localhost:<SUA_PORTA>/health
```

Substitua `<SUA_PORTA>` pela porta que você configurou (padrão `8080`).

Você deve receber uma resposta JSON indicando o status da aplicação, por exemplo:

```json
{
  "status": "UP",
  "timestamp": "2023-10-27T12:00:00Z"
}
```

Parabéns! Você configurou e executou com sucesso a aplicação `go-mcpdocs` localmente.