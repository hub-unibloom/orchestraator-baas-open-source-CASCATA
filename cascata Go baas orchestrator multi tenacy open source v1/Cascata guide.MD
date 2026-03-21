# Cascata — Guia de Desenvolvimento

## Contexto do Projeto

O Cascata é um BaaS multi-tenant open source de auto-hospedagem. Sua arquitetura segue o modelo de condomínio: o orquestrador central (Cascata) gerencia múltiplos bancos de dados isolados fisicamente — cada tenant tem seu próprio apartamento, completamente independente dos demais, seguro desde a entrada até o armazenamento e permite as pesaos orchetrrem com liberddae suas portas, portões, janelas, etc. O objetivo é criar um BaaS que seja ao mesmo tempo seguro, escalável e fácil de usar. Onde o orquestrador é o proprio banco de dados, ou seja, o cascata é um banco de dados que gerencia outros bancos de dados, e cada banco de dados é um tenant, e cada tenant tem seu próprio banco de dados operano sob o mes

Este repositório é o **Cascata v1.0.0.0**, reescrito em Go para atingir a performance que a arquitetura exige. O desenvolvimento acontece aqui:
```
/home/cocorico/Documentos/proejetos/orchestraator baas open source/cascata Go baas orchestrator multi tenacy open source v1
```

A versão anterior (v0.1, Node.js) está disponível como **referência de leitura** em:
```
/home/cocorico/Documentos/proejetos/orchestraator baas open source/cascata
```
Essa pasta não deve ser modificada. Ela existe para consulta de ideias, fluxos e features consolidadas que devem ser transportadas — não copiadas — para a nova base Go.

---

## Fase Atual

Organização e ambientação do novo ambiente Go. O objetivo desta fase é estabelecer a estrutura de diretórios, convenções, módulos e fundação arquitetural que sustentará todo o desenvolvimento seguinte. O front-end e as ideias inovadoras do v0.1 serão aproveitados.

---

## Padrão de Engenharia

Todo o trabalho aqui é produzido no nível que um engenheiro sênior de sistemas distribuídos defenderia em code review bancário.

**Cada arquivo entregue deve:**
- Ser sinérgico com o restante do sistema — uma alteração considera todos os pontos de contato
- Conter comentários precisos e valores justificados
- Ter estrutura coesa e padronização consistente em todo o codebase

**A abordagem correta para qualquer implementação:**
1. Ler o código relevante antes de agir
2. Identificar todos os pontos de impacto da mudança
3. Implementar a solução real e completa — não aproximações
4. Entregar cada arquivo em estado melhor do que estava

Se falta um bloco, crie-o completo. Se há um conflito, resolva o conflito na raiz. O caminho é sempre a implementação correta, nunca o atalho.

---

## Regras Operacionais

- **Nunca solicitar acesso ao terminal.** O projeto roda em outra máquina; aqui é apenas desenvolvimento de código.
- **Nenhuma feature do v0.1 pode ser perdida** sem decisão explícita de descarte.
- **Isolamento de tenants é inegociável** — nenhuma implementação pode criar superfície de vazamento entre tenants.
- **Sem mocks em código de produção.** Interfaces de teste existem em camadas de teste, não no core.

---

## Estrutura de Diretórios v1.0.0.0 (Go)

- `/cmd/cascata`: Ponto de entrada do executável principal do orquestrador.
- `/internal`: Código privado do orquestrador (não pode ser importado por outros projetos).
    - `/internal/orchestrator`: Core de gerenciamento de tenants e orquestração de containers.
    - `/internal/api`: Handlers REST e roteamento.
    - `/internal/auth`: Governança AI (MCP 3-modes), JWT e Segurança Zero Trust.
    - `/internal/database`: Interface com PostgreSQL e PostgreSQL RLS (Row Level Security).
    - `/internal/privacy`: Engine de masking, blur e criptografia em repouso.
    - `/internal/config`: Gerenciamento de variáveis de ambiente e arquivos de configuração.
- `/deployments`: Manifestos Docker para orquestração local e de produção.

## Pilares Atuais de Desenvolvimento

1. **MCP Triple-Mode Auth**: Suporte a tokens de Admin (`service_key`), Público (`anon_key`) e Usuário Real (`JWT User Token`), garantindo que o agente AI opere sob o mesmo contexto de segurança de um usuário humano.
2. **PostgreSQL RLS Nativo**: Injeção de claims do JWT diretamente no PostgreSQL para filtragem automática em nível de linha.
3. **Isolamento Físico**: Garantia que dados de diferentes tenants nunca residam no mesmo banco de dados físico ou instância, mantendo o princípio do "Apartamento".