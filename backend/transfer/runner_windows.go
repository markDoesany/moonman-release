//go:build windows

package transfer

import (
	"os/exec"
	"syscall"
)

// configureCommand prevents Dali from opening a second console window. Its
// stdout and stderr are still captured by service.go and shown in the UI.
func configureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
