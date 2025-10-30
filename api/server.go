package api

import (
	"context"
	"fmt"
	"log"
	"os"

	"cortex/scanner"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Run initializes dependencies and starts the API server.
func Run() error {
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis at %s: %w", redisAddr, err)
	}

	store := NewRedisStore(redisClient)

	probes, stats, err := scanner.LoadProbes("nmap-service-probes")
	if err != nil {
		return fmt.Errorf("failed to load probes: %w", err)
	}
	if len(stats.ErrorLines) > 0 {
		log.Printf("probe loader reported %d warnings", len(stats.ErrorLines))
	}

	probeCache := scanner.NewProbeCache(probes)

	StartWorkers(store, probeCache, 5)

	router := gin.Default()
	server := NewServer(store)
	server.RegisterRoutes(router)

	log.Printf("starting Cortex API server on :8080")
	return router.Run(":8080")
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
