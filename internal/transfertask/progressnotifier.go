package transfertask

import "github.com/google/uuid"

// Interface to notify about progress of a task
type ProgressNotifier interface {
	OnTaskScheduled(id uuid.UUID)
	OnTaskCanceled(id uuid.UUID)
	OnTaskAdded(id uuid.UUID, folder string)
	OnTaskRemoved(id uuid.UUID)
	OnTaskFailed(id uuid.UUID, err error)
	OnTaskCompleted(id uuid.UUID, seconds_elapsed int)
	OnTaskProgress(id uuid.UUID, percentage int)
}
