package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	s3adapter "github.com/betoth/generic-microservice/internal/adapter/out/s3"
	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/betoth/generic-microservice/internal/port"
	"github.com/google/uuid"
)

type EntryService struct {
	txManager   port.TransactionManager
	entryRepo   port.EntryRepository
	outboxRepo  port.OutboxRepository
	fileStorage port.EntryFileStorage
	s3Bucket    string
}

func NewEntryService(txManager port.TransactionManager, entryRepo port.EntryRepository, outboxRepo port.OutboxRepository, fileStorage port.EntryFileStorage, s3Bucket string) *EntryService {
	return &EntryService{
		txManager:   txManager,
		entryRepo:   entryRepo,
		outboxRepo:  outboxRepo,
		fileStorage: fileStorage,
		s3Bucket:    s3Bucket,
	}
}

type CreateEntryInput struct {
	Date    time.Time
	Subject string
	Content string
}

type CreateEntryOutput struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

func (s *EntryService) CreateEntry(ctx context.Context, input CreateEntryInput) (*CreateEntryOutput, error) {
	now := time.Now().UTC()

	if !input.Date.UTC().Truncate(24 * time.Hour).Equal(now.Truncate(24 * time.Hour)) {
		return nil, domain.ErrDateMustBeToday
	}

	entry := &domain.Entry{
		ID:        uuid.New(),
		Date:      input.Date,
		Subject:   input.Subject,
		Content:   input.Content,
		Status:    domain.EntryStatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}

	snapshotID := uuid.New()

	if err := s.fileStorage.WriteSnapshot(ctx, *entry, snapshotID); err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%s.json", snapshotID)
	filePath := s3adapter.SnapshotPath(s.s3Bucket, snapshotID)

	eventPayload := struct {
		ID          uuid.UUID `json:"id"`
		BusinessKey uuid.UUID `json:"business_key"`
		EventType   string    `json:"event_type"`
		Topic       string    `json:"topic"`
		EntryData   struct {
			Subject   string             `json:"subject"`
			Status    domain.EntryStatus `json:"status"`
			CreatedAt time.Time          `json:"created_at"`
			UpdatedAt time.Time          `json:"updated_at"`
		} `json:"entry_data"`
		File struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"file"`
	}{
		ID:          uuid.New(),
		BusinessKey: entry.ID,
		EventType:   "entry.created",
		Topic:       "generic-microservice",
	}
	eventPayload.EntryData.Subject = entry.Subject
	eventPayload.EntryData.Status = entry.Status
	eventPayload.EntryData.CreatedAt = entry.CreatedAt
	eventPayload.EntryData.UpdatedAt = entry.UpdatedAt
	eventPayload.File.Name = fileName
	eventPayload.File.Path = filePath

	eventData, err := json.Marshal(eventPayload)
	if err != nil {
		return nil, err
	}

	err = s.txManager.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.entryRepo.Insert(ctx, entry); err != nil {
			return err
		}
		return s.outboxRepo.Insert(ctx, &domain.Outbox{
			ID:          uuid.New(),
			BusinessKey: entry.ID,
			EventType:   "entry.created",
			Topic:       "generic-microservice",
			EventData:   string(eventData),
			CreatedAt:   now,
		})
	})
	if err != nil {
		return nil, err
	}

	return &CreateEntryOutput{
		ID:        entry.ID,
		CreatedAt: entry.CreatedAt,
	}, nil
}
