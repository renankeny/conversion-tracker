-- 0005_event_counters.sql
--
-- Adiciona tabela `event_counters` para agregação de eventos não-críticos
-- (PageView, ViewContent, scroll, timer, etc) sem persistir payload.
--
-- Estratégia de armazenamento:
--   - sent_events  → eventos importantes (Lead, InitiateCheckout, Purchase, ...)
--                    com payload completo, retenção curta.
--   - event_counters → contadores agregados por (pixel, event, hora)
--                      com retenção longa, ocupação mínima.
--
-- Bucket por hora (truncado para o início da hora) permite gráficos detalhados
-- e ainda mantém a tabela enxuta: ~24 rows por (pixel, evento) por dia.
--
-- Retenção sugerida: 90 dias (configurável via ENV no worker de prune).

CREATE TABLE IF NOT EXISTS event_counters (
    id            BIGSERIAL PRIMARY KEY,
    pixel_name    TEXT NOT NULL,
    event_name    TEXT NOT NULL,
    bucket_at     TIMESTAMPTZ NOT NULL,
    count_total   BIGINT NOT NULL DEFAULT 0,
    count_sent    BIGINT NOT NULL DEFAULT 0,
    count_failed  BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_event_counters_bucket UNIQUE (pixel_name, event_name, bucket_at)
);

CREATE INDEX IF NOT EXISTS idx_event_counters_bucket
    ON event_counters(bucket_at DESC);

CREATE INDEX IF NOT EXISTS idx_event_counters_pixel_event_bucket
    ON event_counters(pixel_name, event_name, bucket_at DESC);

-- Índice em sent_events.created_at também é útil para a regra de retenção
-- (DELETE WHERE created_at < threshold).
CREATE INDEX IF NOT EXISTS idx_sent_events_created_at
    ON sent_events(created_at DESC);

COMMENT ON TABLE event_counters IS
    'Contadores agregados por (pixel, evento, hora). Eventos não-críticos vão para cá em vez de sent_events para não inflar o banco.';
COMMENT ON COLUMN event_counters.bucket_at IS
    'Início da hora (date_trunc(hour, ...)). Garante UPSERT determinístico.';
COMMENT ON COLUMN event_counters.count_total IS
    'Total de eventos enfileirados nessa hora (independente de sucesso).';
COMMENT ON COLUMN event_counters.count_sent IS
    'Quantos chegaram com sucesso na Meta CAPI.';
COMMENT ON COLUMN event_counters.count_failed IS
    'Quantos falharam (erro de rede, 4xx, 5xx).';
