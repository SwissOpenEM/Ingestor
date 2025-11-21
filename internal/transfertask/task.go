package transfertask

import (
	"context"
	"fmt"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
)

type TransferMethod int

const (
	TransferS3 TransferMethod = iota + 1
	TransferGlobus
	TransferExtGlobus
	TransferNone
)

type TransferOptions struct {
	S3Endpoint  string
	S3Bucket    string
	Md5checksum bool
}

type TaskTransferConfig struct {
	S3TransferConfig
	GlobusTransferConfig
}

type TaskDetails struct {
	BytesTransferred int64
	BytesTotal       int64
	FilesTransferred int32
	FilesTotal       int32
	Status           Status
	Message          string
}

type Status int

const (
	Waiting Status = iota
	Transferring
	Finished
	Failed
	Cancelled
)

func (i *Status) ToStr() string {
	switch *i {
	case Waiting:
		return "waiting"
	case Transferring:
		return "transferring"
	case Finished:
		return "finished"
	case Failed:
		return "failed"
	default:
		return "invalid status"
	}
}

func buildMessage(datasetID string, datasetFolder DatasetFolder, message string) string {
	return fmt.Sprintf("%s (pid: %s): %s", path.Base(datasetFolder.FolderPath), datasetID, message)
}

type ArchivalJobInfo struct {
	OwnerUser    string
	OwnerGroup   string
	AutoArchive  bool
	ContactEmail string
}

type TransferTask struct {
	DatasetFolder   DatasetFolder
	datasetID       string
	fileList        []datasetIngestor.Datafile
	archivalJobInfo ArchivalJobInfo
	TransferMethod  TransferMethod
	Context         context.Context
	Cancel          context.CancelFunc
	transferObjects map[string]interface{}

	statusLock *sync.RWMutex
	details    *TaskDetails
}

type Result struct {
	ElapsedSeconds int
	DatasetPID     string
	Error          error
}

func CreateTransferTask(datasetID string, fileList []datasetIngestor.Datafile, datasetFolder DatasetFolder, datasetOwnerUser string, datasetOwnerGroup string, contactEmail string, autoArchive bool, transferMethod TransferMethod, transferObjects map[string]interface{}, cancel context.CancelFunc) TransferTask {
	totalBytes := int64(0)
	for _, file := range fileList {
		totalBytes += int64(file.Size)
	}
	return TransferTask{
		datasetID:     datasetID,
		fileList:      fileList,
		DatasetFolder: datasetFolder,
		archivalJobInfo: ArchivalJobInfo{
			OwnerUser:    datasetOwnerUser,
			OwnerGroup:   datasetOwnerGroup,
			ContactEmail: contactEmail,
			AutoArchive:  autoArchive,
		},
		TransferMethod:  transferMethod,
		transferObjects: transferObjects,
		Cancel:          cancel,
		details: &TaskDetails{
			BytesTransferred: 0,
			BytesTotal:       totalBytes,
			FilesTransferred: 0,
			FilesTotal:       int32(len(fileList)),
			Status:           Waiting,
			Message:          buildMessage(datasetID, datasetFolder, "in waiting list"),
		},
		statusLock: &sync.RWMutex{},
	}
}

func (t *TransferTask) GetDetails() TaskDetails {
	t.statusLock.RLock()
	defer t.statusLock.RUnlock()
	copy := *t.details
	return copy
}

func (t *TransferTask) Queued() {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status != Waiting {
		return
	}
	t.details.Message = buildMessage(t.datasetID, t.DatasetFolder, "queued")
}

func (t *TransferTask) TransferStarted() {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status != Waiting {
		return
	}
	t.details.Status = Transferring
	t.details.Message = buildMessage(t.datasetID, t.DatasetFolder, "transferring")
}

func (t *TransferTask) UpdateProgress(bytesTransferred *int64, filesTransferred *int32) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status != Transferring {
		return
	}
	if bytesTransferred != nil {
		t.details.BytesTransferred = *bytesTransferred
	}
	if filesTransferred != nil {
		t.details.FilesTransferred = *filesTransferred
	}
}

func (t *TransferTask) Finished() {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status != Transferring {
		return
	}
	t.details.Status = Finished
	t.details.Message = buildMessage(t.datasetID, t.DatasetFolder, "transfer finished")
}

func (t *TransferTask) Failed(msg string) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status == Finished ||
		t.details.Status == Cancelled ||
		t.details.Status == Failed {
		return
	}
	t.details.Status = Failed
	t.details.Message = buildMessage(t.datasetID, t.DatasetFolder, msg)
}

func (t *TransferTask) Cancelled(msg string) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if t.details.Status == Failed ||
		t.details.Status == Cancelled ||
		t.details.Status == Finished {
		return
	}
	t.details.Status = Cancelled
	t.details.Message = buildMessage(t.datasetID, t.DatasetFolder, msg)
}

func (t *TransferTask) GetDatasetID() string {
	return t.datasetID
}

func (t *TransferTask) GetArchivalJobInfo() ArchivalJobInfo {
	return t.archivalJobInfo
}

func (t *TransferTask) GetFileList() []datasetIngestor.Datafile {
	return t.fileList
}

func (t *TransferTask) GetTransferObject(name string) interface{} {
	return t.transferObjects[name]
}

type TransferNotifier struct {
	totalBytes       int64
	bytesTransferred int64
	filesTransferred int32
	startTime        time.Time
	id               uuid.UUID
	notifier         ProgressNotifier
	TaskStatus       *Status
	TaskProgress     *TransferTask
}

func NewTransferNotifier(total int64, uploadID uuid.UUID, notifier ProgressNotifier, task *TransferTask) TransferNotifier {
	return TransferNotifier{totalBytes: total,
		bytesTransferred: 0,
		startTime:        time.Now(),
		id:               uploadID,
		notifier:         notifier,
		TaskProgress:     task,
	}
}

func (tn *TransferNotifier) IncreaseFileCount(i int32) {
	atomic.AddInt32(&tn.filesTransferred, i)
}

func (tn *TransferNotifier) AddUploadedBytes(numBytes int64) {
	atomic.AddInt64(&tn.bytesTransferred, numBytes)
}

func (tn *TransferNotifier) OnTaskCanceled(id uuid.UUID) {
	tn.notifier.OnTaskCanceled(id)
}

func (tn *TransferNotifier) UpdateTaskProgress() {
	tn.notifier.OnTaskProgress(tn.id, (int)(100*tn.bytesTransferred/tn.totalBytes))
	tn.TaskProgress.UpdateProgress(&tn.bytesTransferred, &tn.filesTransferred)
}
