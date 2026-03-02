package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Backup original env
	originalDBHost := os.Getenv("DB_HOST")
	defer os.Setenv("DB_HOST", originalDBHost)
	
	// Set required env vars for test
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	os.Setenv("DB_NAME", "gopickup")
	os.Setenv("PLUNK_API_KEY", "test")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, "test", cfg.PlunkAPIKey)
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	// Clear a required var
	os.Unsetenv("DB_HOST")
	
	// Ensure we don't pick up .env file content if possible, or override it
	// But LoadConfig loads .env first. 
	// To test failure we might need to be careful if .env exists.
	// However, os.Setenv overrides .env values usually if godotenv doesn't overwrite existing env vars.
	// godotenv.Load() does NOT overload existing env vars by default.
	// So if I Unsetenv, but .env has it, it might still be there if godotenv loaded it into process env.
	// Actually godotenv loads into os env.
	
	// Let's just test that it validates correctly if we can control the environment.
	// Since .env exists in the root, and we are running test in internal/config,
	// godotenv.Load() looks for .env in current directory? No, usually it looks in current.
	// If I run go test ./internal/config, the CWD is internal/config.
	// .env is in root. So godotenv.Load() might fail to find it, which is good for testing "missing vars".
	
	os.Clearenv() // Clear everything
	
	_, err := LoadConfig()
	assert.Error(t, err)
}
