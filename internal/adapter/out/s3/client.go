package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/betoth/generic-microservice/internal/domain"
	"github.com/google/uuid"
)

type s3API interface {
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
}

type Client struct {
	api    s3API
	bucket string
}

func NewClient(api *awss3.Client, bucket string) *Client {
	return &Client{api: api, bucket: bucket}
}

// WriteSnapshot writes the entry to entries/<snapshotID>.json.
func (c *Client) WriteSnapshot(ctx context.Context, entry domain.Entry, snapshotID uuid.UUID) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = c.api.PutObject(ctx, &awss3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(snapshotKey(snapshotID)),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	return err
}

// ReadSnapshot reads the entry from entries/<snapshotID>.json.
func (c *Client) ReadSnapshot(ctx context.Context, snapshotID uuid.UUID) (domain.Entry, error) {
	out, err := c.api.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(snapshotKey(snapshotID)),
	})
	if err != nil {
		return domain.Entry{}, err
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return domain.Entry{}, err
	}

	var entry domain.Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return domain.Entry{}, err
	}
	return entry, nil
}

// SnapshotPath returns the full S3 URI for a snapshot.
func SnapshotPath(bucket string, snapshotID uuid.UUID) string {
	return fmt.Sprintf("s3://%s/%s", bucket, snapshotKey(snapshotID))
}

func snapshotKey(id uuid.UUID) string {
	return fmt.Sprintf("entries/%s.json", id)
}
