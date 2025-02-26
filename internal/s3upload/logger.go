package s3upload

import (
	"log/slog"
	"sync"
)

var logger *slog.Logger
var loggerOnce sync.Once

func log() *slog.Logger {
	loggerOnce.Do(func() {
		logger = slog.Default().With("package", "s3upload")
	})
	return logger
}
