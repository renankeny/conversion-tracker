-- =====================================================================
-- 0019_sent_events_input_json.sql
-- Adiciona input_json em sent_events para armazenar o SendEventInput
-- original serializado — permite reprocessar eventos falhos sem perder
-- o payload original que foi enviado às plataformas.
-- =====================================================================

ALTER TABLE sent_events
    ADD COLUMN IF NOT EXISTS input_json JSONB;

COMMENT ON COLUMN sent_events.input_json IS
    'domain.SendEventInput serializado no momento do processamento. Usado pelo endpoint de retry para reenfileirar eventos falhos com os mesmos dados originais.';
