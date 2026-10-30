package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient connects to Redis at redisURL and verifies the connection.
func NewRedisClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS.URL is required")
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS.URL: %w", err)
	}
	client := redis.NewClient(options)

	pingContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingContext).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to redis: %w", err)
	}
	log.Println("redis connected successfully")

	return client, nil
}
