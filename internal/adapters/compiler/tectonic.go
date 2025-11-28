package compiler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

// TectonicCompiler implements the Compiler port using Tectonic
type TectonicCompiler struct {
	vault      *vault.Vault
	binaryPath string // Path to tectonic binary
}

// NewTectonicCompiler creates a new Tectonic-based compiler
func NewTectonicCompiler(vault *vault.Vault) *TectonicCompiler {
	return &TectonicCompiler{
		vault:      vault,
		binaryPath: "", // Will be resolved on first use
	}
}

// Compile compiles a note to PDF using Tectonic
func (c *TectonicCompiler) Compile(ctx context.Context, slug string, env []string) error {
	// Ensure Tectonic is available
	if err := c.ensureTectonic(ctx); err != nil {
		return err
	}

	// Find the source file
	sourceFile, err := c.findSourceFile(slug)
	if err != nil {
		return err
	}

	sourcePath := c.vault.GetNotePath(sourceFile)

	// Prepare Tectonic command
	// -X compile: Use the V2 CLI interface
	// --outdir: Output directory for PDF
	// --keep-logs: Keep log files for debugging
	args := []string{
		"-X", "compile",
		sourcePath,
		"--outdir", c.vault.CachePath,
		"--keep-logs",
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)

	// Set working directory to cache
	cmd.Dir = c.vault.CachePath

	// Prepare environment
	cmdEnv := os.Environ()

	// Add TEXINPUTS for template discovery
	texinputs := c.vault.GetTexInputsEnv()
	cmdEnv = append(cmdEnv, "TEXINPUTS="+texinputs)

	// Add any additional environment variables
	cmdEnv = append(cmdEnv, env...)

	cmd.Env = cmdEnv

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetOutputPath returns the path to the compiled PDF
func (c *TectonicCompiler) GetOutputPath(slug string) string {
	// Find the source file to get the base name
	sourceFile, err := c.findSourceFile(slug)
	if err != nil {
		// Fallback to constructing from slug
		return c.vault.GetCachePath(slug + ".pdf")
	}

	// Replace .tex with .pdf
	pdfName := strings.TrimSuffix(sourceFile, ".tex") + ".pdf"
	return c.vault.GetCachePath(pdfName)
}

// Clean removes auxiliary files for a specific note
func (c *TectonicCompiler) Clean(ctx context.Context, slug string) error {
	// Tectonic doesn't create as many auxiliary files as latexmk,
	// but we still need to clean up logs and the PDF

	sourceFile, err := c.findSourceFile(slug)
	if err != nil {
		return err
	}

	baseName := strings.TrimSuffix(sourceFile, ".tex")

	// Files to remove
	extensions := []string{".pdf", ".log", ".aux", ".out", ".toc"}

	for _, ext := range extensions {
		filename := baseName + ext
		path := c.vault.GetCachePath(filename)

		// Ignore errors if file doesn't exist
		os.Remove(path)
	}

	return nil
}

// ensureTectonic ensures the Tectonic binary is available
func (c *TectonicCompiler) ensureTectonic(ctx context.Context) error {
	// Already resolved
	if c.binaryPath != "" {
		return nil
	}

	// Try to find in PATH first
	if path, err := exec.LookPath("tectonic"); err == nil {
		c.binaryPath = path
		return nil
	}

	// Try to find in vault's bin directory
	localBin := filepath.Join(c.vault.RootPath, "bin", "tectonic")
	if _, err := os.Stat(localBin); err == nil {
		c.binaryPath = localBin
		return nil
	}

	// Not found - offer to install
	return c.offerInstall(ctx)
}

// offerInstall prompts the user to install Tectonic
func (c *TectonicCompiler) offerInstall(ctx context.Context) error {
	fmt.Println("⚠️  Tectonic compiler not found.")
	fmt.Println()
	fmt.Println("Tectonic is a modern, self-contained LaTeX engine that:")
	fmt.Println("  • Automatically downloads packages as needed")
	fmt.Println("  • Requires no system LaTeX installation")
	fmt.Println("  • Compiles faster than traditional engines")
	fmt.Println()
	fmt.Print("Would you like to install a portable copy? (y/n): ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "y" && response != "yes" {
		return fmt.Errorf("tectonic installation declined")
	}

	return c.installTectonic(ctx)
}

// installTectonic downloads and installs Tectonic
func (c *TectonicCompiler) installTectonic(ctx context.Context) error {
	fmt.Println()
	fmt.Println("📦 Installing Tectonic...")

	// Detect platform
	goos := runtime.GOOS

	if goos == "windows" {
		fmt.Println()
		fmt.Println("⚠️  Automatic installation is not yet supported on Windows.")
		fmt.Println("Please visit: https://tectonic-typesetting.github.io/install.html")
		return fmt.Errorf("manual installation required on Windows")
	}

	// Construct download URL
	// Tectonic provides a universal installer script
	installerURL := "https://drop-sh.fullyjustified.net"

	// Create bin directory in vault
	binDir := filepath.Join(c.vault.RootPath, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Download using the official installer
	fmt.Println("Downloading from: " + installerURL)

	// Create a temporary script
	tmpScript := filepath.Join(binDir, "install-tectonic.sh")
	defer os.Remove(tmpScript)

	// Download the installer
	resp, err := http.Get(installerURL)
	if err != nil {
		return fmt.Errorf("failed to download installer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Save the installer script
	scriptFile, err := os.Create(tmpScript)
	if err != nil {
		return fmt.Errorf("failed to create installer script: %w", err)
	}

	_, err = io.Copy(scriptFile, resp.Body)
	scriptFile.Close()
	if err != nil {
		return fmt.Errorf("failed to save installer: %w", err)
	}

	// Make script executable
	if err := os.Chmod(tmpScript, 0755); err != nil {
		return fmt.Errorf("failed to make installer executable: %w", err)
	}

	// Run the installer with custom target directory
	fmt.Println("Installing to: " + binDir)

	installCmd := exec.CommandContext(ctx, "sh", tmpScript, "--prefix", binDir)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	// Verify installation
	tectonicPath := filepath.Join(binDir, "tectonic")
	if _, err := os.Stat(tectonicPath); err != nil {
		return fmt.Errorf("installation verification failed: %w", err)
	}

	c.binaryPath = tectonicPath

	fmt.Println()
	fmt.Println("✅ Tectonic installed successfully!")
	fmt.Println()

	return nil
}

// findSourceFile finds the .tex file matching the slug
func (c *TectonicCompiler) findSourceFile(slug string) (string, error) {
	entries, err := os.ReadDir(c.vault.NotesPath)
	if err != nil {
		return "", fmt.Errorf("failed to read notes directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tex") {
			continue
		}

		// Check if filename contains the slug
		name := strings.TrimSuffix(entry.Name(), ".tex")
		if strings.HasSuffix(name, slug) || strings.Contains(name, "-"+slug) {
			return entry.Name(), nil
		}
	}

	return "", fmt.Errorf("source file not found for slug: %s", slug)
}

// IsTectonicAvailable checks if Tectonic can be used
func IsTectonicAvailable() bool {
	// Check system PATH
	if _, err := exec.LookPath("tectonic"); err == nil {
		return true
	}

	// Check for local installation
	// Note: We don't have vault context here, so we check the default location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	localBin := filepath.Join(homeDir, ".local", "share", "lx", "bin", "tectonic")
	_, err = os.Stat(localBin)
	return err == nil
}
