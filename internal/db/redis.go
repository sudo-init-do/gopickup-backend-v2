package db

import (
	"context"
	"fmt"
	"log"

	"gopickup/internal/config"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis(cfg *config.Config) {
	// Debug log to see what we are trying to connect to
	// Masking password for safety in logs
	safeURL := cfg.RedisURL
	if len(safeURL) > 10 {
		log.Printf("Attempting to connect to Redis URL: %s...", safeURL[:10])
	} else {
		log.Printf("Attempting to connect to Redis URL (short): %s", safeURL)
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Printf("Failed to parse Redis URL: %v. Using default localhost.", err)
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	} else {
		log.Printf("Parsed Redis Options -> Addr: %s, DB: %d", opt.Addr, opt.DB)
	}

	RedisClient = redis.NewClient(opt)

	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		// Don't panic here, allow app to start without Redis (fallback to memory)
		// But for now, we'll set it to nil or keep it and let calls fail?
		// Better to keep it and let individual calls handle errors or fallback.
	} else {
		fmt.Println("Connected to Redis")
	}
}

func GetRedis() *redis.Client {
	return RedisClient
}
