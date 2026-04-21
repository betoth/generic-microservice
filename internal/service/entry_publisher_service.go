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

type EntryPublisherService struct {
	txManager   port.TransactionManager
	entryRepo   port.EntryRepository
	outboxRepo  port.OutboxRepository
	fileStorage port.EntryFileStorage
	s3Bucket    string
}

func NewEntryPublisherService(txManager port.TransactionManager, entryRepo port.EntryRepository, outboxRepo port.OutboxRepository, fileStorage port.EntryFileStorage, s3Bucket string) *EntryPublisherService {
	return &EntryPublisherService{
		txManager:   txManager,
		entryRepo:   entryRepo,
		outboxRepo:  outboxRepo,
		fileStorage: fileStorage,
		s3Bucket:    s3Bucket,
	}
}

type sqsInputMessage struct {
	BusinessKey uuid.UUID `json:"business_key"`
	File        struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"file"`
}

// Handle processes an SQS message body, transitioning the entry from processing to published.
// Returns ErrEntryAlreadyPublished if the entry is already in published status.
func (s *EntryPublisherService) Handle(ctx context.Context, msg string) error {
	var input sqsInputMessage
	if err := json.Unmarshal([]byte(msg), &input); err != nil {
		return fmt.Errorf("invalid sqs message: %w", err)
	}

	status, err := s.entryRepo.GetStatus(ctx, input.BusinessKey)
	if err != nil {
		return err
	}
	if status == domain.EntryStatusPublished {
		return domain.ErrEntryAlreadyPublished
	}

	// Read the processing snapshot using the snapshotID from the SQS message
	processingSnapshotID, err := snapshotIDFromFileName(input.File.Name)
	if err != nil {
		return fmt.Errorf("invalid snapshot file name: %w", err)
	}

	entry, err := s.fileStorage.ReadSnapshot(ctx, processingSnapshotID)
	if err != nil {
		return fmt.Errorf("read snapshot from s3: %w", err)
	}

	now := time.Now().UTC()
	entry.Status = domain.EntryStatusPublished
	entry.UpdatedAt = now

	// Generate a new snapshot ID for the published snapshot
	publishedSnapshotID := uuid.New()
	publishedFileName := fmt.Sprintf("%s.json", publishedSnapshotID)
	publishedFilePath := s3adapter.SnapshotPath(s.s3Bucket, publishedSnapshotID)

	// Write published snapshot to S3
	if err := s.fileStorage.WriteSnapshot(ctx, entry, publishedSnapshotID); err != nil {
		return fmt.Errorf("write snapshot to s3: %w", err)
	}

	// Build full event_data for outbox
	publishedEvent := eventData{
		ID:          uuid.New(),
		BusinessKey: input.BusinessKey,
		EventType:   "entry.published",
		Topic:       "generic-microservice",
	}
	publishedEvent.EntryData.Subject = entry.Subject
	publishedEvent.EntryData.Status = entry.Status
	publishedEvent.EntryData.CreatedAt = entry.CreatedAt
	publishedEvent.EntryData.UpdatedAt = entry.UpdatedAt
	publishedEvent.File.Name = publishedFileName
	publishedEvent.File.Path = publishedFilePath

	publishedEventData, err := json.Marshal(publishedEvent)
	if err != nil {
		return err
	}

	return s.txManager.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.entryRepo.UpdateStatus(ctx, input.BusinessKey, domain.EntryStatusPublished); err != nil {
			return err
		}
		return s.outboxRepo.Insert(ctx, &domain.Outbox{
			ID:          uuid.New(),
			BusinessKey: input.BusinessKey,
			EventType:   "entry.published",
			Topic:       "generic-microservice",
			EventData:   string(publishedEventData),
			CreatedAt:   now,
		})
	})
}
