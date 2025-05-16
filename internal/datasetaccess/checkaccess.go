package datasetaccess

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
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
	prevPath := ""
	for i := len(splitPath); i > 0; i-- {
		/// read and parse file
		currPath := filepath.Join(append(splitPath[0:i], accessControlFilename)...)

		if _, err := os.Stat(currPath); errors.Is(err, os.ErrNotExist) {
			continue // skip the path if it doesn't contain any access restrictions
		}

		parsedAccessFile, err := parseAccessFile(currPath)
		if err != nil {
			return err
		}

		// path rule check
		if parsedAccessFile.Path != currPath {
			return newPathError(parsedAccessFile.Path, currPath)
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
			return newInvalidGroupsError(prevPath, invalidGroups)
		}

		// update map of all encountered groups
		for group := range parsedGroups {
			allFoundGroups[group] = true
		}
		prevPath = currPath
	}
	return nil
}

func CheckUserAccess(ctx context.Context, path string) error {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return fmt.Errorf("internal context error")
	}

	userSession := sessions.DefaultMany(ginCtx, "user")
	hasGroups, ok := userSession.Get("access_groups").([]string)
	if !ok {
		return fmt.Errorf("internal user session error: can't get access groups of user")
	}

	splitPath := strings.Split(filepath.Clean(path), string(filepath.Separator))

	allowedGroups := map[string]bool{}
	hasGroupsMap := map[string]bool{}
	for _, group := range hasGroups {
		hasGroupsMap[group] = true
	}

	for i := len(splitPath); i > 0; i-- {
		/// read and parse file
		path := filepath.Join(append(splitPath[0:i], accessControlFilename)...)
		if splitPath[0] == "" {
			path = string(filepath.Separator) + path // UNIX fix
		}

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

	// if there's no allowed group found, allow all
	if len(allowedGroups) == 0 {
		return nil
	}

	// check if user has required groups
	allowedGroupsSlice := make([]string, 0, len(allowedGroups))
	for group := range allowedGroups {
		if _, ok := hasGroupsMap[group]; ok {
			return nil // if the user has at least one group from the list, they're allowed to access the dataset
		}
		allowedGroupsSlice = append(allowedGroupsSlice, group)
	}

	return newAccessError(allowedGroupsSlice)
}

func IsFolderCheck(path string) error {
	folder, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !folder.IsDir() {
		return newNotFolderError(path)
	}
	return nil
}
