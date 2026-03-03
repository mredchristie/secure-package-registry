package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"git.duti.dev/secure-package-registry/cmd/core-svc/server"
	"git.duti.dev/secure-package-registry/internal/gen/coredb"
	"git.duti.dev/secure-package-registry/pkg/config"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/pkgdb"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/golang-migrate/migrate/v4"
	pgxdriver "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	log = logger.WithComponent("core-svc")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info().Msg("Received shutdown signal")
		cancel()
	}()

	cfg := config.NewCoreConfig(config.WithEnv())

	log.Info().
		Str("database_url", cfg.DatabaseURL).
		Msg("Connecting to database")

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to create connection pool")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to ping database")
	}

	log.Info().Msg("Connected to database successfully")

	if err := runMigrations(ctx, cfg.DatabaseURL); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to run migrations")
	}

	log.Info().Msg("Migrations completed successfully")

	db := pkgdb.NewClient(pool)
	queries := coredb.New(pool)

	publisher, err := amqp.NewPublisher(
		amqp.NewDurableQueueConfig(cfg.RabbitMQURL),
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize RabbitMQ publisher")
	}
	defer func() {
		if cerr := publisher.Close(); cerr != nil {
			log.Warn().Err(cerr).Msg("Failed to close RabbitMQ publisher")
		}
	}()

	externalServer := server.NewExternal("0.0.0.0:"+cfg.CoreSvc.ExternalPort, db)
	internalServer := server.NewInternal("0.0.0.0:"+cfg.CoreSvc.InternalPort, queries, publisher)

	externalErr := externalServer.Start()
	internalErr := internalServer.Start()

	select {
	case <-ctx.Done():
		log.Info().Msg("Shutting down servers...")
	case err := <-externalErr:
		log.Fatal().Err(err).Msg("External server error")
	case err := <-internalErr:
		log.Fatal().Err(err).Msg("Internal server error")
	}

	shutdownCtx := context.Background()
	if err := externalServer.Stop(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to stop external server")
	}
	if err := internalServer.Stop(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to stop internal server")
	}

	log.Info().Msg("Shutdown complete")
}

func runMigrations(_ context.Context, databaseURL string) (err error) {
	log.Info().Msg("Running database migrations")

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer func() {
		if dberr := db.Close(); dberr != nil {
			log.Error().
				Err(dberr).
				Msg("Failed to close database connection after migrations")
			if err == nil {
				err = fmt.Errorf("failed to close database connection after migrations: %w", dberr)
			} else {
				err = fmt.Errorf("%v; additionally failed to close database connection after migrations: %w", err, dberr)
			}
		}
	}()

	driver, err := pgxdriver.WithInstance(db, &pgxdriver.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://infra/migrations",
		"pgx",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
