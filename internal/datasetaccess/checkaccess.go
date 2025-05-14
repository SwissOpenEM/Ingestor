package datasetaccess

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

const accessControlFilename = "openem-access.yaml"

func parseAccessFile(path string) (accessFile, error) {
	rawAccessFile, err := os.ReadFile(path)
	if err != nil {
		return accessFile{}, err
	}

	var parsedAccessFile accessFile
	err = yaml.Unmarshal(rawAccessFile, &parsedAccessFile)
	return parsedAccessFile, err
}

func CheckAccessIntegrity(path string) error {
	splitPath := strings.Split(filepath.Clean(path), string(filepath.Separator))
	allFoundGroups := map[string]bool{}

	// check if the access hierarchy follows the rules by traversing parent directories
	for i := len(splitPath); i > 0; i-- {
		/// read and parse file
		path := filepath.Join(append(splitPath[0:i], accessControlFilename)...)

		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue // skip the path if it doesn't contain any access restrictions
		}

		parsedAccessFile, err := parseAccessFile(path)
		if err != nil {
			return err
		}

		// path rule check
		if parsedAccessFile.Path != path {
			return newPathError(parsedAccessFile.Path, path)
		}

		parsedGroups := map[string]bool{}
		for _, group := range parsedAccessFile.AccessGroups {
			parsedGroups[group] = true
		}

		invalidGroups := []string{}
		for group := range allFoundGroups {
			if _, ok := parsedGroups[group]; !ok {
				invalidGroups = append(invalidGroups, group)
			}
		}
		if len(invalidGroups) > 0 { // return an error in case of any mismatched groups
			return newInvalidGroupsError(path, invalidGroups)
		}

		// update map of all encountered groups
		for group := range parsedGroups {
			allFoundGroups[group] = true
		}
	}
	return nil
}

func CheckUserAccess(path string, hasGroups []string) error {
	splitPath := strings.Split(filepath.Clean(path), string(filepath.Separator))

	allowedGroups := map[string]bool{}
	hasGroupsMap := map[string]bool{}
	for _, group := range hasGroups {
		hasGroupsMap[group] = true
	}

	for i := len(splitPath); i > 0; i-- {
		/// read and parse file
		path := filepath.Join(append(splitPath[0:i], accessControlFilename)...)

		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue // skip the path if it doesn't contain any access restrictions
		}

		parsedAccessFile, err := parseAccessFile(path)
		if err != nil {
			return err
		}

		// the first file closest to the final path determines the strictest level of required groups
		for _, group := range parsedAccessFile.AccessGroups {
			allowedGroups[group] = true
		}
		break // no need to continue iterate further, we got the (theoretical) strictest set of groups
	}

	// check if user has required groups
	allowedGroupsSlice := make([]string, 0, len(allowedGroups))
	for group := range allowedGroups {
		if _, ok := hasGroupsMap[group]; ok {
			return nil
		}
		allowedGroupsSlice = append(allowedGroupsSlice, group)
	}

	return newAccessError(allowedGroupsSlice)
}
