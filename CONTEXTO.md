# Contexto do Projeto — Conversion Tracker

> Documento de handoff. Use isto pra dar contexto a um novo chat sobre onde paramos.
> Última atualização: 2026-06-13

---

## 1. Visão geral

Este é um **novo sistema de rastreamento e envio de conversões** para a **Meta Conversions API** (Facebook/Instagram), construído do zero pra substituir um sistema legado em Laravel.

- **Backend**: Go 1.22 (HTTP API + worker assíncrono)
- **Frontend**: Nuxt 4 (Vue 3 + TypeScript) — _planejado, ainda não implementado_
- **Banco**: Postgres 16
- **Fila/Cache**: Redis 7 (via biblioteca `asynq`)
- **Foco atual**: somente Meta Ads. Google Ads fica pra fase 2.

### Por que Go + Nuxt

A decisão veio de uma análise do sistema atual em Laravel. Conclusões:

- Webhooks + fan-out pra Meta CAPI são caso ideal pra Go (latência baixa, goroutines, binário único, deploy simples).
- Nuxt 4 dá produtividade alta no painel admin (file-based routing, auto-import, SSR, TypeScript).
- API REST/JSON desacoplada deixa front e back evoluírem independentes.
- Tipagem estática do Go ajuda com a variedade de payloads dos diferentes gateways de pagamento.

### Sistema legado (referência)

O sistema antigo é um Laravel 11 que ficará em `referencia/` no novo workspace. Ele tem:

- ~25 webhooks de gateways (Hotmart, Kiwify, Eduzz, Braip, Cakto, Ticto, Perfect Pay, etc.)
- Eventos de tracking client-side (PageView, ViewContent, Lead, Purchase, Scroll, Vídeo)
- Integração Meta CAPI via `esign/laravel-conversions-api`
- Integração Google Ads via `googleads/google-ads-php`
- GeoIP2 (MaxMind)
- Painel admin com Blade
- Deploy via Docker Swarm/Portainer + GitHub Actions
- Inúmeros scripts `.sh` de manutenção

**Não vamos copiar o código**, mas usar como referência funcional do domínio (formatos de payload, regras de negócio, normalização, etc.).

---

## 2. Estrutura de diretórios atual

```
conversion-tracker/
├── README.md
├── CONTEXTO.md             ← este arquivo
├── docker-compose.yml      # postgres, redis, api, worker
├── .env.example
├── .gitignore
└── backend/
    ├── go.mod / go.sum
    ├── Dockerfile          # multi-stage: target api + target worker
    ├── migrations/
    │   └── 0001_init.sql
    ├── cmd/
    │   ├── api/main.go     # binário HTTP
    │   └── worker/main.go  # binário consumidor da fila
    └── internal/
        ├── auth/jwt.go     # JWT HS256 + bcrypt + ctx claims
        ├── config/config.go
        ├── db/db.go        # pool pgx/v5 + migrate idempotente
        ├── domain/
        │   ├── pixel.go
        │   ├── tracked_user.go
        │   └── event.go
        ├── geoip/lookup.go
        ├── handlers/
        │   ├── events.go       # /events/init, /events/send
        │   ├── webhooks.go     # /webhooks/kiwify, /webhooks/hotmart
        │   ├── admin_auth.go   # /admin/login, /admin/me, bootstrap
        │   ├── admin_pixels.go # CRUD /admin/pixels
        │   └── helpers.go
        ├── httpserver/
        │   ├── server.go       # router chi + CORS + rotas
        │   └── middleware.go   # JWTMiddleware
        ├── meta/client.go      # cliente Meta CAPI Graph API
        ├── queue/
        │   ├── tasks.go        # asynq enqueuer
        │   └── worker.go       # asynq server + handler
        ├── repo/
        │   ├── pixel_repo.go
        │   └── user_repo.go
        └── util/normalize.go   # normalização + SHA256
```

---

## 3. Decisões de arquitetura

### 3.1 Multi-pixel por nome
Um `pixel.name` (slug usado pelo client e gravado no `content_id` dos webhooks) pode ter **múltiplos pixel_ids** do Meta com seus respectivos `access_tokens` e `test_codes`. Isso permite enviar o mesmo evento pra vários pixels (ex: pixel principal + redundância). É um array no Postgres (`TEXT[]`).

### 3.2 Pipeline assíncrono
1. Cliente (pixel.js ou webhook) chama a API.
2. API valida, faz GeoIP, faz upsert do `tracked_user`.
3. API enfileira um `meta:send_event` em Redis via `asynq`.
4. Worker pega da fila, busca o pixel, monta o payload com user_data hasheado (SHA256), envia pra `graph.facebook.com/{version}/{pixel_id}/events`.
5. Cada tentativa é gravada em `sent_events` (auditoria + futura dedup).

API responde rápido (`202 Accepted`) e o worker faz o trabalho pesado com retry automático (até 5 tentativas).

