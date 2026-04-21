package port

import (
	"context"

	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/google/uuid"
)

type EntryRepository interface {
	Insert(ctx context.Context, entry *domain.Entry) error
	GetStatus(ctx context.Context, id uuid.UUID) (domain.EntryStatus, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.EntryStatus) error
}
