package datasetaccess

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

const accessControlFilename = "openem-access.yaml"

func CheckAccess(path string, hasGroups []string) error {
	splitPath := strings.Split(filepath.Clean(path), string(filepath.Separator))

	allowedGroups := map[string]bool{}
	hasGroupsMap := map[string]bool{}
	for _, group := range hasGroups {
		hasGroupsMap[group] = true
	}

	firstFile := true
	for i := len(splitPath) - 1; i >= 0; i-- {
		path := filepath.Join(append(splitPath[0:i], accessControlFilename)...)
		rawAccessFile, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var parsedAccessFile accessFile
		yaml.Unmarshal(rawAccessFile, &parsedAccessFile)
		if firstFile {
			firstFile = false
			for _, group := range parsedAccessFile.AccessGroups {
				allowedGroups[group] = true
			}
		} else {
			invalidGroups := []string{}
			for _, group := range parsedAccessFile.AccessGroups {
				if _, ok := allowedGroups[group]; !ok {
					invalidGroups = append(invalidGroups, group)
				}
			}
			if len(invalidGroups) > 0 {
				return NewInvalidGroupsError(path, invalidGroups)
			}
		}
	}

	allowedGroupsSlice := make([]string, 0, len(allowedGroups))
	for group := range allowedGroups {
		if _, ok := hasGroupsMap[group]; ok {
			return nil
		}
		allowedGroupsSlice = append(allowedGroupsSlice, group)
	}

	return NewAccessError(allowedGroupsSlice)
}
