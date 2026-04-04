package logging

import (
	"io"
	"log/slog"
	"os"
)

type Config struct {
	Level  slog.Level
	Format string
	Output io.Writer
}

func New(cfg Config) *slog.Logger {
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	handlerOpts := &slog.HandlerOptions{Level: cfg.Level}
	switch cfg.Format {
	case "json":
		return slog.New(slog.NewJSONHandler(output, handlerOpts))
	default:
		return slog.New(slog.NewTextHandler(output, handlerOpts))
	}
}
