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

const accessControlFilename = ".ingestor-access.yaml"

func parseAccessFile(path string) (accessFile, error) {
	rawAccessFile, err := os.ReadFile(path)
	if err != nil {
		return accessFile{}, err
	}

	var parsedAccessFile accessFile
	err = yaml.Unmarshal(rawAccessFile, &parsedAccessFile)
	return parsedAccessFile, err
}

func CheckUserAccess(ctx context.Context, path string) error {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return fmt.Errorf("internal context error")
	}

	userSession := sessions.DefaultMany(ginCtx, "user")
	userGroups, ok := userSession.Get("access_groups").([]string)
	if !ok {
		return fmt.Errorf("internal user session error: can't get access groups of user")
	}

	splitPath := strings.Split(filepath.Clean(path), string(filepath.Separator))
	allowedGroups := map[string]bool{}
	blockedGroups := map[string]bool{}
	conflictedGroups := []string{}
	invalidAllowGroups := []string{}

	// assemble allowed and blocked group sets
	for i := len(splitPath); i > 0; i-- {
		/// read and parse file
		currPath := filepath.Join(append(splitPath[0:i], accessControlFilename)...)
		if splitPath[0] == "" {
			currPath = string(filepath.Separator) + currPath // UNIX fix
		}

		if _, err := os.Stat(currPath); errors.Is(err, os.ErrNotExist) {
			continue // skip the path if it doesn't contain any access restrictions
		}

		parsedAccessFile, err := parseAccessFile(currPath)
		if err != nil {
			return err
		}

		// find groups
		if len(allowedGroups) > 0 {
			// check if lowest level allowed groups list align with upper levels (same or stricter set of groups)
			currAllowedGroups := map[string]bool{}
			for _, group := range parsedAccessFile.AllowedGroups {
				currAllowedGroups[group] = true
			}
			for group := range allowedGroups {
				if _, ok := currAllowedGroups[group]; !ok {
					invalidAllowGroups = append(invalidAllowGroups, group)
				}
			}
		} else {
			// set final list of allow groups (lowest level where it's defined)
			for _, group := range parsedAccessFile.AllowedGroups {
				allowedGroups[group] = true
			}
		}
		for _, group := range parsedAccessFile.BlockedGroups {
			blockedGroups[group] = true
		}
	}

	// check for conflicted groups
	for group := range blockedGroups {
		if _, ok := allowedGroups[group]; ok {
			conflictedGroups = append(conflictedGroups, group)
		}
	}

	if len(conflictedGroups) > 0 || len(invalidAllowGroups) > 0 {
		return newGroupError(path, invalidAllowGroups, conflictedGroups)
	}

	// check user permissions based on groups
	failedWhitelist := false
	userBlockedGroups := []string{}
	if len(allowedGroups) > 0 {
		failedWhitelist = true
	}
	for _, group := range userGroups {
		if _, ok := allowedGroups[group]; ok {
			failedWhitelist = false
			break
		}
		if _, ok := blockedGroups[group]; ok {
			userBlockedGroups = append(userBlockedGroups, group)
		}
	}

	// return error if user doesn't have the access rights
	if failedWhitelist {
		allowedGroupsList := make([]string, 0, len(allowedGroups))
		for k := range allowedGroups {
			allowedGroupsList = append(allowedGroupsList, k)
		}
		return newAccessError(allowedGroupsList, userBlockedGroups)
	} else if len(userBlockedGroups) > 0 {
		return newAccessError([]string{}, userBlockedGroups)
	}

	return nil
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

func IsDatasetFolder(path string) bool {
	accessFile, err := parseAccessFile(filepath.Join(filepath.Dir(path), accessControlFilename))
	if err == nil {
		return accessFile.HasDatasetFolders
	}
	return false
}
