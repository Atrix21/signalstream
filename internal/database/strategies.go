package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Strategy struct {
	ID                  uuid.UUID `json:"id"`
	UserID              uuid.UUID `json:"user_id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	Query               string    `json:"query"`
	Source              []string  `json:"source"`
	Tickers             []string  `json:"tickers"`
	SimilarityThreshold float64   `json:"similarity_threshold"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	OwnerEmail          string    `json:"owner_email,omitempty"` // Populated via JOIN
}

func (db *DB) CreateStrategy(ctx context.Context, s *Strategy) error {
	query := `
		INSERT INTO strategies (user_id, name, description, query, source, tickers, similarity_threshold, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	return db.QueryRowContext(ctx, query,
		s.UserID, s.Name, s.Description, s.Query,
		pq.Array(s.Source), pq.Array(s.Tickers), s.SimilarityThreshold, s.IsActive,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (db *DB) GetUserStrategies(ctx context.Context, userID uuid.UUID) ([]Strategy, error) {
	query := `
		SELECT id, user_id, name, description, query, source, tickers, similarity_threshold, is_active, created_at, updated_at
		FROM strategies
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []Strategy
	for rows.Next() {
		var s Strategy
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Name, &s.Description, &s.Query,
			pq.Array(&s.Source), pq.Array(&s.Tickers), &s.SimilarityThreshold, &s.IsActive,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		strategies = append(strategies, s)
	}
	return strategies, nil
}

func (db *DB) GetAllActiveStrategies(ctx context.Context) ([]Strategy, error) {
	query := `
		SELECT s.id, s.user_id, s.name, s.description, s.query, s.source, s.tickers, s.similarity_threshold, s.is_active, s.created_at, s.updated_at, u.email
		FROM strategies s
		JOIN users u ON s.user_id = u.id
		WHERE s.is_active = TRUE
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []Strategy
	for rows.Next() {
		var s Strategy
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Name, &s.Description, &s.Query,
			pq.Array(&s.Source), pq.Array(&s.Tickers), &s.SimilarityThreshold, &s.IsActive,
			&s.CreatedAt, &s.UpdatedAt, &s.OwnerEmail,
		); err != nil {
			return nil, err
		}
		strategies = append(strategies, s)
	}
	return strategies, nil
}

func (db *DB) UpdateStrategy(ctx context.Context, s *Strategy) error {
	query := `
		UPDATE strategies
		SET name = $1, description = $2, query = $3, source = $4, tickers = $5, similarity_threshold = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8 AND user_id = $9
		RETURNING updated_at
	`
	return db.QueryRowContext(ctx, query,
		s.Name, s.Description, s.Query, pq.Array(s.Source), pq.Array(s.Tickers), s.SimilarityThreshold, s.IsActive,
		s.ID, s.UserID,
	).Scan(&s.UpdatedAt)
}

func (db *DB) DeleteStrategy(ctx context.Context, id, userID uuid.UUID) error {
	query := `DELETE FROM strategies WHERE id = $1 AND user_id = $2`
	res, err := db.ExecContext(ctx, query, id, userID)
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

func (db *DB) GetStrategy(ctx context.Context, id, userID uuid.UUID) (*Strategy, error) {
	query := `
		SELECT id, user_id, name, description, query, source, tickers, similarity_threshold, is_active, created_at, updated_at
		FROM strategies
		WHERE id = $1 AND user_id = $2
	`
	var s Strategy
	if err := db.QueryRowContext(ctx, query, id, userID).Scan(
		&s.ID, &s.UserID, &s.Name, &s.Description, &s.Query,
		pq.Array(&s.Source), pq.Array(&s.Tickers), &s.SimilarityThreshold, &s.IsActive,
		&s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}
