//go:build !windows

package utils

import "os/exec"

func HideWindow(cmd *exec.Cmd) {
	// No-op
}
