package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv              string
	AppURL              string
	AppPort             string
	DBDriver            string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	DatabaseURL         string
	PlunkAPIKey         string
	PlunkFromEmail      string
	PlunkFromName       string
	JWTSecret           string
	CorsOrigins         string
	MetricsEnabled      bool
	RedisURL            string
	DefaultProductImage string
}

func LoadConfig() (*Config, error) {

	//  Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		AppURL:              getEnv("APP_URL", "http://localhost:8080"),
		AppPort:             getEnv("APP_PORT", "8080"),
		DBDriver:            getEnv("DB_DRIVER", "postgres"),
		DBHost:              getEnv("DB_HOST", ""),
		DBPort:              getEnv("DB_PORT", ""),
		DBUser:              getEnv("DB_USER", ""),
		DBPassword:          getEnv("DB_PASSWORD", ""),
		DBName:              getEnv("DB_NAME", ""),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		PlunkAPIKey:         getEnv("PLUNK_API_KEY", ""),
		PlunkFromEmail:      getEnv("PLUNK_FROM_EMAIL", ""),
		PlunkFromName:       getEnv("PLUNK_FROM_NAME", ""),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		CorsOrigins:         getEnv("CORS_ALLOW_ORIGINS", "*"),
		MetricsEnabled:      getEnv("ENABLE_METRICS", "false") == "true",
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379/0"),
		DefaultProductImage: getEnv("DEFAULT_PRODUCT_IMAGE", "https://images.unsplash.com/photo-1586528116311-ad8dd3c8310d?w=500&q=80"), // Logistics-themed placeholder
	}

	// VERBOSE DIAGNOSTICS
	log.Printf("[CONFIG_DEBUG] AppEnv: %s", config.AppEnv)
	log.Printf("[CONFIG_DEBUG] DBDriver: %s", config.DBDriver)
	log.Printf("[CONFIG_DEBUG] DBHost: %s", config.DBHost)
	if config.DatabaseURL != "" {
		log.Printf("[CONFIG_DEBUG] DATABASE_URL is DETECTED (length: %d)", len(config.DatabaseURL))
	} else {
		log.Printf("[CONFIG_DEBUG] DATABASE_URL is MISSING or EMPTY")
	}

	// Validate required variables (RELAXED - Warn only to allow startup)
	if config.DatabaseURL == "" && config.DBDriver == "postgres" && config.DBHost == "" {
		log.Printf("CRITICAL WARNING: No DatabaseURL or DB_HOST found. Connection will likely fail.")
	}

	if config.PlunkAPIKey == "" {
		log.Printf("CRITICAL CONFIG WARNING: PLUNK_API_KEY is missing! Email features will not work.")
	}
	if config.JWTSecret == "" {
		log.Printf("CRITICAL CONFIG ERROR: JWT_SECRET is missing! Authentication will fail.")
	}

	return config, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
