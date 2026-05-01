package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TokenRepository — интерфейс для работы с API токенами
type TokenRepository interface {
	// GetUserIDByToken получает user_id по токену (проверяет активность и срок действия)
	GetUserIDByToken(ctx context.Context, token string) (string, error)
}

// tokenRepository — реализация (приватная)
type tokenRepository struct {
	db *DB
}

// NewTokenRepository — конструктор
func NewTokenRepository(db *DB) TokenRepository {
	return &tokenRepository{db: db}
}

// GetUserIDByToken — проверяет токен и возвращает user_id
func (r *tokenRepository) GetUserIDByToken(ctx context.Context, token string) (string, error) {
	var userID string

	query := `
        SELECT user_id
        FROM api_tokens
        WHERE token = $1 
            AND is_active = TRUE 
            AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
    `

	err := r.db.pool.QueryRow(ctx, query, token).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("token %s: %w", token, ErrNotFound)
		}
		return "", fmt.Errorf("failed to validate token: %w", err)
	}

	return userID, nil
}
