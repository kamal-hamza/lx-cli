package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	migrateDryRun bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate notes from \\ref{} to \\lxnote{}",
	Long: `Convert old \ref{slug} note references to the new \lxnote{slug} syntax.

This command scans all notes and converts \ref{} commands that reference
other notes to the new \lxnote{} syntax. Standard LaTeX \ref{} commands
(like references to figures, equations, etc.) are left unchanged.

Only \ref{} commands that match known note slugs will be converted.

Examples:
  lx migrate              # Convert all notes
  lx migrate --dry-run    # Preview changes without modifying files`,
	Run: runMigrate,
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Preview changes without modifying files")
}

func runMigrate(cmd *cobra.Command, args []string) {
	fmt.Println(ui.FormatTitle("🔄 Migrating Note References"))
	fmt.Println()

	// Get all note headers to build slug map
	headers, err := noteRepo.ListHeaders(getContext())
	if err != nil {
		fmt.Println(ui.FormatError("Failed to list notes: " + err.Error()))
		os.Exit(1)
	}

	// Build slug map for lookup
	slugMap := make(map[string]bool)
	for _, h := range headers {
		slugMap[h.Slug] = true
	}

	if len(slugMap) == 0 {
		fmt.Println(ui.FormatInfo("No notes found in vault"))
		return
	}

	fmt.Printf("Found %d notes in vault\n", len(slugMap))
	fmt.Println()

	// Process each note
	totalFiles := 0
	totalConversions := 0
	refRegex := regexp.MustCompile(`\\ref\{([^}]+)\}`)

	for _, header := range headers {
		notePath := appVault.GetNotePath(header.Filename)

		// Read file content
		content, err := os.ReadFile(notePath)
		if err != nil {
			fmt.Printf("%s Failed to read %s: %v\n", ui.FormatError("✘"), header.Slug, err)
			continue
		}

		originalContent := string(content)
		modifiedContent := originalContent
		conversions := 0

		// Find all \ref{} commands and convert those that match note slugs
		modifiedContent = refRegex.ReplaceAllStringFunc(originalContent, func(match string) string {
			submatch := refRegex.FindStringSubmatch(match)
			if len(submatch) < 2 {
				return match
			}

			targetSlug := submatch[1]

			// Only convert if it matches a known note slug
			if slugMap[targetSlug] {
				conversions++
				return fmt.Sprintf(`\lxnote{%s}`, targetSlug)
			}

			// Leave standard LaTeX refs alone
			return match
		})

		// Skip files with no changes
		if conversions == 0 {
			continue
		}

		totalFiles++
		totalConversions += conversions

		if migrateDryRun {
			fmt.Printf("%s %s (%d reference%s)\n",
				ui.StyleMuted.Render("○"),
				header.Slug,
				conversions,
				pluralize(conversions))

			// Show diff preview
			lines := strings.Split(originalContent, "\n")
			modifiedLines := strings.Split(modifiedContent, "\n")

			for i, line := range lines {
				if i < len(modifiedLines) && line != modifiedLines[i] {
					fmt.Printf("  %s %s\n", ui.StyleMuted.Render("-"), ui.StyleError.Render(line))
					fmt.Printf("  %s %s\n", ui.StyleMuted.Render("+"), ui.StyleSuccess.Render(modifiedLines[i]))
				}
			}
			fmt.Println()
		} else {
			// Write changes
			if err := os.WriteFile(notePath, []byte(modifiedContent), 0644); err != nil {
				fmt.Printf("%s Failed to write %s: %v\n", ui.FormatError("✘"), header.Slug, err)
				continue
			}

			fmt.Printf("%s %s (%d reference%s)\n",
				ui.FormatSuccess("✔"),
				header.Slug,
				conversions,
				pluralize(conversions))
		}
	}

	fmt.Println()

	if migrateDryRun {
		fmt.Println(ui.FormatInfo(fmt.Sprintf("Would update %d file%s with %d conversion%s",
			totalFiles,
			pluralize(totalFiles),
			totalConversions,
			pluralize(totalConversions))))
		fmt.Println(ui.FormatInfo("Run without --dry-run to apply changes"))
	} else {
		if totalFiles > 0 {
			fmt.Println(ui.FormatSuccess(fmt.Sprintf("✨ Updated %d file%s with %d conversion%s",
				totalFiles,
				pluralize(totalFiles),
				totalConversions,
				pluralize(totalConversions))))
			fmt.Println()
			fmt.Println(ui.FormatInfo("Run 'lx reindex' to update the knowledge graph"))
		} else {
			fmt.Println(ui.FormatInfo("No migrations needed - all references are up to date!"))
		}
	}
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
