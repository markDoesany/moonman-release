package app

import (
	"fmt"
)

// openReleaseFolderNative is split by platform so the Wails API never treats
// a local directory as a browser URL.
func openReleaseFolderNative(path string) error {
	cmd, err := releaseFolderCommand(path)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open release folder in the file manager: %w", err)
	}
	return nil
}
