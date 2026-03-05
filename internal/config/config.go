package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	AppPort        string
	DBDriver       string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DatabaseURL    string
	PlunkAPIKey    string
	PlunkFromEmail string
	PlunkFromName  string
	JWTSecret      string
	CorsOrigins    string
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		AppPort:        getEnv("APP_PORT", "8080"),
		DBDriver:       getEnv("DB_DRIVER", "postgres"),
		DBHost:         getEnv("DB_HOST", ""),
		DBPort:         getEnv("DB_PORT", ""),
		DBUser:         getEnv("DB_USER", ""),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", ""),
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		PlunkAPIKey:    getEnv("PLUNK_API_KEY", ""),
		PlunkFromEmail: getEnv("PLUNK_FROM_EMAIL", ""),
		PlunkFromName:  getEnv("PLUNK_FROM_NAME", ""),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		CorsOrigins:    getEnv("CORS_ALLOW_ORIGINS", "*"),
	}

	// Validate required variables
	if config.DatabaseURL == "" {
		if config.DBDriver == "postgres" {
			if config.DBHost == "" {
				return nil, fmt.Errorf("DB_HOST is required")
			}
			if config.DBPort == "" {
				return nil, fmt.Errorf("DB_PORT is required")
			}
			if config.DBUser == "" {
				return nil, fmt.Errorf("DB_USER is required")
			}
			if config.DBPassword == "" {
				return nil, fmt.Errorf("DB_PASSWORD is required")
			}
			if config.DBName == "" {
				return nil, fmt.Errorf("DB_NAME is required")
			}
		} else if config.DBDriver == "sqlite" {
			if config.DBName == "" {
				return nil, fmt.Errorf("DB_NAME is required for sqlite")
			}
		}
	}

	if config.PlunkAPIKey == "" {
		return nil, fmt.Errorf("PLUNK_API_KEY is required")
	}
	if config.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return config, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
