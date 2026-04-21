package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsconsumer "github.com/betoth/generic-microservice/internal/adapter/in/sqs"
	"github.com/betoth/generic-microservice/internal/adapter/out/repository"
	s3adapter "github.com/betoth/generic-microservice/internal/adapter/out/s3"
	sqsadapter "github.com/betoth/generic-microservice/internal/adapter/out/sqs"
	"github.com/betoth/generic-microservice/internal/config"
	"github.com/betoth/generic-microservice/internal/service"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func main() {
	cfg, err := config.LoadEntryPublisherConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DBConnString)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "db ping: %v\n", err)
		os.Exit(1)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aws config: %v\n", err)
		os.Exit(1)
	}

	sqsOpts := []func(*awssqs.Options){}
	if cfg.SQSEndpoint != "" {
		sqsOpts = append(sqsOpts, func(o *awssqs.Options) {
			o.BaseEndpoint = &cfg.SQSEndpoint
		})
	}
	sqsClient := awssqs.NewFromConfig(awsCfg, sqsOpts...)

	s3Opts := []func(*awss3.Options){}
	if cfg.S3Endpoint != "" {
		s3Opts = append(s3Opts, func(o *awss3.Options) {
			o.BaseEndpoint = &cfg.S3Endpoint
			o.UsePathStyle = true
		})
	}
	s3Client := s3adapter.NewClient(awss3.NewFromConfig(awsCfg, s3Opts...), cfg.S3Bucket)

	txManager := repository.NewTransactionManager(db)
	entryRepo := repository.NewEntryRepository(db)
	outboxRepo := repository.NewOutboxRepository(db)

	svc := service.NewEntryPublisherService(txManager, entryRepo, outboxRepo, s3Client, cfg.S3Bucket)
	dlqPublisher := sqsadapter.NewDLQPublisher(sqsClient, cfg.SQSDLQURL)
	consumer := sqsconsumer.NewConsumer(sqsClient, cfg.SQSInputQueueURL, dlqPublisher)

	queue := cfg.SQSInputQueueURL[strings.LastIndex(cfg.SQSInputQueueURL, "/")+1:]
	dlq := cfg.SQSDLQURL[strings.LastIndex(cfg.SQSDLQURL, "/")+1:]
	fmt.Printf("  \033[32m✓\033[0m  Entry Publisher  queue: %s  dlq: %s\n", queue, dlq)

	if err := consumer.Run(ctx, func(ctx context.Context, body string) error {
		return svc.Handle(ctx, body)
	}); err != nil {
		fmt.Fprintf(os.Stderr, "consumer error: %v\n", err)
		os.Exit(1)
	}
}
