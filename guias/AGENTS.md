# AI Stack Guidance — Universal Best Practices

> Arquivo de referência para I.A. executar código correto à primeira vez.  
> Leia antes de gerar qualquer código, script ou configuração.

---

## Go 1.26

### Imports

- Nunca invente pacotes. Confirme o caminho exato no `go.mod` antes de importar.
- Imports devem ser agrupados: stdlib → externos → internos. Sempre nessa ordem.
- `"errors"` é stdlib. `"github.com/pkg/errors"` é externo. Não confunda.
- Aliases de import só quando há colisão real de nome. Nunca por estética.
- Remova imports não usados. O compilador rejeita. Nunca deixe `_` para "resolver depois".

### Erros

- Retorne `error` como último valor de retorno. Sempre.
- Use `fmt.Errorf("contexto: %w", err)` para wrapping. Nunca `fmt.Sprintf` para erros.
- Checagem de erro logo após a chamada. Nunca agrupe checagens no final da função.
- `errors.Is` e `errors.As` para comparação. Nunca `err.Error() == "string"`.
- Nunca ignore erros silenciosamente com `_`. Se for descartado, comente o porquê.

### Structs e Interfaces

- Interfaces são definidas no lado do consumidor, não do implementador.
- Interface com um método: nome = verbo + `er`. Ex: `Reader`, `Writer`, `Closer`.
- Nunca embuta interfaces em structs concretas sem necessidade clara.
- Campos exportados em structs são públicos. Pense antes de exportar.
- Tags de struct (`json:"name"`) sempre em backtick. Nunca aspas duplas.

### Goroutines e Concorrência

- Toda goroutine precisa de um dono claro que gerencie seu ciclo de vida.
- Nunca lance goroutine sem garantia de encerramento (`context`, `sync.WaitGroup`, canal de stop).
- `sync.Mutex` para estado compartilhado. `channel` para comunicação. Não misture.
- `defer mu.Unlock()` imediatamente após `mu.Lock()`. Nunca separe os dois.
- `context.Context` é sempre o primeiro parâmetro. Nunca guardado em struct.

### Context

- Assinatura: `func Foo(ctx context.Context, ...)`. Sempre primeiro.
- Nunca passe `nil` como context. Use `context.Background()` ou `context.TODO()`.
- `context.TODO()` só em código temporário. Substitua antes de commitar.
- Valores em context só para dados transversais (request ID, trace). Nunca para lógica de negócio.
- Cancele contexts derivados: `defer cancel()` logo após `WithCancel` / `WithTimeout`.

### Convenções de Pacote

- Nome do pacote = nome do diretório. Singular, minúsculo, sem underscores.
- Não use `util`, `common`, `helpers`, `misc`. Nomeie pelo domínio.
- Funções construtoras: `New` + nome do tipo. Ex: `NewUserService(...)`.
- Variáveis de erro exportadas: `var ErrNaoEncontrado = errors.New("...")`.
- Constantes de enum: tipo named + `iota`. Nunca `int` solto.

### Inicialização e `init()`

- Evite `init()`. Prefira inicialização explícita no `main` ou em funções construtoras.
- Se usar `init()`, nunca execute I/O ou lógica com efeitos colaterais visíveis.

### Build e CGO

