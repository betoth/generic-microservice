package repository

import (
	"context"
	"time"

	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type entryRepo struct {
	db bun.IDB
}

func NewEntryRepository(db *bun.DB) *entryRepo {
	return &entryRepo{db: db}
}

func (r *entryRepo) Insert(ctx context.Context, entry *domain.Entry) error {
	_, err := txFromContext(ctx, r.db).NewInsert().
		Model(toEntryModel(entry)).
		Exec(ctx)
	return err
}

func (r *entryRepo) GetStatus(ctx context.Context, id uuid.UUID) (domain.EntryStatus, error) {
	var m entryModel
	err := txFromContext(ctx, r.db).NewSelect().
		Model(&m).
		Column("status").
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return "", err
	}
	return m.Status, nil
}

func (r *entryRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.EntryStatus) error {
	_, err := txFromContext(ctx, r.db).NewUpdate().
		Model((*entryModel)(nil)).
		Set("status = ?", status).
		Set("updated_at = ?", time.Now().UTC()).
		Where("id = ?", id).
		Exec(ctx)
	return err
}
