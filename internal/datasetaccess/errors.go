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
	return fmt.Sprintf("the user does not have access to the datasets - %s", strings.Join(listErrs, ", "))
}

func newAccessError(groupsWithAccess []string, blockedGroups []string) *AccessError {
	return &AccessError{
		groupsWithAccess: groupsWithAccess,
		blockedGroups:    blockedGroups,
	}
}

type InvalidGroupsError struct {
	allowPath            string
	blockPath            string
	invalidAllowedGroups []string
	invalidBlockedGroups []string
}

func (e *InvalidGroupsError) Error() string {
	listErrs := []string{}
	if len(e.invalidAllowedGroups) > 0 {
		listErrs = append(listErrs, fmt.Sprintf("the following allowed groups at %s are invalid: %v", e.allowPath, e.invalidAllowedGroups))
	}
	if len(e.invalidBlockedGroups) > 0 {
		listErrs = append(listErrs, fmt.Sprintf("the following blocked groups at %s are invalid: '%v'", e.blockPath, e.invalidBlockedGroups))

	}
	return fmt.Sprintf("invalid groups error - %s", strings.Join(listErrs, ", "))
}

func newInvalidGroupsError(allowPath string, blockPath string, allowedGroups []string, blockedGroups []string) *InvalidGroupsError {
	return &InvalidGroupsError{
		allowPath:            allowPath,
		blockPath:            blockPath,
		invalidAllowedGroups: allowedGroups,
		invalidBlockedGroups: blockedGroups,
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
