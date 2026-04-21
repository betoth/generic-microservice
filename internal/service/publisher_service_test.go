package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/google/uuid"
)

// --- fakes ---

type fakeTxManager struct{}

func (f *fakeTxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fakeEntryRepo struct {
	status    domain.EntryStatus
	statusErr error
	updatedTo domain.EntryStatus
	updateErr error
}

func (f *fakeEntryRepo) Insert(_ context.Context, _ *domain.Entry) error { return nil }

func (f *fakeEntryRepo) GetStatus(_ context.Context, _ uuid.UUID) (domain.EntryStatus, error) {
	return f.status, f.statusErr
}

func (f *fakeEntryRepo) UpdateStatus(_ context.Context, _ uuid.UUID, status domain.EntryStatus) error {
	f.updatedTo = status
	return f.updateErr
}

type fakeOutboxRepo struct {
	inserted *domain.Outbox
}

func (f *fakeOutboxRepo) Insert(_ context.Context, o *domain.Outbox) error {
	f.inserted = o
	return nil
}

type fakePublisher struct {
	published string
	err       error
}

func (f *fakePublisher) Publish(_ context.Context, body string) error {
	if f.err != nil {
		return f.err
	}
	f.published = body
	return nil
}

type fakeFileStorage struct {
	entry    domain.Entry
	readErr  error
	writeErr error
}

func (f *fakeFileStorage) WriteSnapshot(_ context.Context, _ domain.Entry, _ uuid.UUID) error {
	return f.writeErr
}
func (f *fakeFileStorage) ReadSnapshot(_ context.Context, _ uuid.UUID) (domain.Entry, error) {
	return f.entry, f.readErr
}

// --- helpers ---

func makeEventData(t *testing.T, eventType string, businessKey uuid.UUID) string {
	t.Helper()
	payload := eventData{
		ID:          uuid.New(),
		BusinessKey: businessKey,
		EventType:   eventType,
		Topic:       "generic-microservice",
	}
	payload.EntryData.Subject = "test"
	payload.EntryData.Status = domain.EntryStatusCreated
	payload.EntryData.CreatedAt = time.Now()
	payload.EntryData.UpdatedAt = time.Now()
	snapshotID := uuid.New()
	payload.File.Name = snapshotID.String() + ".json"
	payload.File.Path = "s3://bucket/entries/" + snapshotID.String() + ".json"

	b, _ := json.Marshal(payload)
	return string(b)
}

func fakeEntry(id uuid.UUID) domain.Entry {
	now := time.Now().UTC()
	return domain.Entry{
		ID:        id,
		Subject:   "test",
		Status:    domain.EntryStatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// --- tests ---

func TestHandle_IgnoresNonCreatedEvents(t *testing.T) {
	svc := NewPublisherService(&fakeTxManager{}, &fakeEntryRepo{}, &fakeOutboxRepo{}, &fakePublisher{}, &fakeFileStorage{}, "bucket")
	id := uuid.New()

	for _, eventType := range []string{"entry.processing", "entry.published", "other"} {
		err := svc.Handle(context.Background(), makeEventData(t, eventType, id))
		if err != nil {
			t.Errorf("event_type %q: expected nil, got %v", eventType, err)
		}
	}
}

func TestHandle_ReturnsBusinessErrorWhenNotCreated(t *testing.T) {
	id := uuid.New()
	repo := &fakeEntryRepo{status: domain.EntryStatusProcessing}
	svc := NewPublisherService(&fakeTxManager{}, repo, &fakeOutboxRepo{}, &fakePublisher{}, &fakeFileStorage{entry: fakeEntry(id)}, "bucket")

	err := svc.Handle(context.Background(), makeEventData(t, "entry.created", id))

	var bizErr *domain.BusinessError
	if !errors.As(err, &bizErr) {
		t.Fatalf("expected BusinessError, got %v", err)
	}
	if bizErr.Code != "GMS-002" {
		t.Errorf("expected code GMS-002, got %s", bizErr.Code)
	}
}

func TestHandle_SuccessPath(t *testing.T) {
	id := uuid.New()
	repo := &fakeEntryRepo{status: domain.EntryStatusCreated}
	outbox := &fakeOutboxRepo{}
	pub := &fakePublisher{}
	fs := &fakeFileStorage{entry: fakeEntry(id)}
	svc := NewPublisherService(&fakeTxManager{}, repo, outbox, pub, fs, "bucket")

	err := svc.Handle(context.Background(), makeEventData(t, "entry.created", id))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pub.published == "" {
		t.Error("expected SQS message to be published")
	}

	var msg sqsMessage
	if err := json.Unmarshal([]byte(pub.published), &msg); err != nil {
		t.Fatalf("invalid SQS message: %v", err)
	}
	if msg.BusinessKey != id {
		t.Errorf("expected business_key %s, got %s", id, msg.BusinessKey)
	}

	if repo.updatedTo != domain.EntryStatusProcessing {
		t.Errorf("expected status processing, got %s", repo.updatedTo)
	}
	if outbox.inserted == nil || outbox.inserted.EventType != "entry.processing" {
		t.Error("expected outbox row with event_type entry.processing")
	}

	// Verify event_data has full format
	var ed eventData
	if err := json.Unmarshal([]byte(outbox.inserted.EventData), &ed); err != nil {
		t.Fatalf("invalid event_data: %v", err)
	}
	if ed.EntryData.Subject == "" {
		t.Error("expected entry_data.subject to be populated")
	}
	if ed.File.Name == "" {
		t.Error("expected file.name to be populated")
	}
}

func TestHandle_SQSFailureDoesNotUpdateDB(t *testing.T) {
	id := uuid.New()
	repo := &fakeEntryRepo{status: domain.EntryStatusCreated}
	pub := &fakePublisher{err: errors.New("sqs down")}
	svc := NewPublisherService(&fakeTxManager{}, repo, &fakeOutboxRepo{}, pub, &fakeFileStorage{entry: fakeEntry(id)}, "bucket")

	err := svc.Handle(context.Background(), makeEventData(t, "entry.created", id))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if repo.updatedTo != "" {
		t.Error("status should not have been updated when SQS fails")
	}
}
