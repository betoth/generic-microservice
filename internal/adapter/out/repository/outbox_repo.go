package repository

import (
	"context"

	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/uptrace/bun"
)

type outboxRepo struct {
	db bun.IDB
}

func NewOutboxRepository(db *bun.DB) *outboxRepo {
	return &outboxRepo{db: db}
}

func (r *outboxRepo) Insert(ctx context.Context, outbox *domain.Outbox) error {
	_, err := txFromContext(ctx, r.db).NewInsert().
		Model(toOutboxModel(outbox)).
		Exec(ctx)
	return err
}
