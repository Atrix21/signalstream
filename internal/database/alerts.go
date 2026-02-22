package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Alert struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	StrategyID      uuid.UUID `json:"strategy_id"`
	EventID         string    `json:"event_id"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	URL             string    `json:"url"`
	SimilarityScore float64   `json:"similarity_score"`
	IsRead          bool      `json:"is_read"`
	CreatedAt       time.Time `json:"created_at"`
}

func (db *DB) CreateAlert(ctx context.Context, alert *Alert) error {
	query := `
		INSERT INTO alerts (user_id, strategy_id, event_id, title, content, url, similarity_score, is_read)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	return db.QueryRowContext(ctx, query,
		alert.UserID, alert.StrategyID, alert.EventID,
		alert.Title, alert.Content, alert.URL,
		alert.SimilarityScore, alert.IsRead,
	).Scan(&alert.ID, &alert.CreatedAt)
}

func (db *DB) GetUserAlerts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Alert, error) {
	query := `
		SELECT id, user_id, strategy_id, event_id, title, content, url, similarity_score, is_read, created_at
		FROM alerts
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.StrategyID, &a.EventID,
			&a.Title, &a.Content, &a.URL, &a.SimilarityScore,
			&a.IsRead, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

func (db *DB) MarkAlertRead(ctx context.Context, alertID, userID uuid.UUID) error {
	query := `
		UPDATE alerts
		SET is_read = TRUE
		WHERE id = $1 AND user_id = $2
	`
	res, err := db.ExecContext(ctx, query, alertID, userID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
