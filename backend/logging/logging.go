package logging

import (
	"log/slog"
	"os"
	"sync"
)

var (
	once   sync.Once
	logger *slog.Logger
)

// Configure initializes the shared JSON logger. It is safe to call multiple times.
func Configure() *slog.Logger {
	once.Do(func() {
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
		logger = slog.New(handler)
	})
	return logger
}

// Logger returns the configured slog logger, configuring it on first use if necessary.
func Logger() *slog.Logger {
	if logger == nil {
		return Configure()
	}
	return logger
}
