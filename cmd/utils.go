package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

// OpenFile opens a file using a custom viewer or the OS default application.
func OpenFile(path string, viewer string) error {
	var cmd *exec.Cmd

	if viewer != "" {
		// Use user-configured viewer (e.g. zathura, skim)
		cmd = exec.Command(viewer, path)
	} else {
		// Fallback to OS default
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", path)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", path)
		default:
			cmd = exec.Command("xdg-open", path)
		}
	}

	// We use Start() to detach the process so lx can exit while the viewer stays open
	if err := cmd.Start(); err != nil {
		if viewer != "" {
			return fmt.Errorf("failed to open '%s' with '%s': %w", path, viewer, err)
		}
		return fmt.Errorf("failed to open '%s': %w", path, err)
	}

	return nil
}

// OpenEditorAtLine opens the user's preferred editor at a specific line number.
// ... [Keep existing OpenEditorAtLine implementation] ...
func OpenEditorAtLine(path string, line int) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default fallback
	}

	var args []string
	lowerEditor := strings.ToLower(editor)

	if strings.Contains(lowerEditor, "code") ||
		strings.Contains(lowerEditor, "cursor") ||
		strings.Contains(lowerEditor, "windsurf") {
		args = []string{"-g", fmt.Sprintf("%s:%d", path, line)}
	} else if strings.Contains(lowerEditor, "subl") ||
		strings.Contains(lowerEditor, "zed") ||
		strings.Contains(lowerEditor, "idea") ||
		strings.Contains(lowerEditor, "goland") {
		args = []string{fmt.Sprintf("%s:%d", path, line)}
	} else {
		args = []string{fmt.Sprintf("+%d", line), path}
	}

	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fallback := exec.Command(editor, path)
		fallback.Stdin = os.Stdin
		fallback.Stdout = os.Stdout
		fallback.Stderr = os.Stderr
		return fallback.Run()
	}

	return nil
}

// ... [Keep checkAndInstallPandoc] ...
func checkAndInstallPandoc() error {
	if _, err := exec.LookPath("pandoc"); err == nil {
		return nil
	}

	fmt.Println()
	fmt.Print(ui.StyleWarning.Render("Pandoc not found. Install it now? (y/n): "))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil || strings.ToLower(strings.TrimSpace(response)) != "y" {
		return fmt.Errorf("missing (required for export)")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("brew", "install", "pandoc")
	case "windows":
		cmd = exec.Command("winget", "install", "Pandoc.Pandoc")
	case "linux":
		if _, err := exec.LookPath("apt-get"); err == nil {
			cmd = exec.Command("sudo", "apt-get", "install", "-y", "pandoc")
		} else if _, err := exec.LookPath("dnf"); err == nil {
			cmd = exec.Command("sudo", "dnf", "install", "-y", "pandoc")
		} else if _, err := exec.LookPath("pacman"); err == nil {
			cmd = exec.Command("sudo", "pacman", "-S", "--noconfirm", "pandoc")
		} else if _, err := exec.LookPath("zypper"); err == nil {
			cmd = exec.Command("sudo", "zypper", "install", "-y", "pandoc")
		} else {
			return fmt.Errorf("could not detect package manager. Please install pandoc manually")
		}
	default:
		return fmt.Errorf("manual installation required for %s", runtime.GOOS)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(ui.FormatInfo("Installing pandoc..."))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	return nil
}
