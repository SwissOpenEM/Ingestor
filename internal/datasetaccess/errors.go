package datasetaccess

import (
	"fmt"
	"strings"
)

type AccessError struct {
	groupsWithAccess []string
	blockedGroups    []string
}

func (e *AccessError) Error() string {
	listErrs := []string{}
	if len(e.groupsWithAccess) > 0 {
		listErrs = append(listErrs, fmt.Sprintf("one of the following groups is needed: %v", e.groupsWithAccess))
	}
	if len(e.blockedGroups) > 0 {
		listErrs = append(listErrs, fmt.Sprintf("the following groups to which the user belongs are blocked from accessing it: %v", e.blockedGroups))
	}
	return fmt.Sprintf("the user does not have access to the path - %s", strings.Join(listErrs, ", "))
}

func newAccessError(groupsWithAccess []string, blockedGroups []string) *AccessError {
	return &AccessError{
		groupsWithAccess: groupsWithAccess,
		blockedGroups:    blockedGroups,
	}
}

type GroupError struct {
	path               string
	invalidAllowGroups []string
	conflictedGroups   []string
}

func (e *GroupError) Error() string {
	errList := []string{}
	if len(e.invalidAllowGroups) > 0 {
		errList = append(errList, fmt.Sprintf("invalid allow groups found on a lower level: %v", e.invalidAllowGroups))
	}
	if len(e.conflictedGroups) > 0 {
		errList = append(errList, fmt.Sprintf("the following groups are in conflict: %v", e.conflictedGroups))
	}
	return "the following group errors occured at '" + e.path + "': " + strings.Join(errList, ", ")
}

func newGroupError(path string, invalidAllowGroups []string, conflictedGroups []string) *GroupError {
	return &GroupError{
		path:               path,
		invalidAllowGroups: invalidAllowGroups,
		conflictedGroups:   conflictedGroups,
	}
}

type PathError struct {
	yamlPath   string
	actualPath string
}

func (e *PathError) Error() string {
	return fmt.Sprintf("the path indicated in the '%s' access file is different from the actual path - got: '%s', wanted: '%s'", accessControlFilename, e.yamlPath, e.actualPath)
}

func newPathError(yamlPath string, actualPath string) *PathError {
	return &PathError{
		yamlPath:   yamlPath,
		actualPath: actualPath,
	}
}

type NotFolderError struct {
	path string
}

func (e *NotFolderError) Error() string {
	return "the path at '" + e.path + "' is not a folder"
}

func newNotFolderError(path string) *NotFolderError {
	return &NotFolderError{path: path}
}
