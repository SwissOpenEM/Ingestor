package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/task"
	"github.com/google/uuid"
)

type TaskQueue struct {
	datasetUploadTasks sync.Map                // Datastructure to store all the upload requests
	inputChannel       chan task.IngestionTask // Requests to upload data are put into this channel
	resultChannel      chan task.Result        // The result of the upload is put into this channel
	AppContext         context.Context
	Config             Config
	Notifier           ProgressNotifier
}

func (w *TaskQueue) Startup() {

	w.inputChannel = make(chan task.IngestionTask)
	w.resultChannel = make(chan task.Result)

	// start multiple go routines/workers that will listen on the input channel
	for worker := 1; worker <= w.Config.Misc.ConcurrencyLimit; worker++ {
		go w.startWorker()
	}

}

func (w *TaskQueue) CreateTaskFromDatasetFolder(folder task.DatasetFolder) error {
	transferMethod := w.getTransferMethod()

	task := task.CreateIngestionTask(folder, map[string]interface{}{}, transferMethod, nil)
	_, found := w.datasetUploadTasks.Load(task.DatasetFolder.Id)
	if found {
		return errors.New("key exists")
	}
	w.datasetUploadTasks.Store(task.DatasetFolder.Id, task)

	w.Notifier.OnTaskAdded(task.DatasetFolder.Id, task.DatasetFolder.FolderPath)
	return nil
}

func (w *TaskQueue) CreateTaskFromMetadata(id uuid.UUID, metadata map[string]interface{}) {
	transferMethod := w.getTransferMethod()
	task := task.CreateIngestionTask(task.DatasetFolder{}, metadata, transferMethod, nil)
	w.datasetUploadTasks.Store(id, task)
}

// Go routine that listens on the channel continously for upload requests and executes uploads.
func (w *TaskQueue) startWorker() {
	for ingestionTask := range w.inputChannel {
		task_context, cancel := context.WithCancel(w.AppContext)

		ingestionTask.Cancel = cancel

		result := w.IngestDataset(task_context, ingestionTask)
		w.resultChannel <- result
	}
}

func (w *TaskQueue) CancelTask(id uuid.UUID) {
	value, found := w.datasetUploadTasks.Load(id)
	if found {
		f := value.(task.IngestionTask)
		if f.Cancel != nil {
			f.Cancel()
		}
		w.Notifier.OnTaskCanceled(id)
	}
}

func (w *TaskQueue) RemoveTask(id uuid.UUID) error {
	value, found := w.datasetUploadTasks.Load(id)
	if !found {
		return errors.New("task not found")
	}
	f := value.(task.IngestionTask)
	if f.Cancel != nil {
		f.Cancel()
	}
	w.datasetUploadTasks.Delete(id)
	w.Notifier.OnTaskRemoved(id)
	return nil
}

func (w *TaskQueue) ScheduleTask(id uuid.UUID) {

	value, found := w.datasetUploadTasks.Load(id)
	if !found {
		fmt.Println("Scheduling upload failed for: ", id)
		return
	}

	ingestionTask := value.(task.IngestionTask)

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
	value, found := w.datasetUploadTasks.Load(id)
	if !found {
		return task.TaskStatus{}, fmt.Errorf("no task exists with id '%s'", id.String())
	}
	t := value.(task.IngestionTask)
	return t.GetStatus(), nil
}

func TestIngestionFunction(task_context context.Context, task task.IngestionTask, config Config, notifier ProgressNotifier) (string, error) {

	start := time.Now()

	for i := 0; i < 10; i++ {
		time.Sleep(time.Second * 1)
		now := time.Now()
		elapsed := now.Sub(start)
		notifier.OnTaskProgress(task.DatasetFolder.Id, i+1, 10, int(elapsed.Seconds()))
	}
	return "1", nil

}

func (w *TaskQueue) IngestDataset(task_context context.Context, ingestionTask task.IngestionTask) task.Result {
	start := time.Now()
	// datasetPID, _ := TestIngestionFunction(task_context, task, w.Config, w.Notifier)
	datasetPID, err := IngestDataset(task_context, ingestionTask, w.Config, w.Notifier)
	end := time.Now()
	elapsed := end.Sub(start)
	return task.Result{Dataset_PID: datasetPID, Elapsed_seconds: int(elapsed.Seconds()), Error: err}
}

func (w *TaskQueue) getTransferMethod() (transferMethod task.TransferMethod) {
	switch strings.ToLower(w.Config.Transfer.Method) {
	case "globus":
		transferMethod = task.TransferGlobus
	case "s3":
		transferMethod = task.TransferS3
	}
	return transferMethod
}
