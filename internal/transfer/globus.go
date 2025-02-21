package transfer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SwissOpenEM/Ingestor/internal/progress"
	"github.com/SwissOpenEM/globus"
)

/*func GlobusHealthCheck() error {
	// NOTE: this is not a proper health check and takes a long time to finish (~900ms)
	_, err := globusClient.TransferGetTaskList(0, 1)
	return err
}*/

type GlobusFile struct {
	Path      string // path to file (must be globus path)
	IsSymlink bool
}

type GlobusTransferParams struct {
	Client                *globus.GlobusClient
	SourceCollection      string       // globus source collection id
	SourcePrefixPath      string       // the prefix path to apply at the source side
	DestinationCollection string       // globus destination collection id
	DestinationPrefixPath string       // the prefix path to apply at the destination side
	TransferFolder        string       // folder to which the file list is relative (must be globus path)
	FileList              []GlobusFile // the filelist to transfer
}

func globusCheckTransfer(client *globus.GlobusClient, globusTaskId string) (bytesTransferred int, filesTransferred int, totalFiles int, completed bool, err error) {
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

func GlobusTransfer(
	transferCtx context.Context,
	gtp GlobusTransferParams,
	notifier progress.TransferNotifier,
) error {
	// transfer given filelist
	var filePathList []string
	var fileIsSymlinkList []bool
	for _, file := range gtp.FileList {
		filePathList = append(filePathList, file.Path)
		fileIsSymlinkList = append(fileIsSymlinkList, file.IsSymlink)
	}

	s := strings.Split(strings.Trim(gtp.TransferFolder, "/"), "/")
	transferFolderName := s[len(s)-1]

	result, err := gtp.Client.TransferFileList(
		gtp.SourceCollection,
		gtp.SourcePrefixPath+"/"+gtp.TransferFolder,
		gtp.DestinationCollection,
		gtp.DestinationPrefixPath+"/"+transferFolderName,
		filePathList,
		fileIsSymlinkList,
		true,
	)
	if err != nil {
		return fmt.Errorf("globus: an error occured when requesting transfer: %v", err)
	}
	if result.Code != "Accepted" {
		return fmt.Errorf("globus: transfer was not accepted - code: \"%s\", message: \"%s\"", result.Code, result.Message)
	}

	// task monitoring
	globusTaskId := result.TaskId
	var taskCompleted bool
	var bytesTransferred, filesTransferred, totalFiles int

	// note: the totalFiles variable here uses the count returned by Globus
	//   this can change over the course of the transfer, as Globus succeeds in finding the files
	//   (recursion, checking their existence...)

	bytesTransferred, filesTransferred, totalFiles, taskCompleted, err = globusCheckTransfer(gtp.Client, globusTaskId)
	if err != nil {
		return err
	}
	if totalFiles == 0 {
		totalFiles = 1 // needed because percentage meter goes NaN otherwise
	}
	notifier.OnTransferProgress(bytesTransferred, filesTransferred)
	if taskCompleted {
		notifier.OnTransferCompleted()
		return nil
	}

	transferUpdater := time.After(1 * time.Minute)
	for {
		select {
		case <-transferCtx.Done():
			// we're cancelling the task
			result, err := gtp.Client.TransferCancelTaskByID(globusTaskId)
			if err != nil {
				return fmt.Errorf("globus: couldn't cancel task: %v", err)
			}
			if result.Code != "Canceled" {
				return fmt.Errorf("globus: couldn't cancel task - code: \"%s\", message: \"%s\"", result.Code, result.Message)
			}
			notifier.OnTransferCancelled()
			return nil
		case <-transferUpdater:
			// check state of transfer
			transferUpdater = time.After(1 * time.Minute)
			bytesTransferred, filesTransferred, _, taskCompleted, err = globusCheckTransfer(gtp.Client, globusTaskId)
			if err != nil {
				return err // transfer cannot be finished: irrecoverable error
			}

			notifier.OnTransferProgress(bytesTransferred, filesTransferred)

			if taskCompleted {
				notifier.OnTransferCompleted()
				return nil // we're done!
			}
		}
	}
}