### 3.3 Identificação de usuário
- Pixel client gera um `user_id` (UUID) e persiste em cookie/localStorage.
- Esse `user_id` vai como `external_id` no Meta e como chave única em `tracked_users.external_id`.
- Webhooks de gateway extraem o `user_id` do `utm_source` no formato `userId<id>|...` (mesmo padrão do sistema legado), com fallback por telefone.

### 3.4 Hash SHA256 antes de enviar
Meta CAPI exige PIIs hasheados (email, phone, fn, ln, ct, st, zp, country, external_id). Fazemos isso no `meta/client.go` via `util.SHA256Hex` depois de normalizar (`util/normalize.go`):
- Email: trim + lowercase
- Phone: só dígitos, prefixa `55` quando for número BR sem DDI
- Nomes/cidade/estado: lowercase + remoção de acentos + remove não-alfa

### 3.5 Migrações idempotentes
Tudo no `migrations/0001_init.sql` usa `CREATE TABLE IF NOT EXISTS` e `CREATE INDEX IF NOT EXISTS`. Não há lib de migração; `db.Migrate` lê o diretório em ordem alfabética e executa os `.sql`. Pra novas migrações, basta criar `0002_xxx.sql`, `0003_xxx.sql`, etc., com cláusulas idempotentes (`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, etc.).

### 3.6 Schema do banco

```sql
admin_users    -- login do painel admin (email + bcrypt hash)
pixels         -- name, label, pixel_ids[], access_tokens[], test_codes[], enabled
tracked_users  -- external_id (UNIQUE), pixel_name, fbp, fbc, geoip,
               -- pii (first_name, last_name, email, phone),
               -- url, utms first/last
sent_events    -- auditoria: event_id, event_name, pixel_name, status,
               -- response_code, response_body, payload (jsonb)
```

Todos os campos texto opcionais são `NOT NULL DEFAULT ''` pra simplificar scans em Go (sem `sql.NullString`).

### 3.7 Auth do painel
- `POST /admin/login` retorna JWT HS256.
- Frontend guarda em cookie `auth_token`.
- Middleware `JWTMiddleware` valida e injeta claims no `context.Context`.
- Bootstrap: na primeira execução, se não houver admin, cria com `ADMIN_EMAIL`/`ADMIN_PASSWORD` do env.

### 3.8 Timeouts e middleware
- HTTP timeout global: 15s.
- Read header timeout: 5s.
- CORS: `*` (ajustar pra dev/prod).
- Recoverer + RequestID + RealIP do chi.

---

## 4. Endpoints

### Públicos (chamados pelo `pixel.js` no site do cliente)

```
POST /api/v1/events/init
  body: { user_id, pixel_name, fbp, fbc }
  resp: { client_ip_address, client_user_agent, country, state, city,
          postal_code, external_id, fbp, fbc }

POST /api/v1/events/send
  body: {
    event_name, event_id, user_id, pixel_name, content_id,
    value, currency, url, fbc, fbp,
    first_name, last_name, email, phone,
    utm_source, utm_medium, utm_campaign, utm_content, utm_term
  }
  resp: { queued: true, event_id }   // 202 Accepted
```

Eventos aceitos: `PageView`, `ViewContent`, `Lead`, `Contact`, `InitiateCheckout`, `AddPaymentInfo`, `AddToCart`, `AddToWishlist`, `CompleteRegistration`, `Purchase`.

### Webhooks de gateway

```
POST /api/v1/webhooks/kiwify    (formato Kiwify)
POST /api/v1/webhooks/hotmart   (formato Hotmart)
```

Ambos: parseiam payload, extraem `external_id` do `utm_source`, fazem fallback por telefone, atualizam `tracked_users` com PIIs do gateway, enfileiram `Purchase` pra Meta.

### Admin (Bearer JWT)

```
POST   /api/v1/admin/login          { email, password } → { access_token, user }
GET    /api/v1/admin/me             (precisa Bearer)
GET    /api/v1/admin/pixels         lista (tokens mascarados)
POST   /api/v1/admin/pixels         cria pixel
PUT    /api/v1/admin/pixels/:id     atualiza pixel
DELETE /api/v1/admin/pixels/:id     remove pixel
```

### Health

```
GET /api/v1/health → { "status": "ok" }
```

---

## 5. Estado atual e validação

- ✅ `go mod tidy` rodou limpo
- ✅ `go build ./...` compila sem erros
- ✅ `go vet ./...` sem warnings
- ✅ `get_diagnostics` zerado nos arquivos principais
- ⏳ Ainda não foi rodado `docker compose up` (precisa do `.env` preenchido e idealmente do `GeoLite2-City.mmdb`)
- ⏳ Frontend Nuxt 4 não foi implementado (decidido adiar)

### Path de import Go

O `go.mod` está como:

```
module github.com/renankeny/conversion-tracker/backend
```

**Se você mover o projeto pra outro repositório ou path GitHub, precisa atualizar:**
1. A linha `module` em `go.mod`
2. Todos os imports `github.com/renankeny/conversion-tracker/backend/...` nos arquivos `.go`

Um `find . -type f -name '*.go' -exec sed -i '' 's|github.com/renankeny/conversion-tracker/backend|NOVO/PATH|g' {} +` resolve, mais editar o `go.mod`.

---

## 6. O que está faltando / próximos passos

### Curto prazo
- [ ] **`pixel.js`** (script client-side embedável) — o que o cliente final cola no site.
- [ ] **Dedup por `event_id`** antes do enqueue (consultar `sent_events`).
- [ ] **Mais webhooks**: Eduzz, Cakto, Braip, Ticto, PerfectPay, etc. Cada um é um handler em `webhooks.go` (ou um arquivo separado).
- [ ] **Testes** unitários (normalização, hash, parsers de webhook).
- [ ] **Frontend Nuxt 4** — login + dashboard + CRUD de pixels.

### Médio prazo
- [ ] Métricas Prometheus (`/metrics`).
- [ ] Rate limiting por IP nos endpoints públicos.
- [ ] CORS configurável por origem.
- [ ] Logs estruturados com nível configurável (já tem `slog`, falta amarrar `LOG_LEVEL`).
- [ ] Validar payloads com `go-playground/validator` (já tá no `go.mod` mas não usado ainda).
- [ ] CI (GitHub Actions): lint + test + build.
- [ ] Deploy pra produção (Traefik/Caddy na frente, TLS).

### Fase 2
- [ ] Google Ads — REST API (não tem SDK oficial em Go).
- [ ] URL black/white mapping (feature do legado pra mapear domínios "black" pra "white" antes de mandar pra Meta).
- [ ] Multi-tenant (hoje é single-tenant).

---

## 7. Como rodar local

```bash
cd conversion-tracker
cp .env.example .env
# edite .env, principalmente:
#   JWT_SECRET=<algo grande aleatório>
#   ADMIN_PASSWORD=<senha do admin inicial>

# (opcional) baixe GeoLite2-City.mmdb (MaxMind) e coloque em backend/data/

docker compose up --build
```

API: http://localhost:8080/api/v1/health

Smoke test:

```bash
# login
curl -s -X POST http://localhost:8080/api/v1/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"changeme"}'

# criar pixel
TOKEN="<cole_o_access_token>"
curl -s -X POST http://localhost:8080/api/v1/admin/pixels \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "produto-x",
    "label": "Produto X - principal",
    "pixel_ids": ["123456789"],
    "access_tokens": ["EAAxxxxx..."],
    "test_codes": ["TEST12345"]
  }'

