package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/internal/adapters/compiler"
	"github.com/kamal-hamza/lx-cli/internal/adapters/repository"
	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/config"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

var (
	// Global vault instance
	appVault  *vault.Vault
	appConfig *config.Config

	// Services
	createNoteService     *services.CreateNoteService
	createTemplateService *services.CreateTemplateService
	buildService          *services.BuildService
	listService           *services.ListService
	indexerService        *services.IndexerService
	graphService          *services.GraphService
	grepService           *services.GrepService

	preprocessor *services.Preprocessor

	// Repositories
	noteRepo     *repository.FileRepository
	templateRepo *repository.TemplateRepository
	assetRepo    *repository.FileAssetRepository

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
	Args:              cobra.ArbitraryArgs,
	PersistentPreRunE: initializeApp,
	RunE:              runSmartEntry,
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
	rootCmd.AddCommand(purgeCmd)
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
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(todoCmd)
	rootCmd.AddCommand(reindexCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(assetsCmd)
	rootCmd.AddCommand(exportAllCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(aliasCmd)

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

	// Load Configuration
	cfg, err := config.Load(appVault.ConfigPath)
	if err != nil {
		// If config is corrupt or fails to load, warn but proceed with defaults
		// config.Load already handles missing files by returning default
		fmt.Println(ui.FormatWarning("Failed to load config: " + err.Error()))
		fmt.Println(ui.FormatMuted("Using default settings."))
		cfg = config.DefaultConfig()
	}
	appConfig = cfg

	// Apply UI Theme
	ui.SetTheme(appConfig.ColorTheme)

	// Check if vault exists (skip for purge command)
	if cmd.Name() != "purge" && !appVault.Exists() {
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
	templateRepo = repository.NewTemplateRepository(appVault, appConfig.CustomTemplateDir)
	assetRepo = repository.NewFileAssetRepository(appVault)

	// Initialize compiler with config
	latexCompiler = compiler.NewLatexmkCompiler(appVault, appConfig)

	// Initialize Preprocessor with caching config
	preprocessor = services.NewPreprocessor(noteRepo, appVault, appConfig.EnableCache, appConfig.CacheExpirationMinutes)

	// Initialize Git service
	gitService := services.NewGitService(appVault.RootPath)

	// Initialize services
	createNoteService = services.NewCreateNoteService(noteRepo, templateRepo, gitService, appConfig)
	createTemplateService = services.NewCreateTemplateService(templateRepo)
	buildService = services.NewBuildServiceWithPreprocessor(noteRepo, latexCompiler, preprocessor, appVault)
	listService = services.NewListService(noteRepo)
	indexerService = services.NewIndexerService(noteRepo, appVault.IndexPath())
	graphService = services.NewGraphService(noteRepo, appConfig)
	grepService = services.NewGrepService(appVault.RootPath, appConfig.GrepCaseSensitive, appConfig.MaxSearchResults)

	return nil
}

// getContext returns a context for operations
func getContext() context.Context {
	return context.Background()
}

// runSmartEntry handles smart entry when lx is called with arbitrary arguments
func runSmartEntry(cmd *cobra.Command, args []string) error {
	// If no arguments provided, launch dashboard
	if len(args) == 0 {
		return runDashboard(cmd, args)
	}

	// Load config first to check for aliases
	cfg, err := loadConfig()
	if err != nil {
		// If config fails to load, continue with default behavior
		cfg = nil
	}

	// Check if first argument is an alias
	if cfg != nil && len(args) > 0 {
		cmdName := args[0]
		remainingArgs := args[1:]

		if expandedArgs, isAlias := TryResolveAlias(cfg, cmdName, remainingArgs); isAlias {
			// Execute the expanded alias command
			return executeAliasCommand(cmd, expandedArgs)
		}
	}

	// Join all arguments as a search query
	query := args[0]
	if len(args) > 1 {
		// Join multiple arguments with spaces
		query = ""
		for i, arg := range args {
			if i > 0 {
				query += " "
			}
			query += arg
		}
	}

	// Execute the default action
	if cfg != nil {
		return runSmartAction(cmd, []string{query}, cfg.DefaultAction)
	}
	return runSmartAction(cmd, []string{query}, "open")
}

// runSmartAction executes either open or edit based on the action parameter
func runSmartAction(cmd *cobra.Command, args []string, action string) error {
	switch action {
	case "edit":
		return runEditNote(cmd, args)
	case "open":
		return runOpenNote(cmd, args)
	default:
		// Fallback to open if invalid action
		return runOpenNote(cmd, args)
	}
}

// loadConfig loads the configuration file
func loadConfig() (*config.Config, error) {
	if appVault == nil {
		return nil, fmt.Errorf("vault not initialized")
	}
	return config.Load(appVault.ConfigPath)
}

// executeAliasCommand executes an expanded alias command by finding and running the appropriate subcommand
func executeAliasCommand(cmd *cobra.Command, expandedArgs []string) error {
	if len(expandedArgs) == 0 {
		return fmt.Errorf("invalid alias expansion: no command")
	}

	// The first element is the command name, the rest are arguments
	cmdName := expandedArgs[0]
	cmdArgs := expandedArgs[1:]

	// Find the subcommand from the parent command to avoid initialization cycle
	subCmd, _, err := cmd.Root().Find([]string{cmdName})
	if err != nil {
		return fmt.Errorf("alias command not found: %s", cmdName)
	}

	// If it's the root command itself, it means the command wasn't found
	if subCmd == cmd.Root() {
		return fmt.Errorf("unknown command in alias: %s", cmdName)
	}

	// Execute the subcommand
	subCmd.SetArgs(cmdArgs)
	return subCmd.Execute()
}
