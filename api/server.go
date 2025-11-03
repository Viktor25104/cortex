package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"cortex/logging"
	"cortex/scanner"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "cortex/docs"
)

// @title           Cortex API
// @version         5.0
// @description     REST API for the Cortex Network Scanner.
// @termsOfService  http://swagger.io/terms/
// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
// @host      localhost:8080
// @BasePath  /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// Run initializes dependencies and starts the API server.
func Run() error {
	logging.Configure()
	logger := logging.Logger()

	if err := godotenv.Load(); err != nil {
		logger.Warn("failed to load .env file", "error", err)
	}

	apiKey := os.Getenv("CORTEX_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("CORTEX_API_KEY environment variable is required")
	}

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
		logger.Warn("probe loader reported warnings", "count", len(stats.ErrorLines))
	}

	probeCache := scanner.NewProbeCache(probes)

	StartWorkers(store, probeCache, 5)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(SecurityHeadersMiddleware())
	router.Use(RequestLoggingMiddleware(logger))

	// Configure Swagger UI endpoint.
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	apiGroup := router.Group("/api/v1")
	apiGroup.Use(AuthMiddleware(apiKey, logger))
	apiGroup.Use(RateLimitMiddleware(redisClient, 100, time.Minute, logger))

	server := NewServer(store)
	server.RegisterRoutes(apiGroup)

	logger.Info("starting Cortex API server", "addr", ":8080")
	logger.Info("swagger documentation available", "url", "http://localhost:8080/docs/index.html")
	return router.Run("0.0.0.0:8080")
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
