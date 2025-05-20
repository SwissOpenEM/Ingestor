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
	allFoundAllowedGroups := map[string]bool{}
	lowestLevelBlockedGroups := map[string]bool{}

	// check if the access hierarchy follows the rules by traversing parent directories
	prevPath := ""
	encounteredAccessFile := false
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

		// path rule check
		if parsedAccessFile.Path != currPath {
			return newPathError(parsedAccessFile.Path, currPath)
		}

		// find invalid allowed groups
		parsedAllowedGroups := map[string]bool{}
		for _, group := range parsedAccessFile.AllowedGroups {
			parsedAllowedGroups[group] = true
		}

		// check if the upper levels define an allowed groups list while the lower levels don't
		invalidAllowedGroups := []string{}
		if len(allFoundAllowedGroups) == 0 && len(parsedAllowedGroups) > 0 && encounteredAccessFile {
			// if yes, add all discovered groups to the invalid list
			for group := range parsedAllowedGroups {
				invalidAllowedGroups = append(invalidAllowedGroups, group)
			}
		} else { // otherwise find groups that only appeared on the lower levels
			for group := range allFoundAllowedGroups {
				if _, ok := parsedAllowedGroups[group]; !ok {
					invalidAllowedGroups = append(invalidAllowedGroups, group)
				}
			}
		}

		// find invalid blocked groups (ones that only appeared on the upper levels of the path)
		invalidBlockedGroups := []string{}
		if len(lowestLevelBlockedGroups) == 0 {
			for _, group := range parsedAccessFile.BlockedGroups {
				lowestLevelBlockedGroups[group] = true
			}
		} else {
			for _, group := range parsedAccessFile.BlockedGroups {
				if _, ok := lowestLevelBlockedGroups[group]; !ok {
					invalidBlockedGroups = append(invalidBlockedGroups, group)
				}
			}
		}

		// return an error in case of any mismatched groups
		if len(invalidAllowedGroups) > 0 || len(invalidBlockedGroups) > 0 {
			return newInvalidGroupsError(prevPath, currPath, invalidAllowedGroups, invalidBlockedGroups)
		}

		// update map of all encountered allowed groups
		for group := range parsedAllowedGroups {
			allFoundAllowedGroups[group] = true
		}
		prevPath = currPath
		encounteredAccessFile = true
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
	blockedGroups := map[string]bool{}
	userGroupsMap := map[string]bool{}
	for _, group := range hasGroups {
		userGroupsMap[group] = true
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

		// the first file closest to the final path determines the strictest level of allowed and blocked groups
		for _, group := range parsedAccessFile.AllowedGroups {
			allowedGroups[group] = true
		}
		for _, group := range parsedAccessFile.BlockedGroups {
			blockedGroups[group] = true
		}
		break // no need to continue iterate further, we got the (theoretical) strictest set of groups
	}

	// if there's no allowed group found, allow all
	if len(allowedGroups) == 0 {
		return nil
	}

	// check if user has required groups
	failedWhitelist := true
	for group := range allowedGroups {
		if _, ok := userGroupsMap[group]; ok {
			failedWhitelist = false
			break // if the user has at least one group from the list, continue
		}
	}

	// check if user should be blocked based on group membership
	userBlockedGroupsSlice := []string{}
	for group := range blockedGroups {
		if _, ok := userGroupsMap[group]; ok {
			userBlockedGroupsSlice = append(userBlockedGroupsSlice, group)
		}
	}

	// return error if found, otherwise return nil
	if failedWhitelist || len(userBlockedGroupsSlice) > 0 {
		allowedGroupsSlice := make([]string, 0, len(allowedGroups))
		if failedWhitelist {
			for k := range allowedGroups {
				allowedGroupsSlice = append(allowedGroupsSlice, k)
			}
		}
		return newAccessError(allowedGroupsSlice, userBlockedGroupsSlice)
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
