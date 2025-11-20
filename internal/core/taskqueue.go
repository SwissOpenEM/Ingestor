package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	task "github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/alitto/pond/v2"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/google/uuid"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
)

type TaskQueue struct {
	taskListLock       sync.RWMutex                                          // locking mechanism for uploadIds and datasetUploadTasks
	datasetUploadTasks *orderedmap.OrderedMap[uuid.UUID, *task.TransferTask] // For storing requests, mapped to the id's above
	taskPool           pond.Pool

	appContext  context.Context
	Config      Config
	notifier    task.ProgressNotifier
	serviceUser *UserCreds
}

func NewTaskQueueFromPool(ctx context.Context, config Config, notifier task.ProgressNotifier, serviceUser *UserCreds, pool pond.Pool) *TaskQueue {

	return &TaskQueue{
		datasetUploadTasks: orderedmap.NewOrderedMap[uuid.UUID, *task.TransferTask](),
		taskPool:           pool,
		appContext:         ctx,
		Config:             config,
		notifier:           notifier,
		serviceUser:        serviceUser,
	}
}

func (w *TaskQueue) AddTransferTask(datasetID string, fileList []datasetIngestor.Datafile, taskID uuid.UUID, folderPath string, ownerUser string, ownerGroup string, contactEmail string, autoArchive bool, transferObjects map[string]interface{}) error {
	transferMethod := w.GetTransferMethod()
	if transferMethod == task.TransferNone {
		return nil
	}
	t := task.CreateTransferTask(
		datasetID,
		fileList,
		task.DatasetFolder{
			ID:         taskID,
			FolderPath: folderPath,
		},
		ownerUser,
		ownerGroup,
		contactEmail,
		autoArchive,
		transferMethod,
		transferObjects,
		nil,
	)

	w.taskListLock.Lock()
	defer w.taskListLock.Unlock()
	w.datasetUploadTasks.Set(taskID, &t)

	return nil
}

func (w *TaskQueue) executeTransferTask(t *task.TransferTask) {
	taskContext, cancel := context.WithCancel(w.appContext)
	t.Cancel = cancel

	r := w.TransferDataset(taskContext, t)
	if r.Error != nil {
		t.Failed(r.Error.Error())
		w.notifier.OnTaskFailed(t.DatasetFolder.ID, r.Error)
		return
	}

	// if not cancelled, mark as finished
	if t.GetDetails().Status != task.Cancelled {
		t.Finished()
		w.notifier.OnTaskCompleted(t.DatasetFolder.ID, r.ElapsedSeconds)
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
		uploadTask.Cancelled("transfer was cancelled by the user")
		w.notifier.OnTaskCanceled(id)
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
	w.notifier.OnTaskRemoved(id)
	return nil
}

func (w *TaskQueue) ScheduleTask(id uuid.UUID) error {
	w.taskListLock.RLock()
	transferTask, found := w.datasetUploadTasks.Get(id)
	w.taskListLock.RUnlock()
	if !found {
		return fmt.Errorf("task with id '%s' not found", id.String())
	}

	taskContext, cancel := context.WithCancel(w.appContext)
	transferTask.Context = taskContext
	transferTask.Cancel = cancel

	transferTask.Queued()
	w.notifier.OnTaskScheduled(transferTask.DatasetFolder.ID)

	w.taskPool.Submit(func() { w.executeTransferTask(transferTask) })
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

func (w *TaskQueue) CreateSubpool(size int) pond.Pool {
	return w.taskPool.NewSubpool(size)
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
	err := TransferDataset(taskCtx, it, w.serviceUser, w.Config, w.notifier)
	end := time.Now()
	elapsed := end.Sub(start)
	return task.Result{ElapsedSeconds: int(elapsed.Seconds()), Error: err}
}

func (w *TaskQueue) GetTransferMethod() (transferMethod task.TransferMethod) {
	switch strings.ToLower(w.Config.Transfer.Method) {
	case "globus":
		transferMethod = task.TransferGlobus
	case "extglobus":
		transferMethod = task.TransferExtGlobus
	case "s3":
		transferMethod = task.TransferS3
	case "none":
		transferMethod = task.TransferNone
	default:
		panic("unknown transfer method")
	}
	return transferMethod
}

func (w *TaskQueue) IsServiceUserSet() bool {
	return w.serviceUser != nil
}
