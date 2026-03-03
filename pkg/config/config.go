// The goal of this package is to standardize the way we configure our microservices.
package config

import (
	"net/http"
	"os"
	"time"
)

type NPMConfig struct {
	RegistryURL   string
	ReplicateURL  string
	HTTPTimeout   time.Duration
	HTTPTransport http.RoundTripper
}

// This includes configuration required by all microservices. For example, RabbitMQ connection details are required to communicate between the services
type CoreSvcConfig struct {
	ExternalPort string
	InternalPort string
}

type CoreConfig struct {
	RabbitMQURL string
	DatabaseURL string
	ValkeyURL   string

	CoreSvc CoreSvcConfig
	NPM     NPMConfig
}

type modifier func(*CoreConfig)

func NewCoreConfig(modifiers ...modifier) *CoreConfig {
	// Set default values
	cfg := &CoreConfig{
		RabbitMQURL: "amqp://admin:admin@rabbitmq:5672/",
		DatabaseURL: "postgres://postgres:postgres@core_db:5432/core?sslmode=disable",
		ValkeyURL:   "http://valkey:6379",
		CoreSvc: CoreSvcConfig{
			ExternalPort: "8080",
			InternalPort: "8081",
		},
	}

	// Apply modifiers
	for _, modify := range modifiers {
		modify(cfg)
	}

	return cfg
}

func WithEnv() modifier {
	return func(cfg *CoreConfig) {
		cfg.RabbitMQURL = getEnv("RABBITMQ_URL", cfg.RabbitMQURL)
		cfg.DatabaseURL = getEnv("DATABASE_URL", cfg.DatabaseURL)
		cfg.ValkeyURL = getEnv("VALKEY_URL", cfg.ValkeyURL)
		cfg.CoreSvc.ExternalPort = getEnv("EXTERNAL_PORT", cfg.CoreSvc.ExternalPort)
		cfg.CoreSvc.InternalPort = getEnv("INTERNAL_PORT", cfg.CoreSvc.InternalPort)
		cfg.NPM.RegistryURL = getEnv("NPM_REGISTRY_URL", "https://registry.npmjs.org")
		cfg.NPM.ReplicateURL = getEnv("NPM_REPLICATE_URL", "https://replicate.npmjs.com")
		if timeoutStr := getEnv("NPM_HTTP_TIMEOUT", "10s"); timeoutStr != "" {
			if timeout, err := time.ParseDuration(timeoutStr); err == nil {
				cfg.NPM.HTTPTimeout = timeout
			} else {
				cfg.NPM.HTTPTimeout = 10 * time.Second
			}
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}
