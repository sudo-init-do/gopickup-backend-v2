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
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Printf("Failed to parse Redis URL: %v. Using default localhost.", err)
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
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
