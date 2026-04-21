package port

import (
	"context"

	"github.com/betoth/generic-microservice/internal/domain"
)

type OutboxRepository interface {
	Insert(ctx context.Context, outbox *domain.Outbox) error
}
