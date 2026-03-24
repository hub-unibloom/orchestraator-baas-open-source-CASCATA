# CASCATA — CODING STANDARDS
**Status:** DEFINITIVO / IMUTÁVEL  
**Aplicável a:** Backend Go, Frontend React/TS, toda camada do sistema

> Este documento define como o código é escrito no Cascata. Toda implementação — nova ou modificada — segue estas regras sem exceção. O padrão exigido é o que um engenheiro sênior de sistemas distribuídos bancários defenderia em code review.

---

## 1. PRINCÍPIOS TRANSVERSAIS

**Universal Synergy First (Holistic Excellence).**  
Segurança não é um anexo; é um componente de uma sinergia maior. Tratamos **Segurança, Performance, Escalabilidade e Harmonia Arquitetural** como pilares inseparáveis e de igual prioridade. Focar apenas em um (ex: segurança) em detrimento de outro (ex: arquitetura correta) colapsa o sistema a longo prazo. Um código Cascata deve ser seguro porque é bem estruturado, e performático porque é inteligentee t rabalha corretamente com a arquritetura e versẽos das stacks como go, react, dragonfly, postgres, vault, opentelemetry, sdk's e etc.

**Zero Mock.**  
Não existem funções placeholder. A implementação é real ou retorna erro explícito de "não implementado". Nunca silenciar, nunca fingir.

**Auto-explicação pelo nome, comentário pelo porquê.**  
O nome da função diz o que faz. O comentário — quando necessário — diz por que foi feita dessa forma e não de outra. Comentários que apenas repetem o código são ruído.

**Idioma no código: sempre inglês.**  
Nomes de variáveis, funções, tipos, pacotes, comentários, mensagens de erro, logs estruturados — tudo em inglês. A interface do dashboard é inglês por padrão. Traduções são arquivos externos carregados em runtime, nunca hardcoded no bundle.

**Fail Fast na borda.**
Dado inválido morre no Handler, antes de qualquer lógica. Em sistema multi-tenant, um erro de tipo que vaza para o banco de um tenant é catastrófico. A validação é imediata e agressiva via `SendError` (OTel Integrated).

**Terminologia Obrigatória (Phase 10-17 Sinergy):**
- **KV/Streaming:** Sempre **Dragonfly** (nunca Redis).
- **Usuários:** Sempre **Resident** (nunca User - ex: `ModeResident`, `IdentityResident`).
- **Audit:** Sempre **Cascata Audit Ledger**.

**Nenhum componente confia no anterior.**  
Go não confia que o Nginx validou. O Postgres não confia que o Go aplicou RLS. Cada camada valida por conta própria. (Ver Filosofia de Segurança no Master Plan.)

**Nomenclatura de dados sensíveis.**  
Nunca `password`, `key`, `token` puros em variáveis. Sempre o estado do dado no nome:
- `hashedPassword`, `rawInputSecret`, `encryptedVaultKey`, `signedJWT`, `derivedSessionKey`

---

## 2. GO — BACKEND

### 2.1 Estrutura de Pacotes
- Nomes: minúsculos, palavra única, funcionais — `auth`, `vault`, `pool`, `phantom`, `storage`
- **Proibido:** `utils`, `helpers`, `common`, `misc`
- Funções e tipos exportados: `PascalCase`
- Erros exportados: prefixo `Err` — `ErrProjectNotFound`, `ErrVaultSealed`, `ErrTenantIsolation`

**Go DAG Alignment (Acyclic Structure):**
- A arquitetura do Cascata **DEVE** ser um Grafo Acíclico Dirigido (DAG).
- **Ciclos de Importação são terminantemente proibidos.** Se o Pacote A precisa do Pacote B e vice-versa, a abstração está errada.
- **Solução Obrigatória:** Use interfaces definidas no pacote `internal/domain` para quebrar ciclos. Componentes dependem de contratos neutros, não de implementações concretas uns dos outros.
- **Zero Reinvenção:** Não tente criar um "pneu novo" para uma roda que já existe. Utilize o melhor que o Go 1.26 oferece (Interfaces, Generics, Context, Slog) em vez de forçar padrões de outras linguagens. Respeite a performance nativa e a filosofia da linguagem.

### 2.2 Arquitetura em Camadas (obrigatória)
```
Handler     → decodifica request, valida na borda, codifica response via `SendJSON/SendError`. Zero lógica de negócio.
Service     → lógica de negócio, orquestração, validações complexas. Zero acesso direto a dados.
Repository  → acesso a dados puro (Postgres, Dragonfly, Vault). Zero lógica de negócio.
```
Cruzar camadas é proibido. Handler não acessa Repository diretamente.

### 2.3 Tipagem
- Toda resposta de API tem `struct` definida com tags `json:"field_name"`
- **Proibido:** `map[string]any` ou `interface{}` em retornos JSON
- **Proibido:** structs aninhadas anônimas em respostas de API