- `CGO_ENABLED=0` significa zero dependência de libc. Não use pacotes que exijam CGO nesse modo.
- Binary estático: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/app .`
- Nunca use `go run` em produção. Compile e execute o binário.

---

## PostgreSQL 18

### Convenções de Schema

- Nomes de tabela em `snake_case`, singular. Ex: `usuario`, `pedido_item`.
- Toda tabela tem `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`.
- Timestamps: `created_at TIMESTAMPTZ DEFAULT now()`, `updated_at TIMESTAMPTZ`.
- Nunca use `SERIAL` ou `BIGSERIAL` em projetos novos. Use `UUID` ou `BIGINT GENERATED ALWAYS AS IDENTITY`.

### Queries e Segurança

- Nunca concatene input do usuário em SQL. Sempre parâmetros posicionais: `$1`, `$2`.
- `RETURNING` em `INSERT` e `UPDATE` evita roundtrip extra para buscar o registro.
- `EXPLAIN ANALYZE` antes de considerar uma query "pronta" em tabelas grandes.
- Índices: crie para todas as colunas usadas em `WHERE`, `JOIN ON`, `ORDER BY` frequentes.
- `NOT NULL` é a regra. `NULL` é a exceção. Seja explícito.

### Row Level Security (RLS)

- RLS ativado: `ALTER TABLE foo ENABLE ROW LEVEL SECURITY`.
- `FORCE ROW LEVEL SECURITY` para que o owner da tabela também seja restringido.
- Policies sempre com `USING` (leitura) e `WITH CHECK` (escrita) explícitos.
- Nunca escreva policy que retorna `TRUE` sem condição. Isso nega o propósito do RLS.
- Teste policies com `SET LOCAL role = 'nome_role'` dentro de transação.

### Transações

- Lógica de negócio crítica dentro de transação explícita (`BEGIN` / `COMMIT`).
- Nunca faça I/O externo (HTTP, fila) dentro de uma transação aberta.
- `ROLLBACK` em caso de erro. Nunca deixe transação pendente.

---

## pgx/v5

### Conexão e Pool

- Use `pgxpool.New` para pool de conexões. Nunca `pgx.Connect` direto em servidor.
- `pgxpool.Config` permite controlar `MaxConns`, `MinConns`, `MaxConnLifetime`.
- Sempre feche o pool no shutdown: `pool.Close()`.
- `ctx` passado para cada operação. Nunca `context.Background()` hardcoded em handlers HTTP.

### Queries

- `pool.QueryRow(ctx, sql, args...)` para uma linha. `pool.Query` para múltiplas.
- `pgx.Rows`: sempre `defer rows.Close()` após `Query`. Sem isso, conexão fica presa.
- Cheque `rows.Err()` após o loop de `rows.Next()`. Erros de scan aparecem ali.
- `pgxscan` (scany) ou scan manual. Nunca `database/sql` misturado com pgx.
- Tipos pgx nativos: `pgtype.UUID`, `pgtype.Timestamptz`. Use-os. Não converta para `string` desnecessariamente.

### Transações com pgx

```go
tx, err := pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback(ctx) // Seguro: no-op se já commitado

// ... operações ...

