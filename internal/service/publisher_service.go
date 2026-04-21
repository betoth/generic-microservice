package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	s3adapter "github.com/betoth/generic-microservice/internal/adapter/out/s3"
	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/betoth/generic-microservice/internal/port"
	"github.com/google/uuid"
)

type PublisherService struct {
	txManager   port.TransactionManager
	entryRepo   port.EntryRepository
	outboxRepo  port.OutboxRepository
	publisher   port.EventPublisher
	fileStorage port.EntryFileStorage
	s3Bucket    string
}

func NewPublisherService(txManager port.TransactionManager, entryRepo port.EntryRepository, outboxRepo port.OutboxRepository, publisher port.EventPublisher, fileStorage port.EntryFileStorage, s3Bucket string) *PublisherService {
	return &PublisherService{
		txManager:   txManager,
		entryRepo:   entryRepo,
		outboxRepo:  outboxRepo,
		publisher:   publisher,
		fileStorage: fileStorage,
		s3Bucket:    s3Bucket,
	}
}

type eventData struct {
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
}

type sqsMessage struct {
	BusinessKey uuid.UUID `json:"business_key"`
	File        struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"file"`
}

// Handle processes a Kafka message value (event_data JSON).
// Returns nil for non-entry.created events (ignored).
// Returns domain.ErrEntryAlreadyProcessing if entry is not in created status.
func (s *PublisherService) Handle(ctx context.Context, value string) error {
	var event eventData
	if err := json.Unmarshal([]byte(value), &event); err != nil {
		return fmt.Errorf("invalid event_data: %w", err)
	}

	if event.EventType != "entry.created" {
		return nil
	}

	// Check current status (outside transaction — optimistic check)
	status, err := s.entryRepo.GetStatus(ctx, event.BusinessKey)
	if err != nil {
		return err
	}
	if status != domain.EntryStatusCreated {
		return domain.ErrEntryAlreadyProcessing
	}

	// Read the created snapshot using the snapshotID from the event
	createdSnapshotID, err := snapshotIDFromFileName(event.File.Name)
	if err != nil {
		return fmt.Errorf("invalid snapshot file name: %w", err)
	}

	entry, err := s.fileStorage.ReadSnapshot(ctx, createdSnapshotID)
	if err != nil {
		return fmt.Errorf("read snapshot from s3: %w", err)
	}

	now := time.Now().UTC()
	entry.Status = domain.EntryStatusProcessing
	entry.UpdatedAt = now

	// Generate a new snapshot ID for the processing snapshot
	processingSnapshotID := uuid.New()
	processingFileName := fmt.Sprintf("%s.json", processingSnapshotID)
	processingFilePath := s3adapter.SnapshotPath(s.s3Bucket, processingSnapshotID)

	// Build SQS message pointing to the new processing snapshot
	msg := sqsMessage{BusinessKey: event.BusinessKey}
	msg.File.Name = processingFileName
	msg.File.Path = processingFilePath

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Write processing snapshot to S3 before sending SQS — consumer must find the file
	if err := s.fileStorage.WriteSnapshot(ctx, entry, processingSnapshotID); err != nil {
		return fmt.Errorf("write snapshot to s3: %w", err)
	}

	if err := s.publisher.Publish(ctx, string(msgBytes)); err != nil {
		return err
	}

	// Build full event_data for outbox
	processingEvent := eventData{
		ID:          uuid.New(),
		BusinessKey: event.BusinessKey,
		EventType:   "entry.processing",
		Topic:       "generic-microservice",
	}
	processingEvent.EntryData.Subject = entry.Subject
	processingEvent.EntryData.Status = entry.Status
	processingEvent.EntryData.CreatedAt = entry.CreatedAt
	processingEvent.EntryData.UpdatedAt = entry.UpdatedAt
	processingEvent.File.Name = processingFileName
	processingEvent.File.Path = processingFilePath

	processingEventData, err := json.Marshal(processingEvent)
	if err != nil {
		return err
	}

	return s.txManager.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.entryRepo.UpdateStatus(ctx, event.BusinessKey, domain.EntryStatusProcessing); err != nil {
			return err
		}
		return s.outboxRepo.Insert(ctx, &domain.Outbox{
			ID:          uuid.New(),
			BusinessKey: event.BusinessKey,
			EventType:   "entry.processing",
			Topic:       "generic-microservice",
			EventData:   string(processingEventData),
			CreatedAt:   now,
		})
	})
}

// snapshotIDFromFileName parses the UUID from a snapshot file name like "abc123.json".
func snapshotIDFromFileName(name string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSuffix(name, ".json"))
}
