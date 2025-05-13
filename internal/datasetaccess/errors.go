package datasetaccess

import "fmt"

type AccessError struct {
	groupsWithAccess []string
}

func (e *AccessError) Error() string {
	return fmt.Sprintf("the user does not have access to the datasets, one of the following groups is needed: %v", e.groupsWithAccess)
}

func NewAccessError(groupsWithAccess []string) *AccessError {
	return &AccessError{
		groupsWithAccess: groupsWithAccess,
	}
}

type InvalidGroupsError struct {
	currentPath   string
	invalidGroups []string
}

func (e *InvalidGroupsError) Error() string {
	return fmt.Sprintf("the following invalid groups were found when checking the path '%s': '%v'", e.currentPath, e.invalidGroups)
}

func NewInvalidGroupsError(path string, groups []string) *InvalidGroupsError {
	return &InvalidGroupsError{
		currentPath:   path,
		invalidGroups: groups,
	}
}
