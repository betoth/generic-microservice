package repository

import (
	"context"

	"github.com/uptrace/bun"
)

type txKey struct{}

type txManager struct {
	db *bun.DB
}

func NewTransactionManager(db *bun.DB) *txManager {
	return &txManager{db: db}
}

func (m *txManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, bun.IDB(tx)))
	})
}

func txFromContext(ctx context.Context, fallback bun.IDB) bun.IDB {
	if tx, ok := ctx.Value(txKey{}).(bun.IDB); ok && tx != nil {
		return tx
	}
	return fallback
}
