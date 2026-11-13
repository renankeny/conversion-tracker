-- Migração 0059: Marcadores de entrada por anúncio/protocolo em whatsapp_messages
-- Garante que conversas iniciadas em dias anteriores só voltem a contabilizar
-- em "Mensagens de Anúncios" / "Leads" em datas futuras quando o lead enviar
-- um novo CTWA ou um protocolo (novo ou repetido via cookie).

ALTER TABLE whatsapp_messages
    ADD COLUMN IF NOT EXISTS ctwa_clid TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS protocol_code TEXT NOT NULL DEFAULT '';

UPDATE whatsapp_messages
SET protocol_code = COALESCE(
    (regexp_match(
        content,
        '(?:protocolo(?:\s+de\s+atendimento)?|c[oó]d(?:igo)?\s+de\s+atendimento|ref(?:er[eê]ncia)?)\s*[:#\-]?\s*\*?([A-Za-z0-9_-]{6,36})\*?',
        'i'
    ))[1],
    ''
)
WHERE is_from_me = FALSE
  AND protocol_code = ''
  AND content <> ''
  AND content ~* '(protocolo|c[oó]d|ref)';

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_portal_entry
    ON whatsapp_messages(chat_id, timestamp DESC)
    WHERE is_from_me = FALSE AND (ctwa_clid <> '' OR source_id <> '' OR protocol_code <> '');

