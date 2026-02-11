package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	pgxdriver "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
)

const (
	defaultDBHost     = "core_db"
	defaultDBPort     = "5432"
	defaultDBUser     = "core"
	defaultDBPassword = "PleaseChangeMe"
	defaultDBName     = "secure_registry"
)

var log zerolog.Logger

func init() {
	log = logger.WithComponent("core-svc")
}

func main() {
	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info().Msg("Received shutdown signal")
		cancel()
	}()

	// Get database connection info from environment
	dbHost := getEnv("DB_HOST", defaultDBHost)
	dbPort := getEnv("DB_PORT", defaultDBPort)
	dbUser := getEnv("DB_USER", defaultDBUser)
	dbPassword := getEnv("DB_PASSWORD", defaultDBPassword)
	dbName := getEnv("DB_NAME", defaultDBName)

	// Build connection string
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	log.Info().
		Str("host", dbHost).
		Str("port", dbPort).
		Str("database", dbName).
		Msg("Connecting to database")

	// Connect to database
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to create connection pool")
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to ping database")
	}

	log.Info().Msg("Connected to database successfully")

	// Run migrations
	if err := runMigrations(ctx, databaseURL); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to run migrations")
	}

	log.Info().Msg("Migrations completed successfully")

	// TODO: Initialize coredb client and start service
	// queries := coredb.New(pool)

	// Keep running until shutdown
	<-ctx.Done()
	log.Info().Msg("Shutting down...")
}

// runMigrations applies database migrations
func runMigrations(ctx context.Context, databaseURL string) (err error) {
	log.Info().Msg("Running database migrations")

	// Open database connection using database/sql for migrations
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

	// Create migrate instance with pgx driver
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

	// Apply migrations (forward only)
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
