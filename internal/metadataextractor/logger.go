package metadataextractor

import (
	"log/slog"
	"sync"
)

var logger *slog.Logger
var loggerOnce sync.Once

func log() *slog.Logger {
	loggerOnce.Do(func() {
		logger = slog.Default().With("package", "metadataextractor")
	})
	return logger
}
