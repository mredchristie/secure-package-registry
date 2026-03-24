package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	regproxy "git.duti.dev/secure-package-registry/pkg/services/reg-proxy"

	"git.duti.dev/secure-package-registry/pkg/config"
	"git.duti.dev/secure-package-registry/pkg/logger"
	"git.duti.dev/secure-package-registry/pkg/services"
)

func main() {
	log := logger.WithComponent("reg-proxy")

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

	if err := regproxy.Start(ctx, deps); err != nil {
		log.Fatal().Err(err).Msg("reg-proxy exited with error")
	}
}
