# TrackFind

Plataforma de rastreamento de conversões para Meta Ads (e futuramente Google Ads).
Espelha eventos web em tempo real para a **Meta Conversions API** com cookies first-party,
GeoIP enrichment e enfileiramento assíncrono via Redis.

## Stack

- **Backend**: Go 1.22 + chi + pgx + asynq + JWT
- **Frontend**: Nuxt 4 + Tailwind CSS + TypeScript
- **Banco**: Postgres 16 (TIMESTAMPTZ em America/Sao_Paulo)
- **Fila/Cache**: Redis 7
- **GeoIP**: MaxMind GeoLite2-City
- **Container**: Docker + Docker Compose

## Estrutura

```
trackfind/
├── backend/                  # API Go + Worker Go
│   ├── cmd/api/              # binário HTTP
│   ├── cmd/worker/           # binário consumidor da fila
│   ├── cmd/admin-set/        # CLI: upsert do admin a partir do .env
│   ├── migrations/           # SQL idempotente
│   ├── internal/             # auth, db, domain, handlers, queue, repo, util...
│   └── Dockerfile
├── frontend/                 # Painel Nuxt 4
│   ├── app/
│   │   ├── components/       # Base*, AppSidebar, pixels/*, etc.
│   │   ├── composables/      # useAuth, usePixels, useTheme, useToast
│   │   ├── layouts/          # default, app
│   │   ├── pages/            # login, index, integracoes/pixels, ...
│   │   └── plugins/
│   ├── shared/types/         # DTOs compartilhados
│   └── Dockerfile
├── scripts/                  # crontab, update-geoip.sh
├── docker-compose.yml
├── BANCO DE DADOS.sql        # Espelho consolidado do schema
└── .env.example
```

## Recursos principais

- **Tracker.js dinâmico** servido pelo backend (estilo GTM), gerado a partir de `tracker_config` JSONB editável
- **Pixel multi-Meta**: cada pixel TrackFind pode espelhar eventos pra múltiplos `pixel_ids` da Meta
- **First-party tracking**: cliente aponta `tracking.dominio.com` via CNAME, evita adblockers
- **Webhooks de gateway**: Kiwify e Hotmart prontos (mais virão)
- **Admin protegido**: JWT + rate limit no login + sanitização de tokens nos logs
- **Dedup por event_id**: previne envio duplicado pra Meta

## Endpoints

### Públicos (consumidos pelo `tracker.js`)

- `GET  /api/v1/loader.js?pixel=<slug>`
- `GET  /api/v1/tracker.js?pixel=<slug>`
- `POST /api/v1/events/init`
- `POST /api/v1/events/send`

### Webhooks

- `POST /api/v1/webhooks/kiwify`
- `POST /api/v1/webhooks/hotmart`

### Admin (Bearer JWT)

- `POST   /api/v1/admin/login`
- `GET    /api/v1/admin/me`
- `GET    /api/v1/admin/pixels` `POST /api/v1/admin/pixels`
- `GET    /api/v1/admin/pixels/{id}`
- `PUT    /api/v1/admin/pixels/{id}`
- `DELETE /api/v1/admin/pixels/{id}` (exige `{ "confirm": true }`)
- `POST   /api/v1/admin/pixels/{id}/verify-dns`
- `GET    /api/v1/admin/sent-events` `GET /api/v1/admin/tracked-users`
- `GET    /api/v1/admin/stats`
- `GET    /api/v1/admin/url-mappings` `POST /api/v1/admin/url-mappings`

## Rodar localmente (dev)

### Pré-requisitos

- Docker + Docker Compose
- Go 1.22 (opcional, se quiser rodar sem Docker)
- Node 20 (opcional, frontend)

### Setup

```bash
cp .env.example .env
# Edite .env com:
#   JWT_SECRET (gerar com `openssl rand -hex 64`)
#   ADMIN_EMAIL / ADMIN_PASSWORD
#   MAXMIND_LICENSE_KEY (opcional, pra rodar update-geoip.sh)
```

### Subir tudo via Docker

```bash
docker compose up --build
```

API em `http://localhost:8080/api/v1/health`.

### Subir backend localmente sem Docker

Em 2 terminais separados (use o Postgres/Redis remotos do `.env`):

```bash
# API
cd backend
set -a && source ../.env && set +a && go run ./cmd/api

# Worker
cd backend
set -a && source ../.env && set +a && go run ./cmd/worker
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Painel em `http://localhost:3000`.

## Deploy no EasyPanel

### Visão geral

O EasyPanel já tem **Postgres** e **Redis** configurados em outros services.
Vamos criar 3 services novos a partir do GitHub:

1. **trackfind-api** (backend Go - porta 8080)
2. **trackfind-worker** (worker Go - sem porta exposta)
3. **trackfind-frontend** (Nuxt 4 SSR - porta 3000)

### 1. Conectar o repositório

Em cada service, escolha **App → GitHub** e conecte o repositório
`renankeny/TrackFind`. Você precisa autorizar o EasyPanel a ler o repo.

### 2. Service `trackfind-api` (backend)

| Campo | Valor |
|-------|-------|
| Source | GitHub: `renankeny/TrackFind` |
| Branch | `main` |
| Build context | `backend/` |
| Dockerfile | `backend/Dockerfile` |
| Build target | `api` |
| Porta | `8080` |
| Domínio | `api.seu-dominio.com` (ou subpath) |

