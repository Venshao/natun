//go:build windows
// +build windows

package main

import (
	"os/exec"
	"syscall"
)

func IsAdmin() bool {
	cmd := exec.Command("net", "session")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
