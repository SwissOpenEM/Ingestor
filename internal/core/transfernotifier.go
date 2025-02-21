package core

import (
	"fmt"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/notifiers"
	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/google/uuid"
)

type TransferTaskNotifier struct {
	id    uuid.UUID
	qn    notifiers.QueueNotifier
	tt    *task.TransferTask
	start time.Time
}

func createTransferTaskNotifier(id uuid.UUID, qn notifiers.QueueNotifier, tt *task.TransferTask, start time.Time) TransferTaskNotifier {
	return TransferTaskNotifier{
		id:    id,
		qn:    qn,
		tt:    tt,
		start: start,
	}
}

func (n TransferTaskNotifier) OnTransferProgress(bytesTransferred int, filesTransferred int) {
	n.qn.OnTaskProgress(n.id, filesTransferred, n.tt.GetDetails().FilesTotal, int(time.Since(n.start).Seconds()))
	n.tt.UpdateDetails(task.SetBytesTransferred(bytesTransferred), task.SetFilesTransferred(filesTransferred))
}

func (n TransferTaskNotifier) OnTransferCompleted() {
	n.qn.OnTaskCompleted(n.id, int(time.Since(n.start).Seconds()))
	n.tt.UpdateDetails(task.SetStatus(task.Finished), task.SetMessage("finished"))
}

func (n TransferTaskNotifier) OnTransferCancelled() {
	n.qn.OnTaskCanceled(n.id)
	n.tt.UpdateDetails(task.SetStatus(task.Cancelled), task.SetMessage("cancelled"))
}

func (n TransferTaskNotifier) OnTransferFailed(err error) {
	n.qn.OnTaskFailed(n.id, err)
	n.tt.UpdateDetails(task.SetStatus(task.Failed), task.SetMessage(fmt.Sprintf("task failed with the following error: %s", err.Error())))
}
