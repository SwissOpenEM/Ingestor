package collections

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetDatasetAbsolutePath(t *testing.T) {
	collectionLocations := map[string]string{
		"path1": "/some/path1",
		"path2": "/some/path2",
	}
	if runtime.GOOS == "windows" {
		collectionLocations = map[string]string{
			"path1": "C:\\some\\path1",
			"path2": "C:\\some\\path2",
		}
	}

	expectedPath := "/some/path1/sub/path/to/dataset"
	if runtime.GOOS == "windows" {
		expectedPath = "C:\\some\\path1\\sub\\path\\to\\dataset"
	}
	absPath, err := GetDatasetAbsolutePath(collectionLocations, filepath.Clean("/path1/sub/path/to/dataset"))
	if err != nil {
		t.Errorf("received error from function: %s", err.Error())
		return
	}
	if absPath != expectedPath {
		t.Errorf("path result is different from what is expected - got: '%s', want: '%s'", absPath, expectedPath)
		return
	}

	expectedPath = "/some/path2/another/path/to/dataset"
	if runtime.GOOS == "windows" {
		expectedPath = "C:\\some\\path2\\another\\path\\to\\dataset"
	}
	absPath, err = GetDatasetAbsolutePath(collectionLocations, filepath.Clean("/path2/another/path/to/dataset"))
	if err != nil {
		t.Errorf("received error from function: %s", err.Error())
		return
	}
	if absPath != expectedPath {
		t.Errorf("path result is different from what is expected - got: '%s', want: '%s'", absPath, expectedPath)
		return
	}
}