# disparar evento (simula pixel.js)
curl -s -X POST http://localhost:8080/api/v1/events/send \
  -H 'Content-Type: application/json' \
  -d '{
    "event_name": "PageView",
    "user_id": "user_abc",
    "pixel_name": "produto-x",
    "url": "https://exemplo.com/landing"
  }'
```

---

## 8. Tech stack e libs (Go)

```go
github.com/go-chi/chi/v5             // roteador HTTP
github.com/go-chi/cors               // middleware CORS
github.com/go-playground/validator/v10 // validação (ainda não usado)
github.com/golang-jwt/jwt/v5         // JWT
github.com/google/uuid               // event IDs
github.com/hibiken/asynq             // fila com Redis
github.com/jackc/pgx/v5              // driver Postgres + pool
github.com/oschwald/geoip2-golang    // MaxMind GeoLite2
golang.org/x/crypto                  // bcrypt
```

Nada de ORM. SQL cru com `pgx`. Logs com `slog` da stdlib.

---

## 9. Padrões de código adotados

- Pacote `internal/` pra esconder símbolos privados ao módulo.
- Cada subpacote tem responsabilidade clara (`auth`, `config`, `db`, `domain`, `geoip`, `handlers`, `httpserver`, `meta`, `queue`, `repo`, `util`).
- Sem ciclos de import (claims context vive em `auth`).
- Erros propagados com `fmt.Errorf("contexto: %w", err)`.
- Logs estruturados com `slog.Info/Warn/Error` e chaves consistentes.
- Handlers retornam JSON via `writeJSON`/`writeError` (em `handlers/helpers.go`).

---

## 10. Como retomar num novo chat

Cole isto como primeira mensagem (ou aponte pro arquivo):

> Estou continuando o projeto `conversion-tracker`. É uma reescrita de um sistema legado de tracking pra Meta Conversions API, agora em Go (backend) + Nuxt 4 (frontend, pendente). O contexto completo, decisões e estado atual estão em `CONTEXTO.md` na raiz. O sistema legado em Laravel 11 fica em `referencia/`.
> Já está implementado: backend Go com endpoints públicos (`/events/*`), webhooks (Kiwify, Hotmart) e admin (login + CRUD de pixels), fila assíncrona com asynq, Postgres com migrations idempotentes, integração Meta CAPI com hashing SHA256.
> Quero seguir com [SEU PRÓXIMO PASSO AQUI: ex. "implementar o pixel.js client-side", "adicionar webhook do Eduzz", "construir o painel Nuxt 4", etc.].

E confirme que ele leu o `CONTEXTO.md` e o `README.md` antes de tocar em qualquer coisa.
