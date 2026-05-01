package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UserRepository — интерфейс для работы с пользователями
type UserRepository interface {
	// GetByUserID получает пользователя по его user_id (логину)
	GetByUserID(ctx context.Context, userID string) (*UserDB, error)

	// GetBalance получает только баланс пользователя
	GetBalance(ctx context.Context, userID string) (float64, error)

	// GetTotalTokens получает общее количество потраченных токенов
	GetTotalTokens(ctx context.Context, userID string) (int64, error)

	// AddTokens добавляет токены к счётчику пользователя
	AddTokens(ctx context.Context, userID string, tokens int64) error

	DeductBalance(ctx context.Context, userID string, amount float64) (float64, error)
}

// userRepository — реализация UserRepository (приватная структура)
type userRepository struct {
	db *DB
}

// NewUserRepository — конструктор (возвращает интерфейс!)
func NewUserRepository(db *DB) UserRepository {
	return &userRepository{db: db}
}

// GetByUserID — получает пользователя по user_id
func (r *userRepository) GetByUserID(ctx context.Context, userID string) (*UserDB, error) {
	var user UserDB

	query := `
        SELECT id, user_id, balance_usd, total_tokens_used, created_at, updated_at, deleted_at
        FROM users
        WHERE user_id = $1 AND deleted_at IS NULL
    `

	err := r.db.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.UserID,
		&user.BalanceUSD,
		&user.TotalTokensUsed,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user %s: %w", userID, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get user %s: %w", userID, err)
	}

	return &user, nil
}

// GetBalance — получает только баланс пользователя
func (r *userRepository) GetBalance(ctx context.Context, userID string) (float64, error) {
	var balance float64

	query := `
        SELECT balance_usd
        FROM users
        WHERE user_id = $1 AND deleted_at IS NULL
    `

	err := r.db.pool.QueryRow(ctx, query, userID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("user %s: %w", userID, ErrNotFound)
		}
		return 0, fmt.Errorf("failed to get balance for user %s: %w", userID, err)
	}

	return balance, nil
}

// GetTotalTokens — получает общее количество потраченных токенов
func (r *userRepository) GetTotalTokens(ctx context.Context, userID string) (int64, error) {
	var tokens int64

	query := `
        SELECT total_tokens_used
        FROM users
        WHERE user_id = $1 AND deleted_at IS NULL
    `

	err := r.db.pool.QueryRow(ctx, query, userID).Scan(&tokens)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("user %s: %w", userID, ErrNotFound)
		}
		return 0, fmt.Errorf("failed to get total tokens for user %s: %w", userID, err)
	}

	return tokens, nil
}

// AddTokens — добавляет токены к счётчику пользователя
func (r *userRepository) AddTokens(ctx context.Context, userID string, tokens int64) error {
	query := `
        UPDATE users
        SET total_tokens_used = total_tokens_used + $1,
            updated_at = CURRENT_TIMESTAMP
        WHERE user_id = $2 AND deleted_at IS NULL
    `

	tag, err := r.db.pool.Exec(ctx, query, tokens, userID)
	if err != nil {
		return fmt.Errorf("failed to add tokens for user %s: %w", userID, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user %s: %w", userID, ErrNotFound)
	}

	return nil
}

func (r *userRepository) DeductBalance(ctx context.Context, userID string, amount float64) (float64, error) {
	var newBalance float64

	query := `
        UPDATE users 
        SET balance_usd = balance_usd - $1,
            updated_at = CURRENT_TIMESTAMP
        WHERE user_id = $2 
          AND balance_usd >= $1
          AND deleted_at IS NULL
        RETURNING balance_usd
    `

	err := r.db.pool.QueryRow(ctx, query, amount, userID).Scan(&newBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Получаем текущий баланс для информативной ошибки
			currentBalance, _ := r.GetBalance(ctx, userID)
			return 0, fmt.Errorf("insufficient balance: have %.4f, need %.4f", currentBalance, amount)
		}
		return 0, fmt.Errorf("failed to deduct balance for user %s: %w", userID, err)
	}

	return newBalance, nil
}

/*
Что важно в этом коде:

Элемент	Объяснение
UserRepository интерфейс	Определяет контракт. Service будет зависеть от интерфейса, а не от реализации
userRepository структурa	Приватная (с маленькой буквы) — скрывает детали реализации
NewUserRepository()	Конструктор, возвращает интерфейс, а не структуру
deleted_at IS NULL	Мягкое удаление — записи не удаляются физически
pgx.ErrNoRows	Обработка ситуации "пользователь не найден"
RowsAffected()	Проверяем, обновилась ли запись
*/
