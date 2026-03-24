# Cascata — Guia de Desenvolvimento

## Contexto do Projeto

O Cascata é um BaaS multi-tenant open source de auto-hospedagem. Sua arquitetura segue o modelo de condomínio: o orquestrador central (Cascata) gerencia múltiplos bancos de dados isolados fisicamente — cada tenant tem seu próprio apartamento, completamente independente dos demais, seguro desde a entrada até o armazenamento e permite as pessoas orquestrarem com liberdade suas portas, portões, janelas, etc. O objetivo é criar um BaaS que seja ao mesmo tempo seguro, escalável e fácil de usar. Onde o orquestrador é o próprio banco de dados, ou seja, o Cascata é um banco de dados que gerencia outros bancos de dados, e cada banco de dados é um tenant, e cada tenant tem seu próprio banco de dados operando sob o mesmo.

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
Todo o trabalho aqui é produzido no nível que um engenheiro sênior de sistemas distribuídos defenderia em code review bancário. Implementamos uma **Visão Holística**, onde Segurança, Performance, Escalabilidade e Harmonia Arquitetural têm o mesmo peso. Focar apenas em um colapsa o sistema; o Cascata é uma nação que protege sua fronteira, mas que também garante saúde e infraestrutura interna impecáveis.

**Cada arquivo entregue deve:**
- Ser sinérgico com o restante do sistema — uma alteração considera todos os pontos de contato.
- Honrar a arquitetura nativa da linguagem (Go 1.26+) — não force padrões de outras linguagens.
- Contribuir para a manutenção de um Grafo Acíclico Dirigido (DAG).
- Conter comentários precisos e valores justificados.
- Ter estrutura coesa e padronização consistente em todo o codebase.

**A abordagem correta para qualquer implementação:**
1. Ler o código relevante antes de agir.
2. Identificar todos os pontos de impacto da mudança.
3. Respeitar o fluxo de dependências — **Proibido criar ciclos de importação.** Utilize o pacote `internal/domain` para contratos neutros.
4. Implementar a solução real e completa — não aproximações.
5. Entregar cada arquivo em estado melhor do que estava.

Se falta um bloco, crie-o completo. Se há um conflito, resolva o conflito na raiz. O caminho é sempre a implementação correta, nunca o atalho ou a "burla" tecnológica.

---

## Léxico de Sinergia (Dicionário Inegociável)

Para garantir a perfeição real e a consistência absoluta em todo o código, estas são as únicas nomenclaturas permitidas:

1.  **Identity (Identidade):** Termo genérico para qualquer entidade que realiza uma ação.
2.  **Member (Membro):** Operador do Dashboard/Orquestrador (Worner, Manager, Dev). Tabela: `system.members`.
3.  **Resident (Residente):** Usuário final de um inquilino/tenant. Tabela: `auth.users` (do tenant).
4.  **Agent (Agente):** Entidade programática (AI ou Automação Phantom).
5.  **IdentityContext (Contexto de Identidade):** Estrutura que carrega `IdentityID` e `IdentityType`.
6.  **Audit Ledger (Livro de Auditoria):** O registro imutável e chained de todas as operações críticas.

**Proibido o uso de variantes como:** Actor, Scribe, User (genérico), Sindicante, Morador (em pt-br no código), ou qualquer termo poético fora deste léxico.

---

## Regras Operacionais

- **Nunca solicitar acesso ao terminal.** O projeto roda em outra máquina; aqui é apenas desenvolvimento de código.
- **Nenhuma feature do v0.1 pode ser perdida** sem decisão explícita de descarte.
- **Isolamento de tenants é inegociável** — nenhuma implementação pode criar superfície de vazamento entre tenants.
- **Sem mocks em código de produção.** Interfaces de teste existem em camadas de teste, não no core.
- **Terminologia Cascata (Sinergia):** Use **Dragonfly** para KV/Streaming e **Resident** para identidade de usuário final.

---

## Estrutura de Diretórios v1.0.0.0 (Go)

- `/cmd/cascata`: Ponto de entrada do executável principal do orquestrador.
- `/internal`: Código privado do orquestrador.
    - `/internal/api`: Handlers REST e roteamento resiliente via anti-fragility middleware.
    - `/internal/auth`: Governança AI (MCP 3-modes: Service, Resident, Public), JWT e Segurança Zero Trust.
    - `/internal/database`: Interface com PostgreSQL RLS e Pooling Gerenciado por Tenant.
    - `/internal/telemetry`: Mesh de rastreamento (OpenTelemetry) para observabilidade transversal.
    - `/internal/automation`: Fila de eventos e engine de workflows via Dragonfly Streams.
    - `/internal/privacy`: Engine de masking, blur e criptografia em repouso.
    - `/internal/config`: Gerenciamento de variáveis de ambiente e segredos via Vault.
- `/deployments`: Manifestos Docker para orquestração local e de produção.

---

## Pilares Atuais de Desenvolvimento

1. **MCP Triple-Mode Auth**: Suporte a tokens de Admin (`service_key`), Público (`anon_key`) e Usuário Real (`JWT Resident Token`), garantindo que o agente AI opere sob o mesmo contexto de segurança de um residente humano.
2. **PostgreSQL RLS Nativo**: Injeção de claims do JWT diretamente no PostgreSQL para filtragem automática em nível de linha via injeção atômica no `WithRLS`.
3. **Alto Desempenho via Dragonfly**: Uso de DragonflyDB como motor de alta performance para Caching, Rate-limiting e Streaming de eventos assíncronos (Phase 10-17).
4. **Observabilidade Total (OTel)**: Rastreabilidade distribuída ponta-a-ponta, desde a requisição na API até a persistência no banco e execução de workflows (Phase 17-22).
5. **Isolamento Físico**: Garantia que dados de diferentes tenants nunca residam no mesmo banco de dados físico ou instância, mantendo o princípio do "Apartamento".