-- =====================================================================
-- 0047_traffic_audit_indexes.sql
--
-- Índices para acelerar a Auditoria de Tráfego e contagem de visitantes
-- filtradas por período (created_at e last_seen_at) e conta/pixel.
-- =====================================================================

CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_last_seen
    ON tracked_users(account_id, last_seen_at DESC);

CREATE INDEX IF NOT EXISTS idx_tracked_users_acc_created
    ON tracked_users(account_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tracked_users_pixel_last_seen
    ON tracked_users(pixel_name, last_seen_at DESC);

CREATE INDEX IF NOT EXISTS idx_tracked_users_pixel_created
    ON tracked_users(pixel_name, created_at DESC);
