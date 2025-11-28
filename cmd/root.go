package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lx/internal/adapters/compiler"
	"lx/internal/adapters/repository"
	"lx/internal/core/services"
	"lx/pkg/ui"
	"lx/pkg/vault"
)

var (
	// Global vault instance
	appVault *vault.Vault

	// Services
	createNoteService     *services.CreateNoteService
	createTemplateService *services.CreateTemplateService
	buildService          *services.BuildService
	listService           *services.ListService
	indexerService        *services.IndexerService
	graphService          *services.GraphService
	grepService           *services.GrepService

	// Repositories
	noteRepo     *repository.FileRepository
	templateRepo *repository.TemplateRepository

	// Compiler
	latexCompiler *compiler.LatexmkCompiler
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lx",
	Short: "LX - A beautiful LaTeX notes manager",
	Long: ui.StyleTitle.Render("LX") + " - LaTeX Notes Manager\n\n" +
		"A high-performance, opinionated CLI for managing LaTeX notes.\n" +
		"Treat your notes as data while abstracting away file management complexity.",
	PersistentPreRunE: initializeApp,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(buildAllCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(gitCmd)
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(tagCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(grepCmd)
	rootCmd.AddCommand(dailyCmd)
	rootCmd.AddCommand(linksCmd)
	rootCmd.AddCommand(exploreCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(attachCmd)

	// Global flags can be added here if needed
}

// initializeApp initializes the application components
func initializeApp(cmd *cobra.Command, args []string) error {
	// Skip initialization for init command
	if cmd.Name() == "init" {
		return nil
	}

	// Create vault instance
	v, err := vault.New()
	if err != nil {
		return fmt.Errorf("failed to initialize vault: %w", err)
	}
	appVault = v

	// Check if vault exists
	if !appVault.Exists() {
		fmt.Println(ui.FormatError("Vault not initialized"))
		fmt.Println(ui.FormatInfo("Run 'lx init' to initialize the vault"))
		os.Exit(1)
	}

	// Check if latexmk is available (for build commands)
	if cmd.Name() == "build" || cmd.Name() == "build-all" {
		if !compiler.IsAvailable() {
			fmt.Println(ui.FormatError("latexmk not found"))
			fmt.Println(ui.FormatInfo("Please install LaTeX and latexmk to use build commands"))
			os.Exit(1)
		}
	}

	// Initialize repositories
	noteRepo = repository.NewFileRepository(appVault)
	templateRepo = repository.NewTemplateRepository(appVault)

	// Initialize compiler
	latexCompiler = compiler.NewLatexmkCompiler(appVault)

	// Initialize services
	createNoteService = services.NewCreateNoteService(noteRepo, templateRepo)
	createTemplateService = services.NewCreateTemplateService(templateRepo)
	buildService = services.NewBuildService(noteRepo, latexCompiler)
	listService = services.NewListService(noteRepo)
	indexerService = services.NewIndexerService(noteRepo, appVault.IndexPath())
	graphService = services.NewGraphService(noteRepo, appVault.RootPath)
	grepService = services.NewGrepService(appVault.RootPath)

	return nil
}

// getContext returns a context for operations
func getContext() context.Context {
	return context.Background()
}
