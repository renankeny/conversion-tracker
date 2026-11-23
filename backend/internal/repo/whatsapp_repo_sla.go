package repo

import "context"

// RefreshChatSLAMetrics recalcula de forma atômica e indexada as métricas de SLA
// (1ª resposta, tempo médio por turno, última mensagem do lead/atendente e sla_status)
// de uma conversa específica após inserção de nova mensagem.
func (r *WhatsAppRepo) RefreshChatSLAMetrics(ctx context.Context, chatID int64) error {
	if chatID <= 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		WITH ordered_msgs AS (
			SELECT
				m.is_from_me,
				m.timestamp,
				COUNT(*) FILTER (WHERE m.is_from_me = TRUE) OVER (
					ORDER BY m.timestamp ASC, m.id ASC
					ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING
				) AS turn_grp
			FROM whatsapp_messages m
			WHERE m.chat_id = $1
		),
		turn_stats AS (
			SELECT
				turn_grp,
				MIN(timestamp) FILTER (WHERE is_from_me = FALSE) AS turn_in_at,
				MIN(timestamp) FILTER (WHERE is_from_me = TRUE)  AS turn_out_at
			FROM ordered_msgs
			GROUP BY turn_grp
		),
		avg_turns AS (
			SELECT ROUND(AVG(GREATEST(0, EXTRACT(EPOCH FROM (turn_out_at - turn_in_at)))))::INT AS avg_resp_sec
			FROM turn_stats
			WHERE turn_in_at IS NOT NULL AND turn_out_at IS NOT NULL AND turn_out_at >= turn_in_at
		),
		base_agg AS (
			SELECT
				COUNT(*)::INT AS total_msgs,
				COUNT(*) FILTER (WHERE is_from_me = FALSE)::INT AS inbound_msgs,
				COUNT(*) FILTER (WHERE is_from_me = TRUE)::INT AS outbound_msgs,
				MIN(timestamp) FILTER (WHERE is_from_me = FALSE) AS first_in_at,
				MAX(timestamp) FILTER (WHERE is_from_me = FALSE) AS last_in_at,
				MAX(timestamp) FILTER (WHERE is_from_me = TRUE)  AS last_out_at
			FROM whatsapp_messages
			WHERE chat_id = $1
		),
		first_reply AS (
			SELECT MIN(m.timestamp) AS first_out_after_in
			FROM base_agg b
			JOIN whatsapp_messages m
			  ON m.chat_id = $1
			 AND m.is_from_me = TRUE
			 AND b.first_in_at IS NOT NULL
			 AND m.timestamp >= b.first_in_at
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
		CROSS JOIN first_reply fr
		CROSS JOIN avg_turns at
		WHERE c.id = $1
	`, chatID)
	return err
}

func (r *WhatsAppRepo) CountWaitingConversations(ctx context.Context, accountID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::INT
		FROM whatsapp_chats c
		JOIN whatsapp_instances i ON i.id = c.instance_id
		WHERE i.account_id = $1
		  AND c.tracked_user_id IS NOT NULL
		  AND c.sla_status IN ('unanswered', 'waiting_reply')
	`, accountID).Scan(&count)
	return count, err
}
