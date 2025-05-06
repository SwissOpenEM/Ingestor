package collections

import "testing"

func TestGetDatasetAbsolutePath(t *testing.T) {
	collectionLocations := map[string]string{
		"path1": "/some/path1",
		"path2": "/some/path2",
	}

	expectedPath := "/some/path1/sub/path/to/dataset"
	absPath, err := GetDatasetAbsolutePath(collectionLocations, "/path1/sub/path/to/dataset")
	if err != nil {
		t.Errorf("received error from function: %s", err.Error())
		return
	}
	if absPath != expectedPath {
		t.Errorf("path result is different from what is expected - got: '%s', want: '%s'", absPath, expectedPath)
		return
	}

	expectedPath = "/some/path2/another/path/to/dataset"
	absPath, err = GetDatasetAbsolutePath(collectionLocations, "/path2/another/path/to/dataset")
	if err != nil {
		t.Errorf("received error from function: %s", err.Error())
		return
	}
	if absPath != expectedPath {
		t.Errorf("path result is different from what is expected - got: '%s', want: '%s'", absPath, expectedPath)
		return
	}
}
