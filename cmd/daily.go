package cmd

import (
	"fmt"
	"time"

	"lx/internal/core/services"
	"lx/pkg/ui"

	"github.com/spf13/cobra"
)

var dailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Open or create today's daily note",
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

	// 1. Generate Title: "Journal 2025-11-28"
	now := time.Now()
	title := fmt.Sprintf("Journal %s", now.Format("2006-01-02"))

	// 2. Check if it already exists
	// We use the slug generation logic to find the expected slug
	// "Journal 2025-11-28" -> "journal-2025-11-28"
	// Note: We search for this specific note to avoid duplicates

	// We can use the existing search/list service or just try to create it.
	// Since CreateNoteService checks for existence, we can just try to create it.
	// However, if it exists, CreateNoteService returns an error.
	// We want to handle that gracefully by opening the existing one.

	req := services.CreateNoteRequest{
		Title:        title,
		Tags:         []string{"journal"},
		TemplateName: "", // You could set a default "daily" template here if you have one
	}

	fmt.Println(ui.FormatRocket("Opening daily note..."))

	// Try to create
	resp, err := createNoteService.Execute(ctx, req)

	var notePath string

	if err != nil {
		// If error is "already exists", find the existing file
		// This relies on the specific error message from your service or a check
		// For simplicity, let's search for it.

		searchReq := services.SearchRequest{Query: title}
		searchResp, searchErr := listService.Search(ctx, searchReq)
		if searchErr != nil {
			return searchErr
		}

		if searchResp.Total > 0 {
			// Found it!
			target := searchResp.Notes[0]
			notePath = appVault.GetNotePath(target.Filename)
			fmt.Println(ui.FormatInfo("Found existing entry: " + target.Title))
		} else {
			// Real error
			return err
		}
	} else {
		// Created successfully
		notePath = appVault.GetNotePath(resp.FilePath)
		fmt.Println(ui.FormatSuccess("Created new entry for today."))
	}

	// 3. Open in Editor
	return openEditorAtLine(notePath, 0) // Reuse the function from grep.go if exported, or duplicate
}
