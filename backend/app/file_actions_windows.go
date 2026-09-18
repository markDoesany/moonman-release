//go:build windows

package app

import "os/exec"

func releaseFolderCommand(path string) (*exec.Cmd, error) {
	return exec.Command("explorer.exe", path), nil
}
