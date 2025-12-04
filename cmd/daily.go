package cmd

import (
	"fmt"
	"time"

	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/config"
	"github.com/kamal-hamza/lx-cli/pkg/ui"

	"github.com/spf13/cobra"
)

var dailyCmd = &cobra.Command{
	Use:     "daily",
	Aliases: []string{"dd"},
	Short:   "Open or create today's daily note (alias: dd)",
	Long: `Open today's daily note.
If it doesn't exist, it will be created automatically with the tag 'journal'.

This is perfect for:
- Morning pages
- Daily tasks
- Quick scratchpad for thoughts`,
	RunE: runDaily,
}

func runDaily(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Load Configuration
	// We check if a specific template is configured for daily notes
	cfg, err := config.Load(appVault.ConfigPath)
	if err != nil {
		// Fallback to defaults silently if config fails to load
		cfg = config.DefaultConfig()
	}

	// 2. Generate Title: "Journal 2025-11-28"
	now := time.Now()
	title := fmt.Sprintf("Journal %s", now.Format("2006-01-02"))

	// 3. Prepare Creation Request
	// We use the configured DailyTemplate if available
	req := services.CreateNoteRequest{
		Title:        title,
		Tags:         []string{"journal"},
		TemplateName: cfg.DailyTemplate,
	}

	fmt.Println(ui.FormatRocket("Opening daily note..."))

	// 4. Try to Create Note
	// If it already exists, the service returns an error which we handle
	resp, err := createNoteService.Execute(ctx, req)

	var notePath string

	if err != nil {
		// If creation failed, check if it's because the note already exists
		searchReq := services.SearchRequest{Query: title}
		searchResp, searchErr := listService.Search(ctx, searchReq)
		if searchErr != nil {
			return searchErr
		}

		if searchResp.Total > 0 {
			// Found existing daily note
			target := searchResp.Notes[0]
			notePath = appVault.GetNotePath(target.Filename)
			fmt.Println(ui.FormatInfo("Found existing entry: " + target.Title))
		} else {
			// It was a real error (not just duplication)
			return err
		}
	} else {
		// Created successfully
		notePath = appVault.GetNotePath(resp.FilePath)
		fmt.Println(ui.FormatSuccess("Created new entry for today."))
	}

	// 5. Open in Editor
	// Uses the shared helper from cmd/utils.go which handles line numbers
	return OpenEditorAtLine(notePath, 0)
}
