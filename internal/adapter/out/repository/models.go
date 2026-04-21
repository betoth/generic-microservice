package repository

import (
	"time"

	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type entryModel struct {
	bun.BaseModel `bun:"table:entries"`

	ID        uuid.UUID          `bun:"id,pk,type:uuid"`
	Date      time.Time          `bun:"date"`
	Subject   string             `bun:"subject"`
	Content   string             `bun:"content"`
	Status    domain.EntryStatus `bun:"status,type:entry_status"`
	CreatedAt time.Time          `bun:"created_at"`
	UpdatedAt time.Time          `bun:"updated_at"`
}

func toEntryModel(e *domain.Entry) *entryModel {
	return &entryModel{
		ID:        e.ID,
		Date:      e.Date,
		Subject:   e.Subject,
		Content:   e.Content,
		Status:    e.Status,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

type outboxModel struct {
	bun.BaseModel `bun:"table:outbox"`

	ID          uuid.UUID `bun:"id,pk,type:uuid"`
	BusinessKey uuid.UUID `bun:"business_key,type:uuid"`
	EventType   string    `bun:"event_type"`
	Topic       string    `bun:"topic"`
	EventData   string    `bun:"event_data"`
	CreatedAt   time.Time `bun:"created_at"`
}

func toOutboxModel(o *domain.Outbox) *outboxModel {
	return &outboxModel{
		ID:          o.ID,
		BusinessKey: o.BusinessKey,
		EventType:   o.EventType,
		Topic:       o.Topic,
		EventData:   o.EventData,
		CreatedAt:   o.CreatedAt,
	}
}