### 3. Service `trackfind-worker`

| Campo | Valor |
|-------|-------|
| Source | GitHub: `renankeny/TrackFind` |
| Branch | `main` |
| Build context | `backend/` |
| Dockerfile | `backend/Dockerfile` |
| Build target | `worker` |
| Porta | (nenhuma, é background) |

### 4. Service `trackfind-frontend`

| Campo | Valor |
|-------|-------|
| Source | GitHub: `renankeny/TrackFind` |
| Branch | `main` |
| Build context | `frontend/` |
| Dockerfile | `frontend/Dockerfile` |
| Porta | `3000` |
| Domínio | `painel.seu-dominio.com` |

### 5. Variáveis de ambiente

#### `trackfind-api` e `trackfind-worker` (mesmas envs)

```env
# === Aplicação ========================================================
APP_ENV=production
HTTP_ADDR=:8080
LOG_LEVEL=info
TZ=America/Sao_Paulo

# === URLs públicas ====================================================
APP_URL=https://painel.seu-dominio.com
PUBLIC_API_URL=https://api.seu-dominio.com/api/v1

# Host alvo para CNAMEs first-party dos clientes
TRACKING_TARGET_HOST=api.seu-dominio.com

# === Auth admin =======================================================
JWT_SECRET=<gerar com: openssl rand -hex 64>
JWT_EXPIRES_HOURS=24
ADMIN_EMAIL=seu-email@seu-dominio.com
ADMIN_PASSWORD=<senha-forte>

# === Banco (Postgres do EasyPanel) ====================================
DATABASE_URL=postgres://USER:SENHA@HOST:5432/DBNAME?sslmode=disable

# === Redis (do EasyPanel, com auth) ===================================
REDIS_ADDR=HOST:6379
REDIS_PASSWORD=<senha do redis>
REDIS_USERNAME=default
REDIS_DB=0
REDIS_TLS=false

# === Meta CAPI ========================================================
META_API_VERSION=v25.0
META_PARTNER_AGENT=trackfind-1.0.0

# === GeoIP ============================================================
GEOIP_DB_PATH=/app/data/GeoLite2-City.mmdb

# === Segurança ========================================================
STRICT_CORS=true
```

> Onde achar `DATABASE_URL` e `REDIS_ADDR` no EasyPanel:
> abra o service do Postgres → **Connection** → copie a connection string interna.
> Mesmo processo pro Redis. Use sempre os hosts internos (mais rápido e seguro).

#### `trackfind-frontend`

```env
NODE_ENV=production
TZ=America/Sao_Paulo

NUXT_PUBLIC_API_URL=https://api.seu-dominio.com/api/v1
NUXT_PUBLIC_APP_URL=https://painel.seu-dominio.com
NUXT_PUBLIC_APP_NAME=TrackFind
```

### 6. Configurar domínios

No EasyPanel, em cada service:

- **trackfind-api** → adicione domínio `api.seu-dominio.com`. Deixe **HTTPS** ativo.
- **trackfind-frontend** → adicione `painel.seu-dominio.com`. **HTTPS** ativo.

O EasyPanel emite Let's Encrypt automaticamente. No DNS aponte os subdomínios
para o IP público do seu servidor (registro **A**).

### 7. Primeira execução

Depois do deploy:

```bash
# Health check
curl https://api.seu-dominio.com/api/v1/health
```

O admin é criado automaticamente na primeira execução usando `ADMIN_EMAIL` /
`ADMIN_PASSWORD`. Se já existia um admin antigo, atualize executando o utilitário
`admin-set` (build target `admin-set` no Dockerfile) ou via SQL direto.

### 8. GeoIP (opcional mas recomendado)

A base GeoLite2-City não vai no repositório. Para baixar:

```bash
# No servidor (ou via terminal do EasyPanel)
./scripts/update-geoip.sh
```

Coloque sua `MAXMIND_LICENSE_KEY` no `.env`. O script grava o `.mmdb` em
`backend/data/`. Sem essa base o tracking funciona (só não enriquece país/cidade).

Cron sugerido (`scripts/crontab.example`):

```
CRON_TZ=America/Sao_Paulo
0 3 * * 1,5 cd /opt/trackfind && ./scripts/update-geoip.sh >> /var/log/trackfind/geoip.log 2>&1
```

### 9. Tracking first-party (CNAME do cliente)

Para que o cliente use cookies first-party (anti-adblocker), oriente ele a
configurar no DNS do domínio dele:

```
tracking.cliente.com.   CNAME   api.seu-dominio.com.
```

No painel, edite o pixel e preencha o campo **"Subdomínio first-party"** com
`tracking.cliente.com`. O snippet gerado já vai apontar pra esse subdomínio.
Use o botão **Verificar DNS** no modal de instalação pra confirmar.

## Idioma e regras do projeto

Veja `.kiro/steering/project-rules.md` e `.kiro/steering/frontend-design-guide.md`
para as convenções de código, idioma (PT-BR no chat / EN no código),
timezone obrigatória `America/Sao_Paulo` e regras de banco.

## Licença

Projeto interno. Todos os direitos reservados.
