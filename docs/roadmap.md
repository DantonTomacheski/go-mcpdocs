# Roadmap do Projeto go-mcpdocs (Plano para 2025)

Este documento descreve os próximos passos e as direções de desenvolvimento planejadas para o projeto `go-mcpdocs`, com base no plano estratégico para 2025. O objetivo é evoluir o `go-mcpdocs` para se tornar uma fonte ainda mais robusta, segura e atualizada de documentação e exemplos de código, tanto para Modelos de Linguagem Grande (LLMs) quanto para desenvolvedores.

## Áreas de Foco e Melhorias Planejadas

### 1. Autenticação e Segurança
*   **Implementação de JWT/OAuth:** Adicionar mecanismos de autenticação robustos para proteger a API.
*   **Rate Limiting:** Implementar limites de taxa para prevenir abusos e garantir a disponibilidade do serviço.
*   **Prioridade:** Alta, fundamental para a segurança e confiabilidade da API.

### 2. Cache e Performance
*   **Integração Avançada de Cache:** Expandir as opções de cache, possivelmente com Redis como padrão para produção, e otimizar estratégias de invalidação.
*   **Otimização do Pool de Workers:** Ajustar e refinar o gerenciamento do pool de workers para máxima eficiência na extração de dados.
*   **Prioridade:** Alta, crítico para a performance e escalabilidade.

### 3. Expansão dos Processadores de Documentação
*   **Suporte a Múltiplos Formatos:** Além de Markdown, adicionar suporte para outros formatos de documentação comuns (e.g., AsciiDoc, reStructuredText).
*   **Conversão de Markdown para HTML:** Oferecer a opção de obter documentação formatada em HTML.
*   **Enriquecimento de Exemplos de Código:** Melhorar a extração e apresentação de exemplos de código encontrados na documentação.

### 4. Evolução da API e Integração
*   **Novos Endpoints REST:** Desenvolver novos endpoints para funcionalidades adicionais (e.g., estatísticas de processamento, busca mais granular).
*   **Documentação Swagger/OpenAPI:** Gerar e manter documentação da API no formato Swagger/OpenAPI para facilitar a integração por parte dos consumidores.

### 5. Resiliência e Observabilidade
*   **Logging Estruturado:** Implementar logging estruturado para facilitar a análise e depuração.
*   **Métricas de Performance e Saúde:** Coletar e expor métricas detalhadas sobre o desempenho da aplicação e a saúde dos seus componentes.
*   **Testes de Integração e Carga:** Desenvolver um conjunto abrangente de testes de integração e carga para garantir a estabilidade e a capacidade de resposta sob demanda.

### 6. Melhoria da Experiência do Usuário (UX)
*   **Exemplos de Uso da API:** Fornecer exemplos claros e práticos de como consumir a API `go-mcpdocs`.
*   **Respostas Formatadas e Flexíveis:** Oferecer opções para formatar as respostas da API de acordo com as necessidades do cliente.
*   **Health Check Detalhado:** Expandir o endpoint de health check para fornecer mais informações sobre o estado dos serviços dependentes.
*   **Endpoint de Feedback:** Criar um mecanismo para que os usuários possam fornecer feedback sobre a API e a documentação.

### 7. Preparação para Escalabilidade
*   **Deploy Multi-Região:** Investigar e preparar a aplicação para deployments em múltiplas regiões geográficas, visando alta disponibilidade e baixa latência.
*   **Filas para Processamento Assíncrono:** Considerar o uso de filas de mensagens (e.g., RabbitMQ, Kafka) para processamento assíncrono de tarefas mais longas, como a primeira indexação de um repositório muito grande.

## Revisão Contínua
Este plano será revisado periodicamente (a cada ciclo de planejamento) para garantir o alinhamento contínuo com as necessidades dos usuários, os avanços tecnológicos (especialmente em LLMs) e os objetivos estratégicos do projeto.

## Medição de Sucesso
O sucesso das melhorias e do projeto como um todo será medido pela:
*   Adoção da API por desenvolvedores e outros sistemas.
*   Redução de respostas desatualizadas ou "hallucinadas" em LLMs que consomem os dados fornecidos pelo `go-mcpdocs`.
*   Feedback positivo da comunidade de usuários.
