package core

import (
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/google/uuid"
)

/* cannot use notifier (variable of type TransferNotifier) as globustransfer.Notifier value in argument to globustransfer.GlobusTransfer:
TransferNotifier does not implement globustransfer.Notifier (missing method OnTransferCancelled) */

type TransferNotifier struct {
	transferTask *task.TransferTask
	totalBytes   int
	taskNotifier task.ProgressNotifier
	start        time.Time
}

func CreateTransferNotifier(id uuid.UUID, transferTask *task.TransferTask, totalBytes int, taskNotifier task.ProgressNotifier, start time.Time) TransferNotifier {
	return TransferNotifier{
		transferTask: transferTask,
		totalBytes:   totalBytes,
		start:        start,
	}
}

func (t *TransferNotifier) OnTransferProgress(bytesTransferred int, filesTransferred int) {
	t.transferTask.UpdateDetails(
		task.SetBytesTransferred(bytesTransferred),
		task.SetFilesTransferred(filesTransferred),
	)
	t.taskNotifier.OnTaskProgress(t.transferTask.DatasetFolder.Id, bytesTransferred*100/t.totalBytes)
}

func (t *TransferNotifier) OnTransferFinished() {
	t.transferTask.UpdateDetails(
		task.SetStatus(task.Finished),
		task.SetMessage("task completed successfully"),
	)

	t.taskNotifier.OnTaskCompleted(
		t.transferTask.DatasetFolder.Id,
		int(time.Second.Round(time.Since(t.start)).Seconds()),
	)
}

func (t *TransferNotifier) OnTransferCancelled() {
	t.transferTask.UpdateDetails(
		task.SetStatus(task.Cancelled),
		task.SetMessage("the transfer was cancelled"),
	)

	t.taskNotifier.OnTaskCanceled(t.transferTask.DatasetFolder.Id)
}

func (t *TransferNotifier) OnTransferFailed(err error) {
	t.transferTask.UpdateDetails(
		task.SetStatus(task.Failed),
		task.SetMessage(err.Error()),
	)

	t.taskNotifier.OnTaskFailed(t.transferTask.DatasetFolder.Id, err)
}
