package transfertask

import (
	"context"
	"path"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type DatasetFolder struct {
	ID         uuid.UUID
	FolderPath string
}

// Select a folder using a native menu
func SelectFolder(context context.Context) (DatasetFolder, error) {
	dialogOptions := runtime.OpenDialogOptions{
		DefaultDirectory: "./",
		Title:            "Select Dataset",
	}

	folder, err := runtime.OpenDirectoryDialog(context, dialogOptions)
	if err != nil || folder == "" {
		return DatasetFolder{}, err
	}

	id := uuid.New()

	selectedFolder := DatasetFolder{FolderPath: path.Clean(folder), ID: id}
	return selectedFolder, nil
}
