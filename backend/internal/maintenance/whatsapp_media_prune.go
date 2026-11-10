package maintenance

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ObjectBatchDeleter desacopla o pacote maintenance da implementação concreta S3/MinIO.
type ObjectBatchDeleter interface {
	Enabled() bool
	DeleteBatch(ctx context.Context, objectKeys []string) error
}

// ExpireOldWhatsAppMedia remove apenas os binários de mídia no MinIO após N dias (default 14),
// preservando 100% das mensagens de texto, legendas e metadados no Postgres indefinidamente.
func ExpireOldWhatsAppMedia(
	ctx context.Context,
	pool *pgxpool.Pool,
	deleter ObjectBatchDeleter,
	retentionDays int,
) (PruneResult, error) {
	if retentionDays <= 0 {
		retentionDays = 14
	}
	now := time.Now().UTC()
	fallbackThreshold := now.AddDate(0, 0, -retentionDays)
	if pool == nil {
		return PruneResult{Table: "whatsapp_messages_media", RetentionDays: retentionDays, Threshold: fallbackThreshold}, nil
	}

	const selectQ = `
		SELECT id, COALESCE(media_object_key, '')
		FROM whatsapp_messages
		WHERE media_status = 'ready'
		  AND (
		    (media_expires_at IS NOT NULL AND media_expires_at < $1)
		    OR (media_expires_at IS NULL AND timestamp < $2)
		  )
		ORDER BY id ASC
		LIMIT 500
	`
	const updateQ = `
		UPDATE whatsapp_messages
		SET media_status = 'expired',
		    media_object_key = NULL,
		    media_url = NULL
		WHERE id = ANY($1)
	`

	start := time.Now()
	var totalExpired int64

	for {
		if err := ctx.Err(); err != nil {
			return PruneResult{
				Table:         "whatsapp_messages_media",
				RetentionDays: retentionDays,
				Threshold:     fallbackThreshold,
				Deleted:       totalExpired,
				Duration:      time.Since(start),
			}, err
		}

		rows, err := pool.Query(ctx, selectQ, now, fallbackThreshold)
		if err != nil {
			return PruneResult{}, fmt.Errorf("select expired whatsapp media: %w", err)
		}

		ids := make([]int64, 0, 500)
		keys := make([]string, 0, 500)
		for rows.Next() {
			var id int64
			var key string
			if scanErr := rows.Scan(&id, &key); scanErr != nil {
				rows.Close()
				return PruneResult{}, fmt.Errorf("scan expired whatsapp media: %w", scanErr)
			}
			ids = append(ids, id)
			if key != "" {
				keys = append(keys, key)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return PruneResult{}, fmt.Errorf("iterate expired whatsapp media: %w", err)
		}

		if len(ids) == 0 {
			break
		}

		if len(keys) > 0 && deleter != nil && deleter.Enabled() {
			if err := deleter.DeleteBatch(ctx, keys); err != nil {
				return PruneResult{}, fmt.Errorf("delete minio media batch: %w", err)
			}
		}

		tag, err := pool.Exec(ctx, updateQ, ids)
		if err != nil {
			return PruneResult{}, fmt.Errorf("mark whatsapp media expired: %w", err)
		}
		totalExpired += tag.RowsAffected()

		if len(ids) < 500 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	return PruneResult{
		Table:         "whatsapp_messages_media",
		RetentionDays: retentionDays,
		Threshold:     fallbackThreshold,
		Deleted:       totalExpired,
		Duration:      time.Since(start),
	}, nil
}

