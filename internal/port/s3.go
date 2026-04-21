package port

import (
	"context"

	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/google/uuid"
)

// EntryFileStorage stores and retrieves entry snapshots.
// Each snapshot has an independent UUID-based key: entries/<snapshotID>.json
type EntryFileStorage interface {
	WriteSnapshot(ctx context.Context, entry domain.Entry, snapshotID uuid.UUID) error
	ReadSnapshot(ctx context.Context, snapshotID uuid.UUID) (domain.Entry, error)
}
