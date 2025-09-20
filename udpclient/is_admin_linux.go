//go:build linux
// +build linux

package main

import (
	"os/user"
)

func IsAdmin() bool {
	currentUser, err := user.Current()
	if err != nil {
		return false
	}
	// UID = 0 表示 root 用户
	return currentUser.Uid == "0"
}
