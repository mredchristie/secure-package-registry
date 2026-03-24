// Command spr runs all services (core-svc, poller, be-runner, reg-proxy) in a
// single process using an errgroup. If any service returns an error, the
// remaining services are cancelled and the process exits with the first error.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	berunner "git.duti.dev/secure-package-registry/pkg/services/be-runner"
	coresvc "git.duti.dev/secure-package-registry/pkg/services/core-svc"
	packagewatcher "git.duti.dev/secure-package-registry/pkg/services/package-watcher"
	regproxy "git.duti.dev/secure-package-registry/pkg/services/reg-proxy"
	"git.duti.dev/secure-package-registry/pkg/services/seed"

	"git.duti.dev/secure-package-registry/pkg/config"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/services"
	"golang.org/x/sync/errgroup"
)

func main() {
	log := logger.WithComponent("spr")

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

	deps, err := services.NewDeps(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize dependencies")
	}
	defer deps.Close()

	if err := deps.RunMigrations(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	if cfg.MockData {
		log.Info().Msg("SPR_MOCK=true: seeding database with dev data")
		if err := seed.Run(ctx, deps); err != nil {
			log.Fatal().Err(err).Msg("Seeding failed")
		}
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return coresvc.Start(gCtx, deps)
	})

	g.Go(func() error {
		return packagewatcher.Start(gCtx, deps)
	})

	g.Go(func() error {
		return berunner.Start(gCtx, deps)
	})

	g.Go(func() error {
		return regproxy.Start(gCtx, deps)
	})

	if err := g.Wait(); err != nil {
		log.Fatal().Err(err).Msg("Service exited with error")
	}

	log.Info().Msg("All services stopped")
}
