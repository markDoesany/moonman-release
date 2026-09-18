//go:build !windows

package transfer

import "os/exec"

// configureCommand keeps the transfer process attached to the current
// terminal on non-Windows platforms, where there is no extra console window.
func configureCommand(_ *exec.Cmd) {}
