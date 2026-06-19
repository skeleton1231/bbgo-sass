package main

import (
	"log"
	"log/slog"
	"os"
	"strings"
)

// SetupLogging configures slog with JSON or text output based on LOG_FORMAT env.
// LOG_LEVEL controls verbosity: debug, info, warn, error (default info).
func SetupLogging() *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.EqualFold(os.Getenv("LOG_FORMAT"), "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Bridge the legacy log package so existing log.Printf calls emit structured records.
	log.SetFlags(0)
	log.SetOutput(slog.NewLogLogger(handler, level).Writer())

	return logger
}
