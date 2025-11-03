package api

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"log/slog"
)

// RequestLoggingMiddleware emits structured JSON logs for every HTTP request.
func RequestLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		logger.Log(c.Request.Context(), level, "request completed",
			"client_ip", c.ClientIP(),
			"method", c.Request.Method,
			"path", path,
			"status_code", status,
			"latency_ms", float64(latency)/float64(time.Millisecond),
			"user_agent", c.Request.UserAgent(),
		)
	}
}

// AuthMiddleware enforces API key authentication using a constant time comparison.
func AuthMiddleware(expectedKey string, logger *slog.Logger) gin.HandlerFunc {
	expected := []byte(expectedKey)
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			unauthorized(c)
			logger.Warn("missing authorization header", "client_ip", c.ClientIP())
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			unauthorized(c)
			logger.Warn("unsupported authorization header", "client_ip", c.ClientIP())
			return
		}

		providedToken := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		provided := []byte(providedToken)
		if len(provided) != len(expected) || subtle.ConstantTimeCompare(provided, expected) != 1 {
			unauthorized(c)
			logger.Warn("invalid api key", "client_ip", c.ClientIP())
			return
		}

		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}

// RateLimitMiddleware enforces a per-IP rate limit backed by Redis.
func RateLimitMiddleware(client *redis.Client, limit int64, window time.Duration, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		key := fmt.Sprintf("ratelimit:%s", c.ClientIP())
		pipe := client.TxPipeline()
		counter := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, window)
		if _, err := pipe.Exec(ctx); err != nil {
			logger.Error("rate limiter redis error", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if counter.Val() > limit {
			logger.Warn("rate limit exceeded", "client_ip", c.ClientIP(), "count", counter.Val())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds standard security headers to each response.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'")
		c.Next()
	}
}
