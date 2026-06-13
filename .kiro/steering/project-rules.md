---
inclusion: always
---

# Guia de Projeto — Conversion Tracker

> Regras vivas do projeto. Sempre seguir, salvo orientação explícita do desenvolvedor responsável.
> Servem para manter consistência, legibilidade e escalabilidade entre backend (Go) e frontend (Nuxt 4, planejado).

---

## 0) Idioma

- Toda comunicação no chat, comentários significativos, mensagens de log de domínio e documentação devem ser em **português**.
- Identificadores de código (variáveis, funções, tipos, tabelas, colunas) seguem o que já existe no projeto: **inglês**, em snake_case no banco e camelCase/PascalCase no Go conforme a convenção da linguagem.

---

## 1) Estrutura — Backend Go (atual)

```
backend/
├─ cmd/
│  ├─ api/main.go         # binário HTTP
│  └─ worker/main.go      # binário consumidor da fila
├─ migrations/            # SQL idempotente, ordem alfabética
└─ internal/
   ├─ auth/               # JWT + bcrypt + claims em ctx
   ├─ config/             # carrega env
   ├─ db/                 # pool pgx + migrate
   ├─ domain/             # tipos de domínio puros
   ├─ geoip/              # MaxMind GeoLite2
   ├─ handlers/           # handlers HTTP (events, webhooks, admin)
   ├─ httpserver/         # router chi + middlewares
   ├─ meta/               # cliente Meta CAPI
   ├─ queue/              # asynq enqueuer + worker
   ├─ repo/               # acesso a Postgres
   └─ util/               # helpers puros (normalização, hash)
```

Regras:
- **`internal/`** esconde símbolos privados ao módulo. Não expor o que não precisa ser reusado fora.
- **Um pacote = uma responsabilidade clara.** Se um pacote começa a misturar coisas, quebrar.
- **Sem ciclos de import.** Se aparecer ciclo, repensar a fronteira (claims context vive em `auth`, por exemplo).
- **Sem ORM.** SQL cru com `pgx`. Repos isolam acesso a dados.
- **Logs estruturados** com `slog`, chaves consistentes (`pixel_name`, `event_id`, `external_id`, `error`).
- **Erros propagados** com `fmt.Errorf("contexto: %w", err)`.
- **Handlers** retornam JSON via helpers (`writeJSON` / `writeError`).
- **Validação** de payload no handler (camada UI), regra de negócio nos pacotes de domínio/repo/queue.
- **Camadas:** `handlers` → `repo`/`queue` → `meta`/`db`. Handlers nunca falam direto com Postgres ou Meta.

---

## 2) Estrutura — Frontend Nuxt 4 (planejado)

Quando o painel for implementado, seguir esta estrutura:

```
frontend/
├─ app/
│  ├─ assets/         # fontes, ícones, imagens, CSS global
│  ├─ components/     # UI pequena, reusável, sem lógica de dados
│  ├─ composables/    # useX() — busca de dados, estado, regras
│  ├─ layouts/        # DefaultLayout.vue, AdminLayout.vue
│  ├─ middleware/     # authGuard.ts, isAdmin.ts
│  ├─ pages/          # rotas baseadas em arquivo
│  ├─ plugins/        # registro de libs (client/server)
│  ├─ utils/          # funções puras, sem reatividade
│  ├─ app.vue
│  ├─ app.config.ts
│  └─ error.vue
├─ content/           # opcional
├─ public/            # estáticos servidos em /
├─ shared/
│  ├─ types/          # DTOs, contratos da API Go
│  └─ constants/      # enums e constantes globais
├─ server/
│  ├─ api/
│  ├─ middleware/
│  └─ plugins/
├─ tailwind.config.ts
├─ nuxt.config.ts
└─ .env
```

Por ser um projeto médio com backend Go separado, **`shared/types`** fica fora de `app/` para facilitar compartilhamento de DTOs.

---

## 3) Princípios de arquitetura (front + back)

1. **Componentizar / modularizar ao máximo.** Componentes pequenos, coesos, reusáveis.
2. **Lógica de domínio fora da UI.**
   - Frontend: composables (`useAuth`, `usePixels`).
   - Backend: pacotes de domínio/repo/queue, nunca em handlers.
3. **Responsabilidade única.** Um arquivo faz uma coisa bem feita. Se cresce demais, quebrar.
4. **Tipagem correta.**
   - Go: tipar tudo, evitar `interface{}` exceto onde realmente necessário.
   - Vue/TS: tipar props, emits, retornos, estados, contratos de API. Evitar `any`.
5. **Sempre TypeScript no frontend.** `lang="ts"` nos `.vue`. Tipos globais em `shared/types`.
6. **Camadas claras.** UI → Composables/Handlers → Acesso a dados (`server/api`, `repo`, `queue`).
7. **Componentes Vue com `id` fixo** quando há risco de hidratação inconsistente (listas dinâmicas, refs).

---

## 4) Nomenclatura

### Backend Go
- **Pacotes:** minúsculas, sem underscore (`auth`, `httpserver`, `meta`).
- **Tipos exportados:** PascalCase (`PixelRepo`, `EventPayload`).
- **Funções/métodos:** camelCase (privados) ou PascalCase (exportados).
- **Arquivos:** snake_case (`pixel_repo.go`, `admin_auth.go`).
- **Variáveis:** camelCase, descritivas. Sem abreviações obscuras.
- **Constantes:** PascalCase para exportadas, camelCase para privadas. Não usar `SCREAMING_CASE`.

### Banco de dados
- **Tabelas:** plural, snake_case (`tracked_users`, `sent_events`).
- **Colunas:** snake_case (`first_name`, `created_at`).
- **Índices:** `idx_<tabela>_<colunas>` (`idx_tracked_users_phone`).
- **PKs:** `id BIGSERIAL PRIMARY KEY`.
- **Timestamps:** `created_at` e `updated_at` em `TIMESTAMPTZ NOT NULL DEFAULT NOW()`.
- **Texto opcional:** `TEXT NOT NULL DEFAULT ''` (evita `sql.NullString` no Go).

### Frontend Nuxt
- **Componentes Vue (`app/components`)** → PascalCase (`UserCard.vue`, `PixelForm.vue`).
- **Páginas (`app/pages`)** → minúsculas, sem traços, letras apenas. Subpastas pra contexto:
  - `app/pages/login.vue`
  - `app/pages/admin/dashboard.vue`
  - `app/pages/admin/pixels.vue`
- **Layouts (`app/layouts`)** → PascalCase (`DefaultLayout.vue`, `AdminLayout.vue`). Usar via `<NuxtLayout>`.
- **Composables (`app/composables`)** → `use` + PascalCase (`useAuth.ts`, `usePixels.ts`).
- **Middlewares (`app/middleware`)** → camelCase (`authGuard.ts`, `isAdmin.ts`).
- **Utils (`app/utils`)** → camelCase (`formatDate.ts`, `maskToken.ts`).
- **Tipos (`shared/types`)** → PascalCase para interfaces/DTOs (`PixelDTO.ts`, `EventDTO.ts`).

### Imports no Vue/TS
- **Sempre imports explícitos.** Evitar auto-imports do Nuxt no código de domínio para deixar dependências visíveis e facilitar refatorações.

---

## 5) Banco de dados — regras críticas

### 5.1 Espelho oficial: `BANCO DE DADOS.sql`

Existe um arquivo `BANCO DE DADOS.sql` na **raiz do projeto** que representa o **estado completo e atual** do schema (CREATE TABLE, índices, constraints, seeds estruturais).

Sempre que uma alteração de schema for feita (nova migração, ALTER, novo índice, nova tabela, mudança de coluna), **atualizar o `BANCO DE DADOS.sql` no mesmo commit/turno** para refletir o estado final do banco. Esse arquivo é a fonte de verdade visual; as migrações em `backend/migrations/` continuam sendo o que roda na inicialização.

