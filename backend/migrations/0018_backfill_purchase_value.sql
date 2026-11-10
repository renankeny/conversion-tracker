-- =====================================================================
-- 0018_backfill_purchase_value.sql
-- Backfill de purchase_value para registros históricos.
--
-- Estratégia:
--   1. Extrai o valor do payload JSONB do canal 'meta' (data[0].custom_data.value).
--   2. Propaga o valor para os outros canais (tiktok, google_ads) do mesmo
--      evento (event_id + pixel_name), para que a soma não seja subestimada.
--
-- Afeta somente linhas com purchase_value = 0 e event_name relevante.
-- Idempotente: executar N vezes não altera o resultado se já foi rodado.
-- =====================================================================

-- Passo 1: atualiza os canais 'meta' a partir do payload JSONB.
UPDATE sent_events
SET purchase_value = (
    payload -> 'data' -> 0 -> 'custom_data' ->> 'value'
)::NUMERIC(10,2)
WHERE purchase_value = 0
  AND channel = 'meta'
  AND event_name IN ('Purchase', 'AddPaymentInfo', 'InitiateCheckout', 'Lead', 'AddToCart')
  AND payload IS NOT NULL
  AND (payload -> 'data' -> 0 -> 'custom_data' ->> 'value') IS NOT NULL
  AND (payload -> 'data' -> 0 -> 'custom_data' ->> 'value') <> '';

-- Passo 2: propaga o valor já preenchido do canal 'meta' para os outros
-- canais do mesmo evento (mesmo event_id + pixel_name), pra que a soma
-- de receita no dashboard não conte o mesmo evento múltiplas vezes.
-- Usa MAX porque o valor deve ser igual entre todos os canais.
UPDATE sent_events AS target
SET purchase_value = source.purchase_value
FROM (
    SELECT DISTINCT ON (event_id, pixel_name)
           event_id,
           pixel_name,
           purchase_value
    FROM sent_events
    WHERE channel = 'meta'
      AND purchase_value > 0
    ORDER BY event_id, pixel_name
) AS source
WHERE target.event_id    = source.event_id
  AND target.pixel_name  = source.pixel_name
  AND target.channel    <> 'meta'
  AND target.purchase_value = 0;
