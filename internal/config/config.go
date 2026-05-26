package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	AppURL                 string
	AppPort                string
	DBDriver               string
	DBHost                 string
	DBPort                 string
	DBUser                 string
	DBPassword             string
	DBName                 string
	DatabaseURL            string
	PlunkAPIKey            string
	PlunkFromEmail         string
	PlunkFromName          string
	JWTSecret              string
	CorsOrigins            string
	MetricsEnabled         bool
	RedisURL               string
	DefaultProductImage    string
	WhatsAppSupportNumber  string // e.g. 2348012345678 (no + prefix)
}

func LoadConfig() (*Config, error) {

	//  Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		AppEnv:                getEnv("APP_ENV", "development"),
		AppURL:                getEnv("APP_URL", "http://localhost:8080"),
		AppPort:               getEnv("APP_PORT", "8080"),
		DBDriver:              getEnv("DB_DRIVER", "postgres"),
		DBHost:                getEnv("DB_HOST", ""),
		DBPort:                getEnv("DB_PORT", ""),
		DBUser:                getEnv("DB_USER", ""),
		DBPassword:            getEnv("DB_PASSWORD", ""),
		DBName:                getEnv("DB_NAME", ""),
		DatabaseURL:           getEnv("DATABASE_URL", ""),
		PlunkAPIKey:           getEnv("PLUNK_API_KEY", ""),
		PlunkFromEmail:        getEnv("PLUNK_FROM_EMAIL", ""),
		PlunkFromName:         getEnv("PLUNK_FROM_NAME", ""),
		JWTSecret:             getEnv("JWT_SECRET", ""),
		CorsOrigins:           getEnv("CORS_ALLOW_ORIGINS", "*"),
		MetricsEnabled:        getEnv("ENABLE_METRICS", "false") == "true",
		RedisURL:              getEnv("REDIS_URL", "redis://localhost:6379/0"),
		DefaultProductImage:   getEnv("DEFAULT_PRODUCT_IMAGE", "https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?w=500&q=80"),
		WhatsAppSupportNumber: getEnv("WHATSAPP_SUPPORT_NUMBER", "2348087042206"),
	}

	// Production logging
	log.Printf("[CONFIG] Starting app in %s mode", config.AppEnv)

	// Validate required variables
	if config.DatabaseURL != "" {
		log.Printf("[CONFIG] Database connection: DATABASE_URL detected")
	} else if config.DBDriver == "postgres" {
		log.Printf("[CONFIG] Using individual field connection mode (Host: %s)", config.DBHost)
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
	}

	if config.PlunkAPIKey == "" {
		log.Printf("Warning: PLUNK_API_KEY is missing. Email features disabled.")
	}
	if config.JWTSecret == "" {
		if config.AppEnv == "production" {
			// Empty secret means tokens are signed with "" — auth is fully
			// broken/insecure, so fail fast with a clear message.
			return nil, fmt.Errorf("JWT_SECRET is required in production")
		}
		log.Printf("Warning: JWT_SECRET is missing. Authentication will fail.")
	} else if len(config.JWTSecret) < 32 && config.AppEnv == "production" {
		// Warn (don't crash) so an existing deploy with a short secret keeps booting.
		log.Printf("WARNING: JWT_SECRET is shorter than 32 characters. Use a longer random secret for production.")
	}

	if config.AppEnv == "production" && (config.CorsOrigins == "" || config.CorsOrigins == "*") {
		// Don't crash; the CORS middleware disables wildcard in production and
		// falls back to the built-in trusted origin list.
		log.Printf("WARNING: CORS_ALLOW_ORIGINS is empty or '*' in production; wildcard is disabled and only built-in trusted origins will be allowed.")
	}

	return config, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
