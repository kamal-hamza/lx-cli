package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenFileWithDefaultApp opens a file using the OS default application.
// It handles macOS (open), Windows (start), and Linux (xdg-open).
func OpenFileWithDefaultApp(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS
		cmd = exec.Command("open", path)
	case "windows":
		// Windows (needs cmd /c start to detach properly)
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		// Linux / Unix / FreeBSD
		cmd = exec.Command("xdg-open", path)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open '%s': %w", path, err)
	}

	return nil
}
