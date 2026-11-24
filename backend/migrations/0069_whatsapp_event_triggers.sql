-- =====================================================================
-- 0069_whatsapp_event_triggers.sql
-- Lista dinâmica de gatilhos de eventos e ticket médio padrão no WhatsApp
-- =====================================================================

ALTER TABLE whatsapp_instances
    ADD COLUMN IF NOT EXISTS event_triggers JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS average_ticket NUMERIC(12,2) NOT NULL DEFAULT 0;

UPDATE whatsapp_instances
SET event_triggers = (
    SELECT COALESCE(jsonb_agg(elem ORDER BY ord), '[]'::jsonb)
    FROM (
        SELECT jsonb_build_object('event_name', 'Purchase', 'trigger', btrim(purchase_trigger)) AS elem, 1 AS ord
        WHERE btrim(purchase_trigger) <> ''
        UNION ALL
        SELECT jsonb_build_object('event_name', 'QualifiedLead', 'trigger', btrim(lead_trigger)) AS elem, 2 AS ord
        WHERE btrim(lead_trigger) <> ''
        UNION ALL
        SELECT jsonb_build_object('event_name', btrim(custom_event_name), 'trigger', btrim(custom_event_trigger)) AS elem, 3 AS ord
        WHERE btrim(custom_event_name) <> '' AND btrim(custom_event_trigger) <> ''
    ) t
)
WHERE event_triggers = '[]'::jsonb
  AND (
      btrim(purchase_trigger) <> ''
      OR btrim(lead_trigger) <> ''
      OR (btrim(custom_event_name) <> '' AND btrim(custom_event_trigger) <> '')
  );