### 2.4 Erros e Contexto
```go
// CORRETO
if err != nil {
    return fmt.Errorf("vault.DecryptKey: %w", err)
}

// CORRETO — Saída de API com Rastro OTel
SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "Identity verification failed")

// PROIBIDO — sem contexto
if err != nil {
    return err
}

// PROIBIDO — erro silenciado
_ = riskyOperation()
```
- `panic` é rigorosamente proibido fora de `main()`. Em runtime, use o middleware de resiliência `HandlePanic` no `server.go` para capturar falhas e transformá-las em Spans de erro no OTel.
- `context.Context` é o primeiro parâmetro de toda função de I/O (DB, Dragonfly, Vault, FS, HTTP) e DEVE carregar o rastro OTel para propagação transversal.

### 2.5 Logging
- **Proibido:** `fmt.Println`, `log.Print`, qualquer print não estruturado
- Use `slog` (stdlib) com campos consistentes: `project_id`, `tenant_slug`, `user_id`, `request_id`
- Todo request propaga `RequestID` no contexto para rastreabilidade cross-service
- **Proibido em logs:** `Authorization`, `Cookie`, tokens, segredos — dados sensíveis removidos antes de persistir

### 2.6 Segurança no Código
- Funções de criptografia e Vault isoladas em pacotes próprios e auditáveis
- Chaves nunca passadas como `string` pura entre funções — use tipos opacos ou structs dedicadas
- RLS Fail-Closed: se a injeção de contexto falhar, `ROLLBACK` forçado — nunca executa query sem contexto de segurança estabelecido.
- **Database Sinergy (Phase 22):** O padrão `WithRLS` deve implementar loop de **Retry para erros transientes** (conn reset) e injeção atômica em um único RTT SQL.

### 2.7 Concorrência
- Todo canal tem `select` com `case ctx.Done()` — sem goroutine que bloqueia indefinidamente
- Worker Pools para automações assíncronas — sem goroutine solta por request
- Goroutines têm propósito explícito e identificável

### 2.8 Docker & Arquivos
- Todo novo serviço Go tem `Dockerfile.txt` otimizado para produção (distroless ou alpine)
- Arquivos `.env`, `Dockerfile` permanecem com extensão `.txt` no ambiente de desenvolvimento
- Segredos nunca em variáveis de ambiente no docker-compose — sempre via Vault

---

## 3. REACT / TYPESCRIPT — FRONTEND

### 3.1 Componentes
- **Apenas functional components** — sem `Class Components`
- Lógica complexa extraída para Custom Hooks: `useProjectVault`, `useTenantMetrics`, `useSSEChannel`
- Componente renderiza, hook orquestra — nunca misturar

### 3.2 Tipagem
- **Proibido:** `any` em qualquer lugar
- **Proibido:** `as unknown as Type` — se o tipo não bate, a interface de origem está errada
- Use `unknown` com `type guard` explícito quando o tipo é genuinamente incerto
- Generics são preferidos a type casting

### 3.3 Estado e API
- Estado global apenas para o que realmente atravessa toda a aplicação: Auth, configurações globais de tenant
- Estado local para todo o resto
- **Proibido:** `fetch` ou `axios` direto em componente — toda chamada de API passa por camada de Service ou Hook dedicado
- Erros de API têm tipo definido — nunca `catch(e: any)`

### 3.5 Temas & Internacionalização
- Temas implementados via CSS variables — trocar tema é trocar variáveis, não reescrever componentes
- Nenhuma cor, sombra ou espaçamento hardcoded fora das variáveis de tema
- Strings de interface nunca hardcoded em componentes — sempre via chave de i18n
- Arquivo de tradução carregado em runtime — bundle não cresce com novos idiomas
- Inglês é o idioma fallback garantido — se chave não existe no idioma configurado, exibe inglês
- Segredos nunca no bundle ou em `localStorage` — tokens em memória ou `httpOnly cookie`
- Inputs com dados sensíveis: `autocomplete="off"`, sem logging de valores
- Headers de segurança validados na camada de Service antes de retornar dados ao componente

---

## 4. BANCO DE DADOS (PostgreSQL)

- Novas tabelas nascem com `REVOKE ALL` — acesso concedido explicitamente via RLS Policies
- `FORCE ROW LEVEL SECURITY` em toda tabela exposta pela API
- Índices compostos para queries de alta cardinalidade: `(project_slug, created_at DESC)`
- Migrations versionadas com checksum — nunca editar migration já aplicada
- `statement_timeout` ajustado por tipo de operação: agressivo para queries de usuário, relaxado para manutenção
- Nomes de colunas sensíveis descrevem o estado: `hashed_password`, `encrypted_payload`

---

