package collections

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func GetCollectionList(collectionLocations map[string]string) []string {
	locationKeys := make([]string, len(collectionLocations))
	i := 0
	for location := range collectionLocations {
		locationKeys[i] = location
		i++
	}
	sort.Strings(locationKeys)
	return locationKeys
}

// returns: collection name, collection path, the remainder of the path
func GetPathDetails(collectionLocations map[string]string, path string) (string, string, string, error) {
	splitSourceFolder := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	if len(splitSourceFolder) <= 0 {
		return "", "", "", fmt.Errorf("sourceFolder contains an invalid path")
	}

	collectionLocationPath, ok := collectionLocations[splitSourceFolder[0]]
	if !ok {
		return "", "", "", fmt.Errorf("sourceFolder contains an invalid path (invalid collection location)")
	}

	return splitSourceFolder[0], collectionLocationPath, filepath.Join(splitSourceFolder[1:]...), nil
}

// helper function that converts the given datasetFolder to a system absolute path using the 'collectionLocations' map
func GetDatasetAbsolutePath(collectionLocations map[string]string, path string) (string, error) {
	_, colPath, relPath, err := GetPathDetails(collectionLocations, path)
	if err != nil {
		return "", err
	}
	return filepath.Join(colPath, relPath), nil
}
