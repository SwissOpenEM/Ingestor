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
	"github.com/elliotchance/orderedmap/v2"
	"github.com/google/uuid"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
)

type TaskQueue struct {
	taskListLock       sync.RWMutex                                          // locking mechanism for uploadIds and datasetUploadTasks
	datasetUploadTasks *orderedmap.OrderedMap[uuid.UUID, *task.TransferTask] // For storing requests, mapped to the id's above
	inputChannel       chan *task.TransferTask                               // Requests to upload data are put into this channel
	resultChannel      chan task.Result                                      // The result of the upload is put into this channel

	AppContext  context.Context
	Config      Config
	Notifier    ProgressNotifier
	ServiceUser *UserCreds
}

func (w *TaskQueue) Startup() {
	w.inputChannel = make(chan *task.TransferTask)
	w.resultChannel = make(chan task.Result)
	w.datasetUploadTasks = orderedmap.NewOrderedMap[uuid.UUID, *task.TransferTask]()

	// start multiple go routines/workers that will listen on the input channel
	for worker := 1; worker <= w.Config.Misc.ConcurrencyLimit; worker++ {
		go w.startWorker()
	}
}

func (w *TaskQueue) AddTransferTask(datasetId string, fileList []datasetIngestor.Datafile, totalSize int64, metadataMap map[string]interface{}, taskId uuid.UUID) error {
	transferMethod := w.GetTransferMethod()
	task := task.CreateTransferTask(datasetId, fileList, task.DatasetFolder{Id: taskId}, metadataMap, transferMethod, nil)

	switch v := metadataMap["sourceFolder"].(type) {
	case string:
		// the collection location has to be added to get the absolute path of the dataset
		task.DatasetFolder.FolderPath = path.Join(w.Config.WebServer.CollectionLocation, filepath.FromSlash(v))
	default:
		return errors.New("sourceFolder in metadata isn't a string")
	}
	msg := "added"
	size := int(totalSize)
	task.SetStatus(nil, &size, nil, nil, nil, nil, nil, &msg)

	w.taskListLock.Lock()
	defer w.taskListLock.Unlock()
	w.datasetUploadTasks.Set(taskId, &task)

	return nil
}

// Go routine that listens on the channel continously for upload requests and executes uploads.
func (w *TaskQueue) startWorker() {
	for transferTask := range w.inputChannel {
		task_context, cancel := context.WithCancel(w.AppContext)

		transferTask.Cancel = cancel

		result := w.TransferDataset(task_context, transferTask)
		if result.Error == nil {
			falseVal := false
			trueVal := true
			message := "finished"
			transferTask.SetStatus(nil, nil, nil, nil, &falseVal, nil, &trueVal, &message)
		} else {
			trueVal := true
			message := fmt.Sprintf("failed - error: %s", result.Error.Error())
			transferTask.SetStatus(nil, nil, nil, nil, &trueVal, nil, &trueVal, &message)
		}
		w.resultChannel <- result
	}
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

func (w *TaskQueue) ScheduleTask(id uuid.UUID) {
	w.taskListLock.RLock()
	ingestionTask, found := w.datasetUploadTasks.Get(id)
	w.taskListLock.RUnlock()
	if !found {
		fmt.Println("Scheduling upload failed for: ", id)
		return
	}
	msg := "queued"
	ingestionTask.SetStatus(nil, nil, nil, nil, nil, nil, nil, &msg)

	// Go routine to handle result and errors
	go func(id uuid.UUID) {
		taskResult := <-w.resultChannel
		if taskResult.Error != nil {
			w.Notifier.OnTaskFailed(id, taskResult.Error)
			println(fmt.Sprint(taskResult.Error))
		} else {
			w.Notifier.OnTaskCompleted(id, taskResult.Elapsed_seconds)
			println(taskResult.Dataset_PID, taskResult.Elapsed_seconds)
		}
	}(ingestionTask.DatasetFolder.Id)

	// Go routine to schedule the upload asynchronously
	go func(folder task.DatasetFolder) {
		fmt.Println("Scheduled upload for: ", folder)
		w.Notifier.OnTaskScheduled(folder.Id)

		// this channel is read by the go routines that does the actual upload
		w.inputChannel <- ingestionTask
	}(ingestionTask.DatasetFolder)
}

func (w *TaskQueue) GetTaskStatus(id uuid.UUID) (task.TaskStatus, error) {
	w.taskListLock.RLock()
	t, found := w.datasetUploadTasks.Get(id)
	w.taskListLock.RUnlock()
	if !found {
		return task.TaskStatus{}, fmt.Errorf("no task exists with id '%s'", id.String())
	}
	return t.GetStatus(), nil
}

func (w *TaskQueue) GetTaskStatusList(start uint, end uint) (idList []uuid.UUID, statusList []task.TaskStatus, err error) {
	if end < start {
		return idList, statusList, errors.New("end index is smaller than start index")
	}

	w.taskListLock.RLock()
	defer w.taskListLock.RUnlock()

	taskListLen := w.datasetUploadTasks.Len()
	end = min(end, uint(taskListLen))

	keys := w.datasetUploadTasks.Keys()
	for i := start; i < end; i++ {
		task, _ := w.datasetUploadTasks.Get(keys[i])
		idList = append(idList, keys[i])
		statusList = append(statusList, task.GetStatus())
	}

	return idList, statusList, err
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
		return t.FolderPath
	}
	return ""
}

func TestIngestionFunction(task_context context.Context, task task.TransferTask, config Config, notifier ProgressNotifier) (string, error) {
	start := time.Now()

	for i := 0; i < 10; i++ {
		time.Sleep(time.Second * 1)
		now := time.Now()
		elapsed := now.Sub(start)
		notifier.OnTaskProgress(task.DatasetFolder.Id, i+1, 10, int(elapsed.Seconds()))
	}
	return "1", nil

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
	}
	return transferMethod
}
