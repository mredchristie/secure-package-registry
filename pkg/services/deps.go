// Package services provides shared infrastructure dependencies for all services.
package services

import (
	"context"
	"database/sql"
	"fmt"

	"git.duti.dev/secure-package-registry/internal/valkey"
	"git.duti.dev/secure-package-registry/pkg/config"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	pgxdriver "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	log = logger.WithComponent("services")
}

// Deps holds shared infrastructure clients used across services.
// Each service uses only the fields it needs. AMQP subscriber/publisher
// connections are created per-service from Config.RabbitMQURL to avoid
// channel contention and simplify shutdown ordering.
type Deps struct {
	Config *config.CoreConfig
	Pool   *pgxpool.Pool
	Valkey *valkey.Client
}

// NewDeps creates shared infrastructure clients from the given config.
// The caller must call Close when done.
func NewDeps(ctx context.Context, cfg *config.CoreConfig) (*Deps, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("creating postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}

	log.Info().Msg("Connected to database")

	vk, err := valkey.NewClient(cfg.ValkeyURL)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("creating valkey client: %w", err)
	}

	log.Info().Msg("Connected to Valkey")

	return &Deps{
		Config: cfg,
		Pool:   pool,
		Valkey: vk,
	}, nil
}

// Close tears down all shared clients in reverse order.
func (d *Deps) Close() {
	if d.Valkey != nil {
		d.Valkey.Close()
	}
	if d.Pool != nil {
		d.Pool.Close()
	}
}

// RunMigrations applies database migrations from infra/migrations/.
func (d *Deps) RunMigrations() (err error) {
	log.Info().Msg("Running database migrations")

	db, err := sql.Open("pgx", d.Config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("opening database for migrations: %w", err)
	}
	defer func() {
		if dberr := db.Close(); dberr != nil {
			log.Error().Err(dberr).Msg("Failed to close migration database connection")
			if err == nil {
				err = fmt.Errorf("closing migration connection: %w", dberr)
			}
		}
	}()

	driver, err := pgxdriver.WithInstance(db, &pgxdriver.Config{})
	if err != nil {
		return fmt.Errorf("creating migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:///app/infra/migrations",
		"pgx",
		driver,
	)
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("applying migrations: %w", err)
	}

	log.Info().Msg("Migrations completed")
	return nil
}
