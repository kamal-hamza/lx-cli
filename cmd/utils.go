package cmd

import (
	"fmt"
	"lx/pkg/ui"
	"os"
	"os/exec"
	"runtime"
	"strings"
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

func checkAndInstallPandoc() error {
	// 1. Check if installed
	if _, err := exec.LookPath("pandoc"); err == nil {
		return nil
	}

	// 2. Offer to install
	fmt.Println() // Spacing
	fmt.Print(ui.StyleWarning.Render("Pandoc not found. Install it now? (y/n): "))

	var response string
	fmt.Scanln(&response)

	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		return fmt.Errorf("missing (required for export)")
	}

	// 3. Determine Installer
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("brew", "install", "pandoc")
	case "windows":
		cmd = exec.Command("winget", "install", "Pandoc.Pandoc")
	case "linux":
		// Assuming Debian/Ubuntu, strictly speaking we should check distro
		cmd = exec.Command("sudo", "apt-get", "install", "-y", "pandoc")
	default:
		return fmt.Errorf("manual installation required for %s", runtime.GOOS)
	}

	// 4. Run Installer (Interactive)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(ui.FormatInfo("Installing pandoc..."))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	return nil
}
