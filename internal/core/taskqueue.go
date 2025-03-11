package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	task "github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/alitto/pond/v2"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/google/uuid"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
)

type TaskQueue struct {
	taskListLock       sync.RWMutex                                          // locking mechanism for uploadIds and datasetUploadTasks
	datasetUploadTasks *orderedmap.OrderedMap[uuid.UUID, *task.TransferTask] // For storing requests, mapped to the id's above
	inputChannel       chan *task.TransferTask                               // Requests to upload data are put into this channel
	taskPool           pond.Pool

	AppContext  context.Context
	Config      Config
	Notifier    task.ProgressNotifier
	ServiceUser *UserCreds
}

func (w *TaskQueue) Startup() {
	w.inputChannel = make(chan *task.TransferTask)
	w.datasetUploadTasks = orderedmap.NewOrderedMap[uuid.UUID, *task.TransferTask]()
	w.taskPool = pond.NewPool(w.Config.Transfer.ConcurrencyLimit, pond.WithQueueSize(w.Config.Transfer.QueueSize))
}

func (w *TaskQueue) AddTransferTask(datasetId string, fileList []datasetIngestor.Datafile, taskId uuid.UUID, folderPath string, ownerGroup string, autoArchive bool, transferObjects map[string]interface{}) error {
	transferMethod := w.GetTransferMethod()
	t := task.CreateTransferTask(
		datasetId,
		fileList,
		task.DatasetFolder{
			Id:         taskId,
			FolderPath: folderPath,
		},
		ownerGroup,
		transferMethod,
		autoArchive,
		transferObjects,
		nil,
	)

	w.taskListLock.Lock()
	defer w.taskListLock.Unlock()
	w.datasetUploadTasks.Set(taskId, &t)

	return nil
}

func (w *TaskQueue) executeTransferTask(t *task.TransferTask) {
	task_context, cancel := context.WithCancel(w.AppContext)

	t.Cancel = cancel

	r := w.TransferDataset(task_context, t)
	if r.Error != nil {
		t.UpdateDetails(
			task.SetStatus(task.Failed),
			task.SetMessage(fmt.Sprintf("failed - error: %s", r.Error.Error())),
		)
		w.Notifier.OnTaskFailed(t.DatasetFolder.Id, r.Error)
		return
	}

	// if not cancelled, mark as finished
	if t.GetDetails().Status != task.Cancelled {
		t.UpdateDetails(
			task.SetStatus(task.Finished),
			task.SetMessage("task was finished successfully"),
		)
		w.Notifier.OnTaskCompleted(t.DatasetFolder.Id, r.Elapsed_seconds)
	}
}

func (w *TaskQueue) CancelTask(id uuid.UUID) {
	w.taskListLock.RLock()
	uploadTask, ok := w.datasetUploadTasks.Get(id)
	w.taskListLock.RUnlock()
	if !ok {
		return
	}
	if uploadTask.Cancel != nil {
		// note: the task is marked as cancelled in advance in order for the task executer to not mark it as finished
		uploadTask.UpdateDetails(task.SetStatus(task.Cancelled), task.SetMessage("transfer was cancelled"))
		w.Notifier.OnTaskCanceled(id)
		uploadTask.Cancel()
	}
}

func (w *TaskQueue) RemoveTask(id uuid.UUID) error {
	var unlockOnce sync.Once
	w.taskListLock.Lock()
	defer unlockOnce.Do(w.taskListLock.Unlock)

	f, found := w.datasetUploadTasks.Get(id)
	if !found {
		return errors.New("task not found")
	}
	if f.Cancel != nil {
		f.Cancel()
	}
	if !w.datasetUploadTasks.Delete(id) {
		return errors.New("could not delete key")
	}

	unlockOnce.Do(w.taskListLock.Unlock)
	w.Notifier.OnTaskRemoved(id)
	return nil
}

func (w *TaskQueue) ScheduleTask(id uuid.UUID) error {
	w.taskListLock.RLock()
	ingestionTask, found := w.datasetUploadTasks.Get(id)
	w.taskListLock.RUnlock()
	if !found {
		return fmt.Errorf("task with id '%s' not found", id.String())
	}

	task_context, cancel := context.WithCancel(w.AppContext)
	ingestionTask.Context = task_context
	ingestionTask.Cancel = cancel

	ingestionTask.UpdateDetails(task.SetMessage("queued"))
	w.Notifier.OnTaskScheduled(ingestionTask.DatasetFolder.Id)

	w.taskPool.Submit(func() { w.executeTransferTask(ingestionTask) })
	return nil
}

func (w *TaskQueue) GetTaskDetails(id uuid.UUID) (task.TaskDetails, error) {
	w.taskListLock.RLock()
	t, found := w.datasetUploadTasks.Get(id)
	w.taskListLock.RUnlock()
	if !found {
		return task.TaskDetails{}, fmt.Errorf("no task exists with id '%s'", id.String())
	}
	return t.GetDetails(), nil
}

func (w *TaskQueue) GetTaskDetailsList(start uint, end uint) (idList []uuid.UUID, detailsList []task.TaskDetails, err error) {
	if end < start {
		return idList, detailsList, errors.New("end index is smaller than start index")
	}

	w.taskListLock.RLock()
	defer w.taskListLock.RUnlock()

	taskListLen := w.datasetUploadTasks.Len()
	end = min(end, uint(taskListLen))

	keys := w.datasetUploadTasks.Keys()
	for i := start; i < end; i++ {
		task, _ := w.datasetUploadTasks.Get(keys[i])
		idList = append(idList, keys[i])
		detailsList = append(detailsList, task.GetDetails())
	}

	return idList, detailsList, err
}

func (w *TaskQueue) GetTaskCount() int {
	w.taskListLock.RLock()
	defer w.taskListLock.RUnlock()
	return w.datasetUploadTasks.Len()
}

func (w *TaskQueue) GetTaskFolder(id uuid.UUID) string {
	w.taskListLock.RLock()
	defer w.taskListLock.RUnlock()

	if t, ok := w.datasetUploadTasks.Get(id); ok {
		return t.DatasetFolder.FolderPath
	}
	return ""
}

func (w *TaskQueue) TransferDataset(taskCtx context.Context, it *task.TransferTask) task.Result {
	start := time.Now()
	err := TransferDataset(taskCtx, it, w.ServiceUser, w.Config, w.Notifier)
	end := time.Now()
	elapsed := end.Sub(start)
	return task.Result{Elapsed_seconds: int(elapsed.Seconds()), Error: err}
}

func (w *TaskQueue) GetTransferMethod() (transferMethod task.TransferMethod) {
	switch strings.ToLower(w.Config.Transfer.Method) {
	case "globus":
		transferMethod = task.TransferGlobus
	case "s3":
		transferMethod = task.TransferS3
	default:
		panic("unknown transfer method")
	}
	return transferMethod
}
