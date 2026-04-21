package config

import (
	"fmt"
	"os"
)

type EntryPublisherConfig struct {
	DBConnString     string
	SQSInputQueueURL string
	SQSDLQURL        string
	AWSRegion        string
	SQSEndpoint      string
	S3Bucket         string
	S3Endpoint       string
}

func LoadEntryPublisherConfig() (EntryPublisherConfig, error) {
	dbConn := os.Getenv("DB_CONN_STRING")
	if dbConn == "" {
		return EntryPublisherConfig{}, fmt.Errorf("DB_CONN_STRING is required")
	}

	inputQueueURL := os.Getenv("SQS_INPUT_QUEUE_URL")
	if inputQueueURL == "" {
		return EntryPublisherConfig{}, fmt.Errorf("SQS_INPUT_QUEUE_URL is required")
	}

	dlqURL := os.Getenv("SQS_DLQ_URL")
	if dlqURL == "" {
		return EntryPublisherConfig{}, fmt.Errorf("SQS_DLQ_URL is required")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		return EntryPublisherConfig{}, fmt.Errorf("S3_BUCKET is required")
	}

	return EntryPublisherConfig{
		DBConnString:     dbConn,
		SQSInputQueueURL: inputQueueURL,
		SQSDLQURL:        dlqURL,
		AWSRegion:        region,
		SQSEndpoint:      os.Getenv("SQS_ENDPOINT"),
		S3Bucket:         s3Bucket,
		S3Endpoint:       os.Getenv("S3_ENDPOINT"),
	}, nil
}
