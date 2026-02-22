package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Provider     string
	EncryptedKey string
	CreatedAt    time.Time
}

func (db *DB) AddAPIKey(ctx context.Context, userID uuid.UUID, provider, encryptedKey string) error {
	query := `
		INSERT INTO api_keys (user_id, provider, encrypted_key)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, provider) 
		DO UPDATE SET encrypted_key = EXCLUDED.encrypted_key, created_at = NOW()
	`
	_, err := db.ExecContext(ctx, query, userID, provider, encryptedKey)
	return err
}

func (db *DB) GetAPIKeys(ctx context.Context, userID uuid.UUID) ([]APIKey, error) {
	query := `
		SELECT id, user_id, provider, encrypted_key, created_at
		FROM api_keys
		WHERE user_id = $1
	`
	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		if err := rows.Scan(&key.ID, &key.UserID, &key.Provider, &key.EncryptedKey, &key.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (db *DB) GetAPIKey(ctx context.Context, userID uuid.UUID, provider string) (*APIKey, error) {
	query := `
		SELECT id, user_id, provider, encrypted_key, created_at
		FROM api_keys
		WHERE user_id = $1 AND provider = $2
	`
	var key APIKey
	err := db.QueryRowContext(ctx, query, userID, provider).Scan(
		&key.ID, &key.UserID, &key.Provider, &key.EncryptedKey, &key.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &key, nil
}

func (db *DB) DeleteAPIKey(ctx context.Context, userID uuid.UUID, provider string) error {
	query := `DELETE FROM api_keys WHERE user_id = $1 AND provider = $2`
	_, err := db.ExecContext(ctx, query, userID, provider)
	return err
}
