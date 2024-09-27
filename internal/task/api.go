package task

import (
	"context"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type DatasetFolder struct {
	Id         uuid.UUID
	FolderPath string
}

// Select a folder using a native menu
func SelectFolder(context context.Context) (DatasetFolder, error) {
	dialogOptions := runtime.OpenDialogOptions{
		DefaultDirectory: "",
	}

	folder, err := runtime.OpenDirectoryDialog(context, dialogOptions)
	if err != nil || folder == "" {
		return DatasetFolder{}, err
	}

	id := uuid.New()

	selected_folder := DatasetFolder{FolderPath: folder, Id: id}
	return selected_folder, nil
}