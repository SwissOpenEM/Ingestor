package core

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
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
	if w.Config.Transfer.QueueSize > 0 {
		w.taskPool = pond.NewPool(w.Config.Transfer.ConcurrencyLimit, pond.WithQueueSize(w.Config.Transfer.QueueSize))
	} else {
		w.taskPool = pond.NewPool(w.Config.Transfer.ConcurrencyLimit)
	}
}

func (w *TaskQueue) AddTransferTask(transferObjects map[string]interface{}, datasetId string, fileList []datasetIngestor.Datafile, totalSize int64, metadataMap map[string]interface{}, taskId uuid.UUID) error {
	transferMethod := w.GetTransferMethod()
	t := task.CreateTransferTask(datasetId, fileList, task.DatasetFolder{Id: taskId}, metadataMap, transferMethod, transferObjects, nil)

	switch v := metadataMap["sourceFolder"].(type) {
	case string:
		// the collection location has to be added to get the absolute path of the dataset
		t.DatasetFolder.FolderPath = path.Join(w.Config.WebServer.CollectionLocation, filepath.FromSlash(v))
	default:
		return errors.New("sourceFolder in metadata isn't a string")
	}
	t.UpdateDetails(
		task.SetBytesTotal(int(totalSize)),
		task.SetMessage("added"),
	)

	w.taskListLock.Lock()
	defer w.taskListLock.Unlock()
	w.datasetUploadTasks.Set(taskId, &t)

	return nil
}

func (w *TaskQueue) executeTransferTask(t *task.TransferTask) {
	task_context, cancel := context.WithCancel(w.AppContext)

	t.Cancel = cancel

	_, err := w.TransferDataset(task_context, t)
	if err != nil {
		t.UpdateDetails(
			task.SetStatus(task.Failed),
			task.SetMessage(fmt.Sprintf("failed - error: %s", err.Error())),
		)
	}
	t.UpdateDetails(
		task.SetStatus(task.Finished),
		task.SetMessage("finished"),
	)
}

func (w *TaskQueue) CancelTask(id uuid.UUID) {
	w.taskListLock.RLock()
	task, ok := w.datasetUploadTasks.Get(id)
	w.taskListLock.RUnlock()
	if !ok {
		return
	}
	if task.Cancel != nil {
		task.Cancel()
	}
	w.Notifier.OnTaskCanceled(id)
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
	ingestionTask.UpdateDetails(task.SetMessage("queued"))
	task_context, cancel := context.WithCancel(w.AppContext)
	ingestionTask.Context = task_context
	ingestionTask.Cancel = cancel

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
