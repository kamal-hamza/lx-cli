package cmd

import (
	"fmt"
	"time"

	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var dailyCmd = &cobra.Command{
	Use:     "daily",
	Aliases: []string{"dd"},
	Short:   "Create or open today's daily note",
	Long: `Create or open today's daily note.

If a daily note for today already exists, it will be opened.
Otherwise, a new daily note will be created using the configured daily template.

Examples:
  lx daily
  lx dd`,
	RunE: runDaily,
}

func runDaily(cmd *cobra.Command, args []string) error {
	// Generate daily note title from current date
	now := time.Now()
	title := now.Format("2006-01-02") // e.g., "2023-12-14"
	tags := []string{"daily"}

	// Use configured daily template, or fall back to empty string
	templateName := appConfig.DailyTemplate

	// Create request
	req := services.CreateNoteRequest{
		Title:        title,
		Tags:         tags,
		TemplateName: templateName,
		DateFormat:   appConfig.DateFormat,
	}

	ctx := getContext()
	resp, err := createNoteService.Execute(ctx, req)
	if err != nil {
		// If note already exists, just open it
		if err.Error() == fmt.Sprintf("note with slug '%s' already exists", title) {
			fmt.Println(ui.FormatInfo("Daily note already exists, opening..."))
			return runOpenNote(cmd, []string{title})
		}
		fmt.Println(ui.FormatError("Failed to create daily note"))
		return err
	}

	// Success message
	fmt.Println(ui.FormatSuccess("Daily note created successfully!"))
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Title", resp.Note.Header.Title))
	fmt.Println(ui.RenderKeyValue("File", resp.FilePath))
	fmt.Println()

	// Get the full path
	notePath := appVault.GetNotePath(resp.FilePath)

	// Open in editor
	editor := GetPreferredEditor()
	fmt.Println(ui.FormatInfo("Opening in editor: " + editor))
	fmt.Println()

	if err := OpenEditorAtLine(notePath, 1); err != nil {
		fmt.Println(ui.FormatWarning("Failed to open editor: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually edit: " + notePath))
	}

	return nil
}

// init function removed - dailyCmd is registered in root.go
