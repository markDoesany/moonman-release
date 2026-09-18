//go:build !windows

package app

import (
	"fmt"
	"os/exec"
	"runtime"
)

func releaseFolderCommand(path string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path), nil
	case "linux":
		return exec.Command("xdg-open", path), nil
	default:
		return nil, fmt.Errorf("opening folders is not supported on %s", runtime.GOOS)
	}
}
