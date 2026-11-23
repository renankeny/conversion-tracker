-- 0070_whatsapp_chat_sla_metrics.sql
-- Métricas comerciais pré-calculadas de SLA de atendimento nas conversas do WhatsApp.

ALTER TABLE whatsapp_chats
    ADD COLUMN IF NOT EXISTS total_messages INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS inbound_messages INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS outbound_messages INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS first_inbound_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS first_outbound_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_inbound_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_outbound_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS first_response_sec INT,
    ADD COLUMN IF NOT EXISTS avg_response_sec INT,
    ADD COLUMN IF NOT EXISTS sla_status VARCHAR(20) NOT NULL DEFAULT 'unanswered';

CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_account_last_msg
    ON whatsapp_chats(account_id, last_message_time DESC)
    WHERE tracked_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_whatsapp_chats_account_sla_last_msg
    ON whatsapp_chats(account_id, sla_status, last_message_time DESC)
    WHERE tracked_user_id IS NOT NULL;

-- Backfill idempotente das conversas existentes com base no histórico de whatsapp_messages.
WITH ordered_msgs AS (
    SELECT
        m.chat_id,
        m.is_from_me,
        m.timestamp,
        COUNT(*) FILTER (WHERE m.is_from_me = TRUE) OVER (
            PARTITION BY m.chat_id
            ORDER BY m.timestamp ASC, m.id ASC
            ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING
        ) AS turn_grp
    FROM whatsapp_messages m
),
turn_stats AS (
    SELECT
        chat_id,
        turn_grp,
        MIN(timestamp) FILTER (WHERE is_from_me = FALSE) AS turn_in_at,
        MIN(timestamp) FILTER (WHERE is_from_me = TRUE)  AS turn_out_at
    FROM ordered_msgs
    GROUP BY chat_id, turn_grp
),
avg_turns AS (
    SELECT
        chat_id,
        ROUND(AVG(GREATEST(0, EXTRACT(EPOCH FROM (turn_out_at - turn_in_at)))))::INT AS avg_resp_sec
    FROM turn_stats
    WHERE turn_in_at IS NOT NULL AND turn_out_at IS NOT NULL AND turn_out_at >= turn_in_at
    GROUP BY chat_id
),
base_agg AS (
    SELECT
        m.chat_id,
        COUNT(*)::INT AS total_msgs,
        COUNT(*) FILTER (WHERE m.is_from_me = FALSE)::INT AS inbound_msgs,
        COUNT(*) FILTER (WHERE m.is_from_me = TRUE)::INT AS outbound_msgs,
        MIN(m.timestamp) FILTER (WHERE m.is_from_me = FALSE) AS first_in_at,
        MAX(m.timestamp) FILTER (WHERE m.is_from_me = FALSE) AS last_in_at,
        MAX(m.timestamp) FILTER (WHERE m.is_from_me = TRUE)  AS last_out_at
    FROM whatsapp_messages m
    GROUP BY m.chat_id
),
first_reply AS (
    SELECT
        b.chat_id,
        MIN(m.timestamp) AS first_out_after_in
    FROM base_agg b
    JOIN whatsapp_messages m
      ON m.chat_id = b.chat_id
     AND m.is_from_me = TRUE
     AND b.first_in_at IS NOT NULL
     AND m.timestamp >= b.first_in_at
    GROUP BY b.chat_id
)
UPDATE whatsapp_chats c
SET
    total_messages     = b.total_msgs,
    inbound_messages   = b.inbound_msgs,
    outbound_messages  = b.outbound_msgs,
    first_inbound_at   = b.first_in_at,
    first_outbound_at  = fr.first_out_after_in,
    last_inbound_at    = b.last_in_at,
    last_outbound_at   = b.last_out_at,
    first_response_sec = CASE
        WHEN b.first_in_at IS NOT NULL AND fr.first_out_after_in IS NOT NULL
        THEN GREATEST(0, EXTRACT(EPOCH FROM (fr.first_out_after_in - b.first_in_at))::INT)
        ELSE NULL
    END,
    avg_response_sec   = at.avg_resp_sec,
    sla_status         = CASE
        WHEN b.outbound_msgs = 0 THEN 'unanswered'
        WHEN b.last_in_at IS NOT NULL AND (b.last_out_at IS NULL OR b.last_in_at > b.last_out_at) THEN 'waiting_reply'
        ELSE 'replied'
    END
FROM base_agg b
LEFT JOIN first_reply fr ON fr.chat_id = b.chat_id
LEFT JOIN avg_turns at   ON at.chat_id = b.chat_id
WHERE c.id = b.chat_id;
