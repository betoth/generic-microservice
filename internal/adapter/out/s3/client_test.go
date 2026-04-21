package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/google/uuid"
)

type fakeS3 struct {
	putInput *awss3.PutObjectInput
	getBody  string
	getErr   error
	putErr   error
}

func (f *fakeS3) PutObject(_ context.Context, params *awss3.PutObjectInput, _ ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	f.putInput = params
	return &awss3.PutObjectOutput{}, f.putErr
}

func (f *fakeS3) GetObject(_ context.Context, _ *awss3.GetObjectInput, _ ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &awss3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(f.getBody)),
	}, nil
}

func TestWriteSnapshot(t *testing.T) {
	entry := domain.Entry{
		ID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Subject: "test",
		Content: "content",
		Status:  domain.EntryStatusCreated,
	}
	snapshotID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	fake := &fakeS3{}
	c := &Client{api: fake, bucket: "my-bucket"}

	if err := c.WriteSnapshot(context.Background(), entry, snapshotID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *fake.putInput.Bucket != "my-bucket" {
		t.Errorf("bucket: got %q, want %q", *fake.putInput.Bucket, "my-bucket")
	}

	wantKey := "entries/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.json"
	if *fake.putInput.Key != wantKey {
		t.Errorf("key: got %q, want %q", *fake.putInput.Key, wantKey)
	}

	body, _ := io.ReadAll(fake.putInput.Body)
	var got domain.Entry
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got.ID != entry.ID || got.Subject != entry.Subject {
		t.Errorf("body mismatch: got %+v", got)
	}
}

func TestWriteSnapshot_PropagatesError(t *testing.T) {
	fake := &fakeS3{putErr: io.ErrUnexpectedEOF}
	c := &Client{api: fake, bucket: "b"}
	err := c.WriteSnapshot(context.Background(), domain.Entry{ID: uuid.New()}, uuid.New())
	if err != io.ErrUnexpectedEOF {
		t.Errorf("expected ErrUnexpectedEOF, got %v", err)
	}
}

func TestReadSnapshot(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	entry := domain.Entry{
		ID:        id,
		Subject:   "s",
		Content:   "c",
		Status:    domain.EntryStatusCreated,
		CreatedAt: time.Time{},
	}
	data, _ := json.Marshal(entry)

	fake := &fakeS3{getBody: string(data)}
	c := &Client{api: fake, bucket: "b"}

	got, err := c.ReadSnapshot(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != entry.ID || got.Subject != entry.Subject {
		t.Errorf("read mismatch: got %+v, want %+v", got, entry)
	}
}

func TestReadSnapshot_PropagatesError(t *testing.T) {
	fake := &fakeS3{getErr: io.ErrUnexpectedEOF}
	c := &Client{api: fake, bucket: "b"}
	_, err := c.ReadSnapshot(context.Background(), uuid.New())
	if err != io.ErrUnexpectedEOF {
		t.Errorf("expected ErrUnexpectedEOF, got %v", err)
	}
}

func TestSnapshotKey(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	got := snapshotKey(id)
	want := "entries/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.json"
	if got != want {
		t.Errorf("snapshotKey: got %q, want %q", got, want)
	}
}

func TestSnapshotPath(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	got := SnapshotPath("my-bucket", id)
	want := "s3://my-bucket/entries/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.json"
	if got != want {
		t.Errorf("SnapshotPath: got %q, want %q", got, want)
	}
}

var _ = bytes.NewReader // suppress unused import warning
