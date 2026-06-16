// syscall_other.go
//go:build !windows

package metadataextractor

import "os/exec"

func hideWindow(cmd *exec.Cmd) {} // no-op
