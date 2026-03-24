// The goal of this package is to standardize the way we configure our microservices.
package config

import (
	"net/http"
	"os"
	"time"
)

type ReverseProxyConfig struct {
	InternalURL string
	ExternalURL string
}

type NPMConfig struct {
	RegistryURL   string
	ReplicateURL  string
	HTTPTimeout   time.Duration
	HTTPTransport http.RoundTripper
}

type GitHubConfig struct {
	Token         string
	Owner         string
	Repo          string
	WorkflowFile  string
	HTTPTransport http.RoundTripper
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
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
	MockData    bool

	CoreSvc      CoreSvcConfig
	NPM          NPMConfig
	GitHub       GitHubConfig
	MinIO        MinIOConfig
	ReverseProxy ReverseProxyConfig
}

type modifier func(*CoreConfig)

func NewCoreConfig(modifiers ...modifier) *CoreConfig {
	// Set default values
	cfg := &CoreConfig{
		RabbitMQURL: "amqp://admin:admin@rabbitmq:5672/",
		DatabaseURL: "postgres://postgres:postgres@core_db:5432/core?sslmode=disable",
		ValkeyURL:   "valkey:6379",
		CoreSvc: CoreSvcConfig{
			ExternalPort: "8080",
			InternalPort: "8081",
		},
		MinIO: MinIOConfig{
			Endpoint:  "minio:9000",
			AccessKey: "minio",
			SecretKey: "minio_pass",
			Bucket:    "behavior",
		},
		ReverseProxy: ReverseProxyConfig{
			InternalURL: "http://gitea:3000",
			ExternalURL: "http://localhost:7002",
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
		cfg.GitHub.Token = getEnv("GITHUB_TOKEN", "")
		cfg.GitHub.Owner = getEnv("GITHUB_OWNER", "")
		cfg.GitHub.Repo = getEnv("GITHUB_REPO", "")
		cfg.GitHub.WorkflowFile = getEnv("GITHUB_WORKFLOW_FILE", "collect-behavior.yml")
		cfg.MinIO.Endpoint = getEnv("MINIO_ENDPOINT", cfg.MinIO.Endpoint)
		cfg.MinIO.AccessKey = getEnv("MINIO_ACCESS_KEY", cfg.MinIO.AccessKey)
		cfg.MinIO.SecretKey = getEnv("MINIO_SECRET_KEY", cfg.MinIO.SecretKey)
		cfg.MinIO.Bucket = getEnv("MINIO_BUCKET", cfg.MinIO.Bucket)
		if sslStr := getEnv("MINIO_USE_SSL", ""); sslStr != "" {
			cfg.MinIO.UseSSL = sslStr == "true" || sslStr == "1"
		}

		cfg.ReverseProxy.InternalURL = getEnv("REVERSE_PROXY_INTERNAL_URL", cfg.ReverseProxy.InternalURL)
		cfg.ReverseProxy.ExternalURL = getEnv("REVERSE_PROXY_EXTERNAL_URL", cfg.ReverseProxy.ExternalURL)

		if mockStr := getEnv("SPR_MOCK", ""); mockStr != "" {
			cfg.MockData = mockStr == "true" || mockStr == "1"
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}