## 7. PERFORMANCE, ESCALA HORIZONTAL & EXCELÊNCIA ALGORÍTMICA

> O Cascata v0 provou em produção: 5k usuários simultâneos em 2GB de RAM com Node.js. Isso não foi acidente — foi resultado de arquitetura correta, algoritmos precisos e caching bem posicionado. O v1 em Go existe para remover os últimos limites que o runtime anterior impunha. Cada decisão de implementação deve honrar esse legado e ir além.

**O objetivo não é apenas funcionar — é ser digno de referência técnica.**

### 7.1 Princípios de Performance como Lei

**Medir antes de otimizar, mas projetar para escala desde o início.**
Nenhuma estrutura de dados é escolhida sem considerar seu comportamento sob carga. Um `map` onde deveria haver um `slice` ordenado, um lock onde deveria haver lock-free — são decisões que parecem irrelevantes em desenvolvimento e destroem performance em produção.

**Matemática é bem-vinda. Abstrações custam.**
Quando um algoritmo pode ser expresso matematicamente de forma mais eficiente, ele deve ser. Média móvel exponencial em vez de recalcular média completa. Bloom filter antes de query ao banco. Consistent hashing para distribuição de carga. O projeto deve abusar de matemática onde ela traz ganho real — não por elegância, mas por resultado.

**Goroutines são baratas — use-as corretamente.**
~2KB de stack por goroutine vs ~1MB por thread. Isso não é motivo para criar goroutines sem critério — é motivo para modelar concorrência com precisão. Worker pools de tamanho fixo, channels com buffer dimensionado por throughput esperado, select com ctx.Done() sem exceção.

**Zero alocação em hot paths.**
Funções que executam milhares de vezes por segundo não alocam. Usam `sync.Pool` para reutilizar buffers. Usam `[]byte` em vez de `string` onde a conversão seria desnecessária. Usam structs pré-alocadas em vez de maps. O GC do Go é excelente — mas o melhor GC é o que não tem nada para coletar.

**I/O nunca bloqueia o worker principal.**
Toda operação de I/O — banco, Dragonfly, Vault, storage, HTTP externo — ocorre em goroutine própria ou com timeout explícito via context. O orquestrador nunca para para esperar. Cada operação de I/O deve abrir um sub-span no OTel.

### 7.2 Escala Horizontal como Objetivo de Design

**Toda feature deve funcionar com N instâncias do orquestrador rodando simultaneamente.**
Estado compartilhado vai no Dragonfly, não em memória local. Cache local (L1) é para leitura — writes sempre passam pelo Dragonfly para que todas as instâncias vejam o estado correto. Pub/Sub de invalidação garante consistência sem polling.

**Nenhuma decisão de arquitetura pode criar singleton de estado.**
Se uma feature exige que exista exatamente uma instância processando algo — ela está errada. Worker pools distribuídos, filas no Dragonfly, locks distribuídos com TTL: são as ferramentas corretas. Um lock que não expira automaticamente é uma bomba-relógio.

**Sharding por tenant é a unidade natural de escala.**
O modelo de isolamento físico não é só segurança — é escala. Mover um tenant para outro servidor Postgres é uma operação de minutos, não de dias. Quando o sistema crescer, a escala horizontal é horizontal de verdade: mais servidores Postgres, mais instâncias do orquestrador Go, mais nós Dragonfly — sem reescrever nada.

### 7.3 Benchmarks como Critério de Aceitação

Toda Fase que toca performance crítica tem benchmark obrigatório antes de ser considerada concluída:

| Componente | Meta mínima |
|---|---|
| Overhead do orquestrador por request | < 10ms |
| RLS injection + query em um RTT | < 5ms adicional |
| Resolução de variável `{{env.VAR}}` | < 0.5ms |
| Latência SSE após mudança no banco | < 50ms |
| Provisionamento de novo tenant | < 2s |
| Geração de `.caf` (1GB de dados) | sem arquivo temporário em disco |
| Cold start de edge function V8 | < 100ms |
| Throughput de automações assíncronas | > 10k/min por instância |

Se um benchmark regredir em relação à versão anterior da mesma fase, a implementação não foi concluída.

### 7.4 O Padrão de Algoritmo Correto

Antes de implementar qualquer algoritmo com impacto em performance, responder:

1. Qual é a complexidade de tempo no caso médio e no pior caso?
2. Qual é a complexidade de espaço?
3. Existe uma estrutura de dados mais adequada para este acesso pattern?
4. Este algoritmo escala linearmente com o número de tenants, ou há um ponto de inflexão?
5. Existe uma solução matemática que evita iteração completa?

