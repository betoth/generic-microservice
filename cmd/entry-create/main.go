package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	handler "github.com/betoth/generic-microservice/internal/adapter/in/http"
	"github.com/betoth/generic-microservice/internal/adapter/out/repository"
	s3adapter "github.com/betoth/generic-microservice/internal/adapter/out/s3"
	"github.com/betoth/generic-microservice/internal/config"
	"github.com/betoth/generic-microservice/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("config: %v", err))
	}

	runMigrations(cfg.DBConnString)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.DBConnString)))
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
	entrySvc := service.NewEntryService(txManager, entryRepo, outboxRepo, s3Client, cfg.S3Bucket)
	h := handler.New(entrySvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.GetHealth)
	r.Post("/entries", h.PostEntries)

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	fmt.Printf("  \033[32m✓\033[0m  API              port: %s\n", cfg.HTTPPort)
	if err := http.ListenAndServe(addr, r); err != nil {
		panic(err)
	}
}

func runMigrations(dsn string) {
	migrateDB, err := sql.Open("pg", dsn)
	if err != nil {
		panic(fmt.Sprintf("migrate open db: %v", err))
	}
	defer migrateDB.Close()

	driver, err := migratepostgres.WithInstance(migrateDB, &migratepostgres.Config{})
	if err != nil {
		panic(fmt.Sprintf("migrate driver: %v", err))
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		panic(fmt.Sprintf("migrate init: %v", err))
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		panic(fmt.Sprintf("migrate up: %v", err))
	}
}
