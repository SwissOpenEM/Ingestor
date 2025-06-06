package globustransfer

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/transfertask"
	"github.com/SwissOpenEM/globus"
)

type File struct {
	Path      string
	IsSymlink bool
}

func checkTransfer(client *globus.GlobusClient, globusTaskId string) (bytesTransferred int, filesTransferred int, totalFiles int, completed bool, err error) {
	globusTask, err := client.TransferGetTaskByID(globusTaskId)
	if err != nil {
		return 0, 0, 1, false, fmt.Errorf("globus: can't continue transfer because an error occured while polling the task \"%s\": %v", globusTaskId, err)
	}
	switch globusTask.Status {
	case "ACTIVE":
		totalFiles := globusTask.Files
		if globusTask.FilesSkipped != nil {
			totalFiles -= *globusTask.FilesSkipped
		}
		return globusTask.BytesTransferred, globusTask.FilesTransferred, totalFiles, false, nil
	case "INACTIVE":
		return 0, 0, 1, false, fmt.Errorf("globus: transfer became inactive, manual intervention required")
	case "SUCCEEDED":
		totalFiles := globusTask.Files
		if globusTask.FilesSkipped != nil {
			totalFiles -= *globusTask.FilesSkipped
		}
		return globusTask.BytesTransferred, globusTask.FilesTransferred, totalFiles, true, nil
	case "FAILED":
		return 0, 0, 1, false, fmt.Errorf("globus: task failed with the following error - code: \"%s\" description: \"%s\"", globusTask.FatalError.Code, globusTask.FatalError.Description)
	default:
		return 0, 0, 1, false, fmt.Errorf("globus: unknown task status: %s", globusTask.Status)
	}
}

// globus transfer task function, uses the notifier to update the status of the transfer
func TransferFiles(
	client *globus.GlobusClient,
	SourceCollectionID string,
	CollectionRootPath string,
	DestinationCollectionID string,
	DestinationPathTemplate string,
	datasetId string,
	username string,
	taskCtx context.Context,
	datasetPath string,
	fileList []File,
	transferNotifier *transfertask.TransferNotifier,
) error {
	// transfer given filelist
	var filePathList []string
	var fileIsSymlinkList []bool
	for _, file := range fileList {
		filePathList = append(filePathList, filepath.ToSlash(file.Path))
		fileIsSymlinkList = append(fileIsSymlinkList, file.IsSymlink)
	}
	datasetPath = filepath.ToSlash(datasetPath)

	destParams := destPathParamsStruct{
		DatasetFolder: path.Base(datasetPath),
		SourceFolder:  datasetPath,
		Pid:           datasetId,
		PidShort:      path.Base(datasetId),
		PidPrefix:     path.Dir(datasetId),
		PidEncoded:    url.PathEscape(datasetId),
		Username:      username,
	}

	finalDestinationPath, err := templateDestinationFolder(destParams)
	if err != nil {
		return err
	}

	result, err := client.TransferFileList(
		SourceCollectionID,
		strings.TrimPrefix(datasetPath, CollectionRootPath),
		DestinationCollectionID,
		finalDestinationPath,
		filePathList,
		fileIsSymlinkList,
		true,
	)
	if err != nil {
		return fmt.Errorf("globus: an error occured when requesting dataset transfer: %v", err)
	}
	if result.Code != "Accepted" {
		return fmt.Errorf("globus: transfer was not accepted - code: \"%s\", message: \"%s\"", result.Code, result.Message)
	}

	// task monitoring
	globusTaskId := result.TaskId
	var taskCompleted bool
	var bytesTransferred, filesTransferred int

	// note: the totalFiles variable here uses the count returned by Globus
	//   this can change over the course of the transfer, as Globus succeeds in finding the files
	//   (recursion, checking their existence...)

	bytesTransferred, filesTransferred, _, taskCompleted, err = checkTransfer(client, globusTaskId)
	if err != nil {
		return err
	}

	transferNotifier.AddUploadedBytes(int64(bytesTransferred))
	transferNotifier.IncreaseFileCount(int32(filesTransferred))
	transferNotifier.UpdateTaskProgress()

	if taskCompleted {
		return nil
	}

	transferUpdater := time.After(1 * time.Minute)
	for {
		select {
		case <-taskCtx.Done():
			// we're cancelling the task
			result, err := client.TransferCancelTaskByID(globusTaskId)
			if err != nil {
				return fmt.Errorf("globus: couldn't cancel task: %v", err)
			}
			if result.Code != "Canceled" {
				return fmt.Errorf("globus: couldn't cancel task - code: \"%s\", message: \"%s\"", result.Code, result.Message)
			}
			return nil
		case <-transferUpdater:
			// check state of transfer
			transferUpdater = time.After(1 * time.Minute)
			bytesTransferred, filesTransferred, _, taskCompleted, err = checkTransfer(client, globusTaskId)
			if err != nil {
				return err // transfer cannot be finished: irrecoverable error
			}

			transferNotifier.AddUploadedBytes(int64(bytesTransferred))
			transferNotifier.IncreaseFileCount(int32(filesTransferred))
			transferNotifier.UpdateTaskProgress()

			if taskCompleted {
				return nil // we're done!
			}
		}
	}
}