Exemplos de decisões corretas que devem ser padrão:
- **Índices compostos** para queries de alta cardinalidade — não deixar o Postgres fazer seq scan onde um index scan resolve em O(log n)
- **Consistent hashing** para distribuição de tenants entre workers — rebalanceamento mínimo ao adicionar nós
- **Exponential backoff com jitter** em retries — evita thundering herd quando múltiplos agentes falham simultaneamente
- **Sliding window** para rate limiting — mais justo e mais preciso que fixed window
- **Bloom filter** antes de queries de existência ao banco — elimina round trips para casos negativos
- **OTel Trace Context Propagation** — Todo evento assíncrono (XADD no Dragonfly) leva o TraceID/SpanID para o worker de automação.

---

### 5.1 Atomicidade Total (Tudo ou Nada)
Toda operação que envolve múltiplos passos é atômica: ou todos os passos concluem com sucesso, ou toda a cadeia é revertida para o estado original — sem exceção, sem estado intermediário persistido.

```go
// CORRETO — transação explícita, rollback garantido
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return fmt.Errorf("service.CreateTenant: begin tx: %w", err)
}
defer tx.Rollback() // no-op se Commit() foi chamado

if err := stepA(ctx, tx); err != nil {
    // Rollback automático pelo defer — nenhum efeito parcial persiste
    return fmt.Errorf("service.CreateTenant: stepA: %w", err)
}
if err := stepB(ctx, tx); err != nil {
    return fmt.Errorf("service.CreateTenant: stepB: %w", err)
}

return tx.Commit()
```

- Toda falha gera log estruturado evidenciando: qual passo falhou, o contexto completo e o estado revertido
- O log de falha é obrigatório mesmo quando o rollback foi bem-sucedido — a operação tentada deve ser rastreável
- Operações distribuídas (Postgres + Dragonfly + Vault) usam compensação explícita: se o passo 3 falha, os passos 1 e 2 são desfeitos ativamente

### 5.2 Código Defensivo (Resiliência sob Pressão)
O código limpo e direto é o objetivo. A robustez existe nas bordas e transições — onde dados chegam, onde I/O ocorre, onde estado muda. Esses são os pontos onde bugs aparecem em produção sob carga real.

**O princípio:** não confie que o caminho feliz sempre ocorre. Escreva o caminho feliz limpo, e proteja as transições.

```go
// FRÁGIL — assume que o dado sempre existe e é válido
project := cache.Get(slug)
return project.Config.RateLimit

// DEFENSIVO — protege cada transição sem poluir a lógica
project, ok := cache.Get(slug)
if !ok {
    // fallback controlado, não crash
    return defaultRateLimit, nil
}
if project.Config == nil {
    return 0, fmt.Errorf("pool.GetRateLimit: config nil para projeto %s", slug)
}
return project.Config.RateLimit, nil
```

**Onde aplicar obrigatoriamente:**
- Toda deserialização de dados externos (API, banco, fila, webhook)
- Todo acesso a cache (pode estar vazio, expirado ou corrompido)
- Todo I/O com timeout (pode não responder, pode responder parcialmente)
- Toda operação concorrente (pode ter condição de corrida)
- Toda integração externa (Vault, Dragonfly, Qdrant — podem estar temporariamente indisponíveis)

**O que não fazer:**
- Código defensivo no meio da lógica de negócio — isso é complexidade desnecessária
- `recover()` genérico que engole qualquer panic silenciosamente
- Retry infinito sem backoff e sem limite de tentativas

**Padrão de retry para integrações críticas:**
```go
// Backoff exponencial com limite — não tenta para sempre
const maxAttempts = 3
for attempt := range maxAttempts {
    err = vaultClient.GetSecret(ctx, key)
    if err == nil {
        break
    }
    if attempt == maxAttempts-1 {
        return fmt.Errorf("vault.GetSecret: %d tentativas esgotadas: %w", maxAttempts, err)
    }
    time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond)
}
```

---

## 6. O QUE NUNCA É ACEITÁVEL
Segue a lista ddo que é proibido:
- Estado intermediário persistido após falha — operação incompleta reverte tudo
- Ação sem log — toda operação tem evidência estruturada do que aconteceu e quando possivel o por quê.
- Deleção como solução para um bug — se deletar resolve, o problema está em outro lugar,(Solução: então não elete simplesmente e sim invetigue como atuar apra queaa feature exista com performace, qualidade, códigos realmente robustos que previnem falhas).
- Feature parcial em produção — implementação completa ou não entra
- Regressão silenciosa — arquivo modificado sai melhor do que entrou, sempre
- Segredo em texto plano em qualquer lugar — disco, log, variável de ambiente, comentário
- Confiança implícita entre componentes baseada em posição na rede
- `recover()` genérico silenciando panics sem log e sem contexto
- Retry sem limite de tentativas e sem backoff exponencial
- Código que funciona mas que o autor não consegue explicar linha por linha

---

**Diretiva permanente: cada linha de código é defensável em code review bancário.**