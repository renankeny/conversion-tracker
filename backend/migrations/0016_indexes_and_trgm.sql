-- =====================================================================
-- 0016_indexes_and_trgm.sql
-- Otimizações de índices:
--   1. Remove índice duplicado em tracked_users.email
--   2. Adiciona índice composto (account_id, pixel_name) em tracked_users
--   3. Adiciona índice composto (account_id, pixel_name, created_at) em sent_events
--   4. Ativa extensão pg_trgm e adiciona índices GIN para busca ILIKE
-- =====================================================================

-- Extensão pg_trgm para suporte a busca trigram (ILIKE sem seq scan).
-- Idempotente: CREATE EXTENSION IF NOT EXISTS não falha se já existir.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ---------------------------------------------------------------------
-- tracked_users
-- ---------------------------------------------------------------------

-- 1. Remove o índice duplicado (idêntico ao idx_tracked_users_email).
DROP INDEX IF EXISTS idx_tracked_users_email_lookup;

-- 2. Índice composto para o filtro mais comum no painel:
--    List(account_id, pixel_name) — evita seq scan + filter.
CREATE INDEX IF NOT EXISTS idx_tracked_users_account_pixel
    ON tracked_users(account_id, pixel_name);

-- 3. GIN trigram para busca ILIKE nos campos de identidade.
--    Cobre: external_id ILIKE, id_wpp ILIKE, email ILIKE, phone ILIKE.
CREATE INDEX IF NOT EXISTS idx_tracked_users_trgm_external_id
    ON tracked_users USING GIN (external_id gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_tracked_users_trgm_id_wpp
    ON tracked_users USING GIN (id_wpp gin_trgm_ops)
    WHERE id_wpp <> '';

CREATE INDEX IF NOT EXISTS idx_tracked_users_trgm_email
    ON tracked_users USING GIN (email gin_trgm_ops)
    WHERE email <> '';

CREATE INDEX IF NOT EXISTS idx_tracked_users_trgm_phone
    ON tracked_users USING GIN (phone gin_trgm_ops)
    WHERE phone <> '';

-- ---------------------------------------------------------------------
-- sent_events
-- ---------------------------------------------------------------------

-- 4. Índice composto para o filtro mais comum no painel:
--    List(account_id, pixel_name) + ordenação por created_at DESC.
--    Superset do idx_sent_events_pixel para queries com account_id.
CREATE INDEX IF NOT EXISTS idx_sent_events_account_pixel_created
    ON sent_events(account_id, pixel_name, created_at DESC);