return tx.Commit(ctx)
```

- Nunca chame `tx.Commit` e `tx.Rollback` na mesma execução.
- `defer tx.Rollback(ctx)` é idiomático e seguro. Não remova.

---

## chi/v5

### Roteamento

- `r.Use(middleware)` aplica para todas as rotas do grupo. Ordem importa.
- `r.Route("/path", func(r chi.Router) {...})` para subrotas aninhadas.
- `chi.URLParam(r, "id")` para path params. Nunca `r.URL.Query().Get` para params de path.
- Middleware de autenticação sempre antes das rotas protegidas, no grupo correto.

### Handlers

- Assinatura: `func(w http.ResponseWriter, r *http.Request)`. Nunca desvie.
- Leia body uma vez. Use `io.LimitReader` para prevenir DoS por payload gigante.
- Sempre sete `Content-Type` antes de escrever o body: `w.Header().Set("Content-Type", "application/json")`.
- `w.WriteHeader(status)` antes de `w.Write(body)`. Inverter não funciona como esperado.
- Retorne após escrever erro: `http.Error(w, msg, code); return`. Sem o `return`, o handler continua.

### Context de Request

- `r.Context()` para obter o context da requisição. Cancela quando o cliente desconecta.
- Valores no context via key tipada (tipo privado). Nunca string como chave de context.

```go
type ctxKey string
const keyUserID ctxKey = "user_id"
ctx = context.WithValue(ctx, keyUserID, userID)
```

---

## golang-jwt/v5

### Geração de Token

- Nunca use `jwt.SigningMethodNone`. Sempre `RS256`, `ES256` ou `HS256` com chave forte.
- Claims customizados: struct que embute `jwt.RegisteredClaims`.
- `ExpiresAt` sempre definido. Token sem expiração é falha de segurança.

### Validação

- Use `jwt.ParseWithClaims` com `keyFunc` que retorna a chave correta.
- Nunca valide manualmente o campo `exp`. Deixe a lib fazer.
- Cheque o erro de parse antes de acessar os claims. Claims podem estar parcialmente preenchidos mesmo com erro.
- `token.Valid` deve ser `true` após parse sem erro. Confirme explicitamente.

---

## asynq (hibiken/asynq v0.24)

### Tarefas

- Payload de task: JSON serializado. Sempre valide o unmarshal no handler.
- `asynq.NewTask(typeName, payload)` cria a task. `typeName` é string constante, nunca valor dinâmico.
- Defina constantes para os tipos de task: `const TypeEmailBoas-vindas = "email:boas_vindas"`.

### Fila e Retry

- Configure `MaxRetry` por task. Default não é zero — confirme o valor.
- `asynq.Queue("critical")` para prioridade. Defina as filas no servidor também.
- Handler deve retornar `nil` para sucesso e `error` para retry. `asynq.SkipRetry` para falha permanente.
- Nunca bloqueie o handler por tempo indeterminado. Use context com timeout.

### Servidor

```go
srv := asynq.NewServer(redisOpt, asynq.Config{
    Concurrency: 10,
    Queues: map[string]int{
        "critical": 6,
        "default":  3,
        "low":      1,
    },
})
```

- `Concurrency` deve respeitar o limite de conexões do banco.
- Graceful shutdown: `srv.Shutdown()` no signal handler.

---

## templ (v0.3+)

### Geração de Código

- Arquivos `.templ` geram `_templ.go`. Nunca edite `_templ.go` diretamente.
- Execute `templ generate` após qualquer alteração em `.templ`. O build falha sem isso.
- No CI/CD: `templ generate` antes de `go build`. Sempre.

### Componentes

- Componentes são funções Go. Tipos dos parâmetros são verificados em compilação.
- Nunca concatene HTML como string dentro de templ. Use a sintaxe nativa `{ }`.
- Para atributos condicionais, use `if` dentro do template. Nunca `fmt.Sprintf` com HTML.
- Escaping é automático para conteúdo em `{ }`. Para HTML literal, use `templ.Raw()` — com cuidado e sanitização prévia.

---

## yaegi (traefik/yaegi v0.16)

### Execução Dinâmica

- yaegi interpreta Go, mas não suporta toda a stdlib. Confirme o pacote antes de usar em scripts.
- Registre símbolos explicitamente com `interp.Use(stdlib.Symbols)` ou seu mapa customizado.
- Nunca passe input de usuário diretamente como código a ser interpretado sem sanitização e sandbox.
- Erros de compilação do yaegi são retornados em `interp.Eval()`. Sempre cheque.

---

## wazero (tetratelabs/wazero v1.5)

### Módulos WASM

- Compile com `wazero.NewRuntime(ctx)`. O runtime é pesado — reutilize, não recrie por request.
- Módulos são imutáveis após compilação. Compile uma vez, instancie múltiplas vezes.
- Funções exportadas do WASM: acesse via `mod.ExportedFunction("nome")`. Confirme o nome exato.
- Memória WASM é isolada. Para passar dados complexos, use ponteiro + tamanho na memória do módulo.
- Sempre `defer mod.Close(ctx)` após instanciar o módulo.

---

## Docker / Docker Compose

### Imagens

- Baseie em `alpine`. Nunca `latest` em produção — sempre tag explícita.
- Multi-stage build: estágio `builder` com Go toolchain, estágio final só com o binário.
- `COPY --from=builder /app/bin/app /app` — copie apenas o necessário.
- `USER nonroot` no estágio final. Nunca rode como root dentro do container.

### Variáveis de Ambiente e Segredos

- Nunca hardcode credenciais no `docker-compose.yml`. Use `.env` (gitignore) ou secrets.
- Passe credenciais via `environment:` referenciando variável do host: `DB_PASS: ${DB_PASS}`.

### Healthchecks

- Todo serviço que outro depende deve ter `healthcheck` definido.
- `depends_on` com `condition: service_healthy`. Nunca só `depends_on: serviço`.
- Intervalo inicial (`start_period`) deve cobrir o tempo real de boot do serviço.

### Networking

- Serviços na mesma compose network se comunicam pelo nome do serviço, não por `localhost`.
- Exponha portas para o host só quando necessário. Comunicação interna não precisa de `ports`.

---

## Nginx (1.25-alpine)

- `proxy_pass` deve apontar para o nome do serviço Docker, não `localhost`.
- `proxy_set_header Host $host` e `X-Real-IP $remote_addr` sempre configurados.
- Timeouts explícitos: `proxy_read_timeout`, `proxy_connect_timeout`.
- Não sirva arquivos estáticos de dentro de um container de API. Use volume ou CDN.

---

## Princípios Gerais — Qualquer Stack

- **Leia o `go.mod` / `package.json` / `requirements.txt` antes de sugerir dependência.** Nunca invente versões.
- **Versão importa.** API de v4 e v5 do mesmo pacote podem ser incompatíveis. Confirme.
- **Nenhum `TODO` sem dono.** Se não vai resolver agora, não escreva.
- **Sem código morto.** Funções não usadas aumentam superfície de confusão.
- **Logs com contexto.** `log.Printf("falha ao criar usuário id=%s: %v", id, err)`. Nunca só `log.Println(err)`.
- **Nenhuma credencial em código ou log.** Nunca.
- **Teste o caminho de erro, não só o happy path.** A maioria dos bugs vive no `else`.
- **Configuração via ambiente.** Nunca valor hardcoded que muda entre dev e prod.
- **Shutdown gracioso.** Todo servidor/worker captura `SIGTERM`/`SIGINT` e finaliza conexões abertas.