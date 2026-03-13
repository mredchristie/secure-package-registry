package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"git.duti.dev/secure-package-registry/pkg/config"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/services"
	packagewatcher "git.duti.dev/secure-package-registry/pkg/services/package-watcher"
)

func main() {
	log := logger.WithComponent("poller")

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

	if err := packagewatcher.Start(ctx, deps); err != nil {
		log.Fatal().Err(err).Msg("poller exited with error")
	}
}
