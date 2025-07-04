package webserver

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/SwissOpenEM/Ingestor/internal/core"
	"github.com/SwissOpenEM/Ingestor/internal/webserver/wsconfig"
	"github.com/google/uuid"
)

func TestDatasetControllerBrowseFilesystem_LocationList(t *testing.T) {
	collectionLocations := map[string]string{
		"collection1": "/one/collection1",
		"folder":      "/another/collection2",
	}

	wsConf := wsconfig.WebServerConfig{
		AuthConf: wsconfig.AuthConf{
			Disable: true,
			RBACConf: wsconfig.RBACConf{
				AdminRole:             "none",
				CreateModifyTasksRole: "none2",
				ViewTasksRole:         "none3",
			},
		},
		PathsConf: wsconfig.PathsConf{
			CollectionLocations:     collectionLocations,
			ExtractorOutputLocation: "/tmp",
		},
		MetadataExtJobsConf: wsconfig.MetadataExtJobsConf{
			ConcurrencyLimit: 1,
			QueueSize:        1,
		},
		OtherConf: wsconfig.OtherConf{
			BackendAddress:             "localhost",
			Port:                       8080,
			LogLevel:                   "Debug",
			DisableServiceAccountCheck: true,
		},
	}
	i, err := NewIngestorWebServer("test", &core.TaskQueue{}, nil, nil, wsConf)
	if err != nil {
		t.Errorf("NewIngestorWebServer error: %s", err.Error())
		return
	}

	req := DatasetControllerBrowseFilesystemRequestObject{
		Params: DatasetControllerBrowseFilesystemParams{
			Path:     "/",
			Page:     getPointerOrNil(uint(1)),
			PageSize: getPointerOrNil(uint(10)),
		},
	}
	resp, err := i.DatasetControllerBrowseFilesystem(context.Background(), req)
	if err != nil {
		t.Errorf("DatasetControllerBrowseFilesystem error: %s", err.Error())
		return
	}

	resp200, ok := resp.(DatasetControllerBrowseFilesystem200JSONResponse)
	if !ok {
		resp400, ok := resp.(DatasetControllerBrowseFilesystem400TextResponse)
		if !ok {
			t.Errorf("unknown error object received")
			return
		}
		t.Errorf("error returned by controller: %s", resp400)
	}

	if len(resp200.Folders) != len(collectionLocations) {
		t.Errorf("The returned list of collection locations does not match in length the one that was set - got: %d, want: %d", len(resp200.Folders), len(collectionLocations))
		return
	}

	mismatchedKeys := []string{}
	for _, folder := range resp200.Folders {
		if _, ok := collectionLocations[folder.Name]; !ok {
			mismatchedKeys = append(mismatchedKeys, folder.Name)
		}
	}
	if len(mismatchedKeys) > 0 {
		t.Errorf("The returned list contains invalid elements: %v", mismatchedKeys)
		return
	}
}

func TestDatasetControllerBrowseFilesystem_ExampleListInCollection(t *testing.T) {
	collectionLocations := map[string]string{}

	testSession := uuid.New()
	testPath := filepath.Join(os.TempDir(), testSession.String())
	err := os.MkdirAll(testPath, 0777)
	if err != nil {
		t.Errorf("can't create test folder: %s", err.Error())
		return
	}

	collectionLocations["test"] = testPath

	testFolders := []string{"dataset1", "dataset2", "dataset3"}
	for _, folder := range testFolders {
		err = os.MkdirAll(filepath.Join(testPath, folder), 0777)
		if err != nil {
			t.Errorf("couldn't create %s's folder: %s", folder, err.Error())
			return
		}
	}

	wsConf := wsconfig.WebServerConfig{
		AuthConf: wsconfig.AuthConf{
			Disable: true,
			RBACConf: wsconfig.RBACConf{
				AdminRole:             "none",
				CreateModifyTasksRole: "none2",
				ViewTasksRole:         "none3",
			},
		},
		PathsConf: wsconfig.PathsConf{
			CollectionLocations:     collectionLocations,
			ExtractorOutputLocation: "/tmp",
		},
		MetadataExtJobsConf: wsconfig.MetadataExtJobsConf{
			ConcurrencyLimit: 1,
			QueueSize:        1,
		},
		OtherConf: wsconfig.OtherConf{
			BackendAddress:             "localhost",
			Port:                       8080,
			LogLevel:                   "Debug",
			DisableServiceAccountCheck: true,
		},
	}
	i, err := NewIngestorWebServer("test", &core.TaskQueue{}, nil, nil, wsConf)
	if err != nil {
		t.Errorf("NewIngestorWebServer error: %s", err.Error())
		return
	}

	req := DatasetControllerBrowseFilesystemRequestObject{
		Params: DatasetControllerBrowseFilesystemParams{
			Path:     "/test",
			Page:     getPointerOrNil(uint(1)),
			PageSize: getPointerOrNil(uint(10)),
		},
	}
	resp, err := i.DatasetControllerBrowseFilesystem(context.Background(), req)
	if err != nil {
		t.Errorf("DatasetControllerBrowseFilesystem error: %s", err.Error())
		return
	}

	resp200, ok := resp.(DatasetControllerBrowseFilesystem200JSONResponse)
	if !ok {
		resp400, ok := resp.(DatasetControllerBrowseFilesystem400TextResponse)
		if ok {
			t.Errorf("error 400 returned by controller: %s", resp400)
			return
		}
		resp401, ok := resp.(DatasetControllerBrowseFilesystem401TextResponse)
		if ok {
			t.Errorf("error 401 returned by controller: %s", resp401)
			return
		}
		resp500, ok := resp.(DatasetControllerBrowseFilesystem500TextResponse)
		if ok {
			t.Errorf("error 500 returned by controller: %s", resp500)
			return
		}
		t.Errorf("unknown error object received")
		return
	}

	if len(resp200.Folders) != len(testFolders) {
		t.Errorf("The returned list of folders does not match in length the one that was set - got: %d, want: %d", len(resp200.Folders), len(testFolders))
		return
	}

	mismatchedFolders := []string{}
	for _, folder := range resp200.Folders {
		if slices.Contains(testFolders, folder.Name); !ok {
			mismatchedFolders = append(mismatchedFolders, folder.Name)
		}
	}
	if len(mismatchedFolders) > 0 {
		t.Errorf("The returned list of folders contains invalid elements: %v", mismatchedFolders)
		return
	}
}
