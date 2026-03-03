package config_test

import (
	"testing"
	"time"

	"git.duti.dev/secure-package-registry/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestNewCoreConfig_Defaults(t *testing.T) {
	t.Parallel()
	cfg := config.NewCoreConfig()

	assert.Equal(t, "amqp://admin:admin@rabbitmq:5672/", cfg.RabbitMQURL)
	assert.Equal(t, "postgres://postgres:postgres@core_db:5432/core?sslmode=disable", cfg.DatabaseURL)
	assert.Equal(t, "http://valkey:6379", cfg.ValkeyURL)
	assert.Equal(t, "8080", cfg.CoreSvc.ExternalPort)
	assert.Equal(t, "8081", cfg.CoreSvc.InternalPort)
}

func TestWithEnv_OverridesDefaults(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "amqp://test:test@localhost:5672/")
	t.Setenv("DATABASE_URL", "postgres://test@localhost/testdb")
	t.Setenv("VALKEY_URL", "http://localhost:6379")
	t.Setenv("EXTERNAL_PORT", "9090")
	t.Setenv("INTERNAL_PORT", "9091")
	t.Setenv("NPM_REGISTRY_URL", "https://custom.registry")
	t.Setenv("NPM_REPLICATE_URL", "https://custom.replicate")
	t.Setenv("NPM_HTTP_TIMEOUT", "30s")

	cfg := config.NewCoreConfig(config.WithEnv())

	assert.Equal(t, "amqp://test:test@localhost:5672/", cfg.RabbitMQURL)
	assert.Equal(t, "postgres://test@localhost/testdb", cfg.DatabaseURL)
	assert.Equal(t, "http://localhost:6379", cfg.ValkeyURL)
	assert.Equal(t, "9090", cfg.CoreSvc.ExternalPort)
	assert.Equal(t, "9091", cfg.CoreSvc.InternalPort)
	assert.Equal(t, "https://custom.registry", cfg.NPM.RegistryURL)
	assert.Equal(t, "https://custom.replicate", cfg.NPM.ReplicateURL)
	assert.Equal(t, 30*time.Second, cfg.NPM.HTTPTimeout)
}

func TestWithEnv_InvalidTimeoutFallsBackToDefault(t *testing.T) {
	t.Setenv("NPM_HTTP_TIMEOUT", "not-a-duration")

	cfg := config.NewCoreConfig(config.WithEnv())

	assert.Equal(t, 10*time.Second, cfg.NPM.HTTPTimeout)
}

func TestWithEnv_PreservesDefaultsWhenEnvUnset(t *testing.T) {
	cfg := config.NewCoreConfig(config.WithEnv())

	assert.Equal(t, "amqp://admin:admin@rabbitmq:5672/", cfg.RabbitMQURL)
	assert.Equal(t, "8080", cfg.CoreSvc.ExternalPort)
	assert.Equal(t, "https://registry.npmjs.org", cfg.NPM.RegistryURL)
}
