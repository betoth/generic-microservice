package config

import (
	"fmt"
	"os"
	"strings"
)

type PublisherConfig struct {
	KafkaBrokers     []string
	KafkaTopic       string
	KafkaGroupID     string
	SQSInputQueueURL string
	AWSRegion        string
	S3Endpoint       string
	S3Bucket         string
}

func LoadPublisher() (*PublisherConfig, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return nil, fmt.Errorf("KAFKA_BROKERS is required")
	}

	queueURL := os.Getenv("SQS_INPUT_QUEUE_URL")
	if queueURL == "" {
		return nil, fmt.Errorf("SQS_INPUT_QUEUE_URL is required")
	}

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "generic-microservice"
	}

	groupID := os.Getenv("KAFKA_GROUP_ID")
	if groupID == "" {
		groupID = "entry-processor"
	}

	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is required")
	}

	return &PublisherConfig{
		KafkaBrokers:     strings.Split(brokers, ","),
		KafkaTopic:       topic,
		KafkaGroupID:     groupID,
		SQSInputQueueURL: queueURL,
		AWSRegion:        os.Getenv("AWS_REGION"),
		S3Endpoint:       os.Getenv("S3_ENDPOINT"),
		S3Bucket:         bucket,
	}, nil
}
