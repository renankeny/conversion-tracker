-- =====================================================================
-- 0010_tiktok_pixel.sql
-- Adiciona suporte a TikTok Events API (paralelo ao Meta CAPI).
--
-- pixels:
--   tiktok_pixel_ids[]      - array de Pixel IDs (parallel arrays igual Meta)
--   tiktok_access_tokens[]  - tokens (paralelo a tiktok_pixel_ids)
--   tiktok_test_codes[]     - test_event_code por pixel (opcional)
--
-- tracked_users:
--   ttclid    - TikTok Click ID (vem na URL como ?ttclid=...)
--   ttp       - TikTok cookie (_ttp) capturado pelo tracker.js
--
-- Tudo idempotente. Reaplicar e seguro.
-- =====================================================================

ALTER TABLE pixels
    ADD COLUMN IF NOT EXISTS tiktok_pixel_ids     TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS tiktok_access_tokens TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS tiktok_test_codes    TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE tracked_users
    ADD COLUMN IF NOT EXISTS ttclid TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ttp    TEXT NOT NULL DEFAULT '';

-- channel diferencia eventos enviados pra Meta vs TikTok no log de auditoria.
-- Default 'meta' mantém compatibilidade com linhas anteriores.
ALTER TABLE sent_events
    ADD COLUMN IF NOT EXISTS channel TEXT NOT NULL DEFAULT 'meta';

CREATE INDEX IF NOT EXISTS idx_sent_events_channel ON sent_events(channel);
