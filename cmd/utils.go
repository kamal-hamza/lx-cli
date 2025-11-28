package cmd

import (
	"fmt"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
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

// OpenEditorAtLine opens the user's preferred editor at a specific line number.
func OpenEditorAtLine(path string, line int) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default fallback
	}

	var args []string
	lowerEditor := strings.ToLower(editor)

	// Strategy 1: VS Code family (Code, Cursor, Windsurf)
	// These REQUIRE the -g flag to parse line numbers: `code -g file:line`
	if strings.Contains(lowerEditor, "code") ||
		strings.Contains(lowerEditor, "cursor") ||
		strings.Contains(lowerEditor, "windsurf") {
		args = []string{"-g", fmt.Sprintf("%s:%d", path, line)}

		// Strategy 2: Sublime Text, Zed, IntelliJ/GoLand
		// These support direct `file:line` syntax without flags
	} else if strings.Contains(lowerEditor, "subl") ||
		strings.Contains(lowerEditor, "zed") || // <--- Added Zed Support
		strings.Contains(lowerEditor, "idea") ||
		strings.Contains(lowerEditor, "goland") {
		args = []string{fmt.Sprintf("%s:%d", path, line)}

		// Strategy 3: Terminal Editors (Vim, Nano, Kakoune, Emacs)
		// Standard Unix syntax: `vim +line file`
	} else {
		args = []string{fmt.Sprintf("+%d", line), path}
	}

	// Run the editor
	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Fallback: If line number fails, just open the file
		fallback := exec.Command(editor, path)
		fallback.Stdin = os.Stdin
		fallback.Stdout = os.Stdout
		fallback.Stderr = os.Stderr
		return fallback.Run()
	}

	return nil
}
