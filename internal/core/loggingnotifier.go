package core

import (
	"log/slog"

	"github.com/google/uuid"
)

type LoggingNotifier struct {
	logger slog.Logger
}

func NewLoggingNotifier() *LoggingNotifier {
	p := new(LoggingNotifier)
	p.logger = *slog.Default().With()
	return p
}

func (n *LoggingNotifier) OnTaskScheduled(id uuid.UUID) {
	n.logger.Info("Task scheduled", "id", id)
}
func (n *LoggingNotifier) OnTaskCanceled(id uuid.UUID) {
	n.logger.Info("Task canceled", "id", id)
}
func (n *LoggingNotifier) OnTaskAdded(id uuid.UUID, folder string) {
	n.logger.Info("Task added", "id", id, "folder", folder)
}
func (n *LoggingNotifier) OnTaskRemoved(id uuid.UUID) {
	n.logger.Info("Task removed", "id", id)
}
func (n *LoggingNotifier) OnTaskFailed(id uuid.UUID, err error) {
	n.logger.Error("Task failed", "id", id, "err", err.Error())
}
func (n *LoggingNotifier) OnTaskCompleted(id uuid.UUID, elapsedSeconds int) {
	n.logger.Info("Task completed", "id", id, "elapsed", elapsedSeconds)
}
func (n *LoggingNotifier) OnTaskProgress(id uuid.UUID, percentage int) {
	n.logger.Info("Task progress", "id", id, "percentage", percentage)
}
