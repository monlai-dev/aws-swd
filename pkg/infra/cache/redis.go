package cache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"log"
	"os"
)

func NewRedisClient(lc fx.Lifecycle) *redis.Client {

	url := os.Getenv("REDIS_URL")
	if url == "" {
		log.Fatal("REDIS_URL is not set")
	}

	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalf("Failed to parse REDIS_URL: %v", err)
	}

	client := redis.NewClient(opt)

	// Hook lifecycle
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("🔌 Connecting to Redis...")
			if err := client.Ping(ctx).Err(); err != nil {
				return err
			}
			log.Println("✅ Redis connected")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Println("🛑 Closing Redis...")
			return client.Close()
		},
	})

	return client

}
