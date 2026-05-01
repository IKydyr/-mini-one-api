package repository

import (
	"context"
)

// ChargeRecord — DTO для записи о списании
type ChargeRecord struct {
	UserID       string
	Model        string
	Provider     string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// ChargeRepository — интерфейс
type ChargeRepository interface {
	RecordCharge(ctx context.Context, record ChargeRecord) error
}

type chargeRepository struct {
	db *DB
}

func NewChargeRepository(db *DB) ChargeRepository {
	return &chargeRepository{db: db}
}

func (r *chargeRepository) RecordCharge(ctx context.Context, record ChargeRecord) error {
	_, err := r.db.pool.Exec(ctx,
		`INSERT INTO user_charges (user_id, model, provider, input_tokens, output_tokens, total_tokens, cost_usd)
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		record.UserID, record.Model, record.Provider,
		record.InputTokens, record.OutputTokens,
		record.InputTokens+record.OutputTokens, record.CostUSD)
	return err
}
