package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/google/uuid"
)

// --- stubs ---

type stubEntryRepoForPublisher struct {
	status    domain.EntryStatus
	statusErr error
	updateErr error
}

func (r *stubEntryRepoForPublisher) Insert(_ context.Context, _ *domain.Entry) error { return nil }
func (r *stubEntryRepoForPublisher) GetStatus(_ context.Context, _ uuid.UUID) (domain.EntryStatus, error) {
	return r.status, r.statusErr
}
func (r *stubEntryRepoForPublisher) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.EntryStatus) error {
	return r.updateErr
}

type stubOutboxRepoForPublisher struct {
	insertErr error
}

func (r *stubOutboxRepoForPublisher) Insert(_ context.Context, _ *domain.Outbox) error {
	return r.insertErr
}

type stubTxManagerForPublisher struct{}

func (m *stubTxManagerForPublisher) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type stubFileStorageForPublisher struct {
	entry    domain.Entry
	readErr  error
	writeErr error
}

func (s *stubFileStorageForPublisher) WriteSnapshot(_ context.Context, _ domain.Entry, _ uuid.UUID) error {
	return s.writeErr
}
func (s *stubFileStorageForPublisher) ReadSnapshot(_ context.Context, _ uuid.UUID) (domain.Entry, error) {
	return s.entry, s.readErr
}

// --- helpers ---

func stubEntry(id uuid.UUID) domain.Entry {
	now := time.Now().UTC()
	return domain.Entry{
		ID:        id,
		Subject:   "test subject",
		Status:    domain.EntryStatusProcessing,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func validMsg(id uuid.UUID) string {
	snapshotID := uuid.New()
	return `{"business_key":"` + id.String() + `","file":{"name":"` + snapshotID.String() + `.json","path":"s3://b/entries/` + snapshotID.String() + `.json"}}`
}

// --- tests ---

func TestEntryPublisherService_Handle_Success(t *testing.T) {
	id := uuid.New()
	svc := NewEntryPublisherService(
		&stubTxManagerForPublisher{},
		&stubEntryRepoForPublisher{status: domain.EntryStatusProcessing},
		&stubOutboxRepoForPublisher{},
		&stubFileStorageForPublisher{entry: stubEntry(id)},
		"bucket",
	)

	if err := svc.Handle(context.Background(), validMsg(id)); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestEntryPublisherService_Handle_AlreadyPublished(t *testing.T) {
	id := uuid.New()
	svc := NewEntryPublisherService(
		&stubTxManagerForPublisher{},
		&stubEntryRepoForPublisher{status: domain.EntryStatusPublished},
		&stubOutboxRepoForPublisher{},
		&stubFileStorageForPublisher{entry: stubEntry(id)},
		"bucket",
	)

	err := svc.Handle(context.Background(), validMsg(id))
	if !errors.Is(err, domain.ErrEntryAlreadyPublished) {
		t.Fatalf("expected ErrEntryAlreadyPublished, got %v", err)
	}
}

func TestEntryPublisherService_Handle_GetStatusError(t *testing.T) {
	id := uuid.New()
	repoErr := errors.New("db error")
	svc := NewEntryPublisherService(
		&stubTxManagerForPublisher{},
		&stubEntryRepoForPublisher{statusErr: repoErr},
		&stubOutboxRepoForPublisher{},
		&stubFileStorageForPublisher{entry: stubEntry(id)},
		"bucket",
	)

	err := svc.Handle(context.Background(), validMsg(id))
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestEntryPublisherService_Handle_UpdateStatusError(t *testing.T) {
	id := uuid.New()
	updateErr := errors.New("update failed")
	svc := NewEntryPublisherService(
		&stubTxManagerForPublisher{},
		&stubEntryRepoForPublisher{status: domain.EntryStatusProcessing, updateErr: updateErr},
		&stubOutboxRepoForPublisher{},
		&stubFileStorageForPublisher{entry: stubEntry(id)},
		"bucket",
	)

	err := svc.Handle(context.Background(), validMsg(id))
	if !errors.Is(err, updateErr) {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestEntryPublisherService_Handle_InvalidJSON(t *testing.T) {
	svc := NewEntryPublisherService(
		&stubTxManagerForPublisher{},
		&stubEntryRepoForPublisher{},
		&stubOutboxRepoForPublisher{},
		&stubFileStorageForPublisher{},
		"bucket",
	)

	if err := svc.Handle(context.Background(), "not-json"); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestEntryPublisherService_Handle_S3ReadError(t *testing.T) {
	id := uuid.New()
	s3Err := errors.New("s3 unavailable")
	svc := NewEntryPublisherService(
		&stubTxManagerForPublisher{},
		&stubEntryRepoForPublisher{status: domain.EntryStatusProcessing},
		&stubOutboxRepoForPublisher{},
		&stubFileStorageForPublisher{readErr: s3Err},
		"bucket",
	)

	err := svc.Handle(context.Background(), validMsg(id))
	if !errors.Is(err, s3Err) {
		t.Fatalf("expected s3 error, got %v", err)
	}
}
