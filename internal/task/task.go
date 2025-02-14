package task

import (
	"context"
	"sync"

	"github.com/paulscherrerinstitute/scicat-cli/v3/datasetIngestor"
)

type TransferMethod int

const (
	TransferS3 TransferMethod = iota + 1
	TransferGlobus
)

type TransferOptions struct {
	S3_endpoint string
	S3_Bucket   string
	Md5checksum bool
}

type TaskTransferConfig struct {
	S3TransferConfig
	GlobusTransferConfig
}

type TaskStatus struct {
	BytesTransferred int
	BytesTotal       int
	FilesTransferred int
	FilesTotal       int
	State            State
	Message          string
}

type State int

const (
	Waiting State = iota
	Started
	Finished
	Failed
	Cancelled
)

func (i *State) ToStr() string {
	switch *i {
	case Waiting:
		return "waiting"
	case Started:
		return "started"
	case Finished:
		return "finished"
	case Failed:
		return "failed"
	default:
		return "invalid state"
	}
}

type TransferTask struct {
	// DatasetFolderId   uuid.UUID
	DatasetFolder
	datasetId       string
	fileList        []datasetIngestor.Datafile
	DatasetMetadata map[string]interface{}
	TransferMethod  TransferMethod
	Cancel          context.CancelFunc
	status          *TaskStatus
	statusLock      *sync.RWMutex
}

type Result struct {
	Elapsed_seconds int
	Dataset_PID     string
	Error           error
}

func CreateTransferTask(datasetId string, fileList []datasetIngestor.Datafile, datasetFolder DatasetFolder, metadata map[string]interface{}, transferMethod TransferMethod, cancel context.CancelFunc) TransferTask {
	return TransferTask{
		datasetId:       datasetId,
		fileList:        fileList,
		DatasetFolder:   datasetFolder,
		DatasetMetadata: metadata,
		TransferMethod:  transferMethod,
		Cancel:          cancel,
		status:          &TaskStatus{},
		statusLock:      &sync.RWMutex{},
	}
}

func (t *TransferTask) GetStatus() TaskStatus {
	t.statusLock.RLock()
	defer t.statusLock.RUnlock()
	copy := *t.status
	return copy
}

func (t *TransferTask) SetStatus(
	bytesTransferred *int,
	bytesTotal *int,
	filesTransferred *int,
	filesTotal *int,
	state *State,
	message *string,
) {
	t.statusLock.Lock()
	defer t.statusLock.Unlock()
	if bytesTransferred != nil {
		t.status.BytesTransferred = *bytesTransferred
	}
	if bytesTotal != nil {
		t.status.BytesTotal = *bytesTotal
	}
	if filesTransferred != nil {
		t.status.FilesTransferred = *filesTransferred
	}
	if filesTotal != nil {
		t.status.FilesTotal = *filesTotal
	}
	if state != nil {
		t.status.State = *state
	}
	if message != nil {
		t.status.Message = *message
	}
}

func (t *TransferTask) GetDatasetId() string {
	return t.datasetId
}

func (t *TransferTask) GetFileList() []datasetIngestor.Datafile {
	return t.fileList
}
