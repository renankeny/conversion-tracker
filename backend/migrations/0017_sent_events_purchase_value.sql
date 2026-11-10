-- =====================================================================
-- 0017_sent_events_purchase_value.sql
-- Adiciona coluna purchase_value em sent_events para exibir o valor
-- da compra diretamente na listagem e no dashboard, sem precisar
-- fazer parse do payload JSONB a cada request.
-- =====================================================================

ALTER TABLE sent_events
    ADD COLUMN IF NOT EXISTS purchase_value NUMERIC(10,2) NOT NULL DEFAULT 0;

COMMENT ON COLUMN sent_events.purchase_value IS
    'Valor monetário do evento (ex: 99.90 para Purchase). 0 quando não aplicável.';
