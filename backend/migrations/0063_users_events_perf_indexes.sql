-- 0063_users_events_perf_indexes.sql
-- Índices aditivos e idempotentes para listagem e filtros de Usuários e Eventos.

CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_updated
    ON tracked_users (account_id, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_pixel_updated
    ON tracked_users (account_id, pixel_name, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_tracked_users_trgm_first_name
    ON tracked_users USING GIN (first_name gin_trgm_ops)
    WHERE first_name <> '';

CREATE INDEX IF NOT EXISTS idx_tracked_users_trgm_last_name
    ON tracked_users USING GIN (last_name gin_trgm_ops)
    WHERE last_name <> '';

CREATE INDEX IF NOT EXISTS idx_sent_events_acc_event_created
    ON sent_events (account_id, event_name, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sent_events_acc_channel_created
    ON sent_events (account_id, channel, created_at DESC);
