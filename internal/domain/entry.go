package domain

import (
	"time"

	"github.com/google/uuid"
)

type EntryStatus string

const (
	EntryStatusCreated    EntryStatus = "created"
	EntryStatusProcessing EntryStatus = "processing"
	EntryStatusPublished  EntryStatus = "published"
)

type Entry struct {
	ID        uuid.UUID
	Date      time.Time
	Subject   string
	Content   string
	Status    EntryStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Outbox struct {
	ID          uuid.UUID
	BusinessKey uuid.UUID
	EventType   string
	Topic       string
	EventData   string
	CreatedAt   time.Time
}
