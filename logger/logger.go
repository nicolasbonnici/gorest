package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func init() {
	Log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func SetLogger(logger *slog.Logger) {
	Log = logger
}