Fluxo padrão para mudança de schema:
1. Criar nova migração idempotente em `backend/migrations/000N_xxx.sql` (`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, etc.).
2. Atualizar `BANCO DE DADOS.sql` com o estado final consolidado.
3. Atualizar tipos em `internal/domain/` se a mudança for relevante para o código.

### 5.2 Proibição de exclusão sem dupla confirmação

**Nunca apagar dados ou estruturas do banco em produção/desenvolvimento sem dupla confirmação explícita do usuário.** Vale para:

- `DROP TABLE`, `DROP COLUMN`, `DROP INDEX`, `DROP CONSTRAINT`, `DROP DATABASE`, `DROP SCHEMA`.
- `DELETE FROM ...` em massa (sem `WHERE` específico ou afetando muitas linhas).
- `TRUNCATE`.
- Qualquer migração destrutiva.
- Comandos `docker compose down -v` ou similares que removem volumes.

Protocolo:
1. Pedir a primeira confirmação descrevendo **o que será apagado** e **o impacto** (irreversibilidade, dados afetados, dependências).
2. Aguardar confirmação explícita do usuário.
3. Pedir a **segunda confirmação** repetindo o resumo.
4. Só então executar. Mesmo assim, preferir alternativas não destrutivas (soft delete via flag `deleted_at`, renomear coluna em vez de remover, etc.).

Em qualquer dúvida, **não apagar.** É melhor perguntar de novo do que destruir dado.

### 5.3 Migrações
- Tudo idempotente: `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`, `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`.
- Ordem alfabética (`0001_`, `0002_`, ...). Nunca editar uma migração já aplicada em produção; criar uma nova.
- Sem rollback automático. Se precisar desfazer, criar uma migração reversa explícita (e ainda assim aplicar a regra 5.2 se for destrutiva).

---

## 6) Convenções de API

- Versionar com prefixo `/api/v1/...`.
- Sucesso: `200 OK`, `201 Created`, `202 Accepted` (eventos enfileirados).
- Erro: estrutura `{ "error": "mensagem legível", "code": "machine_readable" }`.
- Auth admin: `Authorization: Bearer <jwt>`.
- CORS configurável por origem em produção. `*` só em dev.
- Timeouts: 15s global, 5s read header.

---

## 7) Quando criar / quando perguntar

- **Criar sem perguntar:** novos handlers, repos, composables, migrações aditivas idempotentes, testes, refactors locais.
- **Perguntar antes:** mudanças destrutivas no banco, mudança de contrato de API pública (endpoints `/events/*`), mudança de stack (libs novas relevantes), exclusão de arquivos significativos.
- **Sempre confirmar duas vezes:** qualquer operação que apague dados ou estruturas no banco.

---

## 8) Estado atual (resumo, ver `CONTEXTO.md` para detalhes)

- Backend Go compilando limpo, com endpoints `/events/*`, webhooks Kiwify e Hotmart, admin (login + CRUD pixels), fila assíncrona, integração Meta CAPI.
- Frontend Nuxt 4 ainda **não implementado** — quando entrar, seguir as regras das seções 2 e 4.
- Foco atual: somente Meta Ads. Google Ads é fase 2.

---

## 9) Timezone obrigatória — `America/Sao_Paulo`

Regra rígida do projeto: **todo o sistema opera em horário de São Paulo.** Sem exceções.

Pontos onde isso é aplicado (todos devem permanecer assim):
- **Containers Docker (api, worker, postgres, redis):** env `TZ=America/Sao_Paulo` no `docker-compose.yml`. Postgres recebe também `PGTZ=America/Sao_Paulo` e é iniciado com `command: ["postgres", "-c", "timezone=America/Sao_Paulo"]`.
- **Imagens Docker (Dockerfile):** instalam `tzdata`, definem `ENV TZ=America/Sao_Paulo` e fazem `cp /usr/share/zoneinfo/America/Sao_Paulo /etc/localtime`.
- **Binários Go:** chamam `config.MustSetTimezone()` no início de `cmd/api/main.go` e `cmd/worker/main.go`. Define `time.Local` como `America/Sao_Paulo` mesmo se a env `TZ` não estiver presente.
- **Cron / scheduled jobs:** crontabs sempre começam com `CRON_TZ=America/Sao_Paulo` (ver `scripts/crontab.example`).

Se for criar binário novo (CLI, worker auxiliar, script de manutenção em Go), chame `config.MustSetTimezone()` antes de qualquer uso de `time.Now()`. Em scripts shell que rodem dentro de containers, a env `TZ` já está setada.

No banco, todos os timestamps são `TIMESTAMPTZ` armazenados em UTC internamente, mas exibidos e comparados na sessão como `America/Sao_Paulo` (graças à configuração do servidor Postgres). NUNCA usar `TIMESTAMP WITHOUT TIME ZONE`.

---

## 10) Operações periódicas (cron)

Tarefas agendadas vivem em `scripts/crontab.example`. Todas com `CRON_TZ=America/Sao_Paulo`.

Atual:
- **Atualização da base GeoIP MaxMind** — `./scripts/update-geoip.sh`, segunda e sexta às **03:00** (horário de São Paulo).

Quando adicionar novo job:
1. Adicionar a linha no `scripts/crontab.example` com explicação curta no comentário.
2. Manter `CRON_TZ=America/Sao_Paulo` no topo do arquivo.
3. Usar caminho absoluto (`cd /opt/conversion-tracker && ./scripts/foo.sh`) e redirecionar stdout/stderr pra `/var/log/conversion-tracker/<job>.log`.
4. Atualizar essa seção do steering com a nova entrada.
