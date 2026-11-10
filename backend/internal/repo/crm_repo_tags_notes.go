package repo

import (
	"context"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func (r *CRMRepo) AddNote(ctx context.Context, note *domain.CRMNote) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO crm_notes (crm_lead_id, content)
		VALUES ($1, $2)
		RETURNING id, created_at
	`, note.CRMLeadID, note.Content).Scan(&note.ID, &note.CreatedAt)
}

func (r *CRMRepo) DeleteNote(ctx context.Context, noteID, accountID int64) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM crm_notes n
		USING crm_leads l
		WHERE n.id = $1 AND n.crm_lead_id = l.id AND l.account_id = $2
	`, noteID, accountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CRMRepo) ListTags(ctx context.Context, accountID int64) ([]domain.CRMTag, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, account_id, name, color, created_at FROM crm_tags WHERE account_id = $1 ORDER BY name ASC", accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.CRMTag
	for rows.Next() {
		var t domain.CRMTag
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *CRMRepo) CreateTag(ctx context.Context, tag *domain.CRMTag) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO crm_tags (account_id, name, color)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, tag.AccountID, tag.Name, tag.Color).Scan(&tag.ID, &tag.CreatedAt)
}

func (r *CRMRepo) DeleteTag(ctx context.Context, tagID, accountID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM crm_tags WHERE id = $1 AND account_id = $2`, tagID, accountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CRMRepo) AssignTag(ctx context.Context, accountID, leadID, tagID int64) error {
	var validTag bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM crm_tags WHERE id = $1 AND account_id = $2)`, tagID, accountID).Scan(&validTag); err != nil {
		return err
	}
	if !validTag {
		return ErrNotFound
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO crm_lead_tags (crm_lead_id, crm_tag_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, leadID, tagID)
	return err
}

func (r *CRMRepo) RemoveTag(ctx context.Context, accountID, leadID, tagID int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM crm_lead_tags lt
		USING crm_tags t
		WHERE lt.crm_lead_id = $1 AND lt.crm_tag_id = $2 AND t.id = $2 AND t.account_id = $3
	`, leadID, tagID, accountID)
	return err
}

