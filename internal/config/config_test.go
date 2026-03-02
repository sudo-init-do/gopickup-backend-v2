package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "postgres")
	t.Setenv("DB_NAME", "gopickup")
	t.Setenv("PLUNK_API_KEY", "test")
	t.Setenv("JWT_SECRET", "test-secret")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "test", cfg.PlunkAPIKey)
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_NAME", "test.db")
	t.Setenv("PLUNK_API_KEY", "")
	t.Setenv("JWT_SECRET", "")

	_, err := LoadConfig()
	assert.Error(t, err)
}
