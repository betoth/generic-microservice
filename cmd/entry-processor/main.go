package main

import (
	"context"
	"database/sql"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	kafkaconsumer "github.com/betoth/generic-microservice/internal/adapter/in/kafka"
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
	cfg, err := config.LoadPublisher()
	if err != nil {
		panic(fmt.Sprintf("config: %v", err))
	}

	dbCfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("db config: %v", err))
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dbCfg.DBConnString)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		panic(fmt.Sprintf("db ping: %v", err))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		panic(fmt.Sprintf("aws config: %v", err))
	}

	sqsOpts := []func(*awssqs.Options){}
	if cfg.S3Endpoint != "" {
		sqsOpts = append(sqsOpts, func(o *awssqs.Options) {
			o.BaseEndpoint = &cfg.S3Endpoint
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
	publisher := sqsadapter.NewPublisher(sqsClient, cfg.SQSInputQueueURL)
	svc := service.NewPublisherService(txManager, entryRepo, outboxRepo, publisher, s3Client, cfg.S3Bucket)

	consumer := kafkaconsumer.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	defer consumer.Close()

	fmt.Printf("  \033[32m✓\033[0m  Entry Processor  topic: %s\n", cfg.KafkaTopic)

	if err := consumer.Run(context.Background(), func(_, value string) error {
		return svc.Handle(context.Background(), value)
	}); err != nil {
		panic(fmt.Sprintf("consumer error: %v", err))
	}
}
