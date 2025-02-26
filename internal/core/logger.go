package core

import (
	"log/slog"
	"sync"
)

var logger *slog.Logger
var loggerOnce sync.Once

func log() *slog.Logger {
	loggerOnce.Do(func() {
		logger = slog.Default().With("package", "core")
	})
	return logger
}
