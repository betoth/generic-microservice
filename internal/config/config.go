package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBConnString string
	HTTPPort     string
	S3Bucket     string
	S3Endpoint   string
	AWSRegion    string
}

func Load() (*Config, error) {
	dbConn, ok := os.LookupEnv("DB_CONN_STRING")
	if !ok {
		return nil, fmt.Errorf("DB_CONN_STRING is required")
	}

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	s3Bucket, ok := os.LookupEnv("S3_BUCKET")
	if !ok {
		return nil, fmt.Errorf("S3_BUCKET is required")
	}

	return &Config{
		DBConnString: dbConn,
		HTTPPort:     port,
		S3Bucket:     s3Bucket,
		S3Endpoint:   os.Getenv("S3_ENDPOINT"),
		AWSRegion:    os.Getenv("AWS_REGION"),
	}, nil
}
