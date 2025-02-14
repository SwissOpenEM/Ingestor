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
	Failed           bool
	Started          bool
	Finished         bool
	StatusMessage    string
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
	transferObjects map[string]interface{}
}

type Result struct {
	Elapsed_seconds int
	Dataset_PID     string
	Error           error
}

func CreateTransferTask(datasetId string, fileList []datasetIngestor.Datafile, datasetFolder DatasetFolder, metadata map[string]interface{}, transferMethod TransferMethod, transferObjects map[string]interface{}, cancel context.CancelFunc) TransferTask {
	return TransferTask{
		datasetId:       datasetId,
		fileList:        fileList,
		DatasetFolder:   datasetFolder,
		DatasetMetadata: metadata,
		TransferMethod:  transferMethod,
		transferObjects: transferObjects,
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
	failed *bool,
	started *bool,
	finished *bool,
	statusMessage *string,
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
	if failed != nil {
		t.status.Failed = *failed
	}
	if started != nil {
		t.status.Started = *started
	}
	if finished != nil {
		t.status.Finished = *finished
	}
	if statusMessage != nil {
		t.status.StatusMessage = *statusMessage
	}
}

func (t *TransferTask) GetDatasetId() string {
	return t.datasetId
}

func (t *TransferTask) GetFileList() []datasetIngestor.Datafile {
	return t.fileList
}

func (t *TransferTask) GetTransferObject(name string) interface{} {
	return t.transferObjects[name]
}
