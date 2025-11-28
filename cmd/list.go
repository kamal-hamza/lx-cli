package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lx/internal/core/services"
	"lx/pkg/ui"
)

var (
	listTagFilter string
	listSortBy    string
	listReverse   bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all LaTeX notes",
	Long: `List all LaTeX notes in a table format.

Examples:
  lx list
  lx list --tag math
  lx list --sort title
  lx list --tag science --reverse`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&listTagFilter, "tag", "", "Filter notes by tag")
	listCmd.Flags().StringVar(&listSortBy, "sort", "date", "Sort by field (date, title)")
	listCmd.Flags().BoolVar(&listReverse, "reverse", false, "Reverse sort order")
}

func runList(cmd *cobra.Command, args []string) error {
	// Execute list service
	req := services.ListRequest{
		TagFilter: listTagFilter,
		SortBy:    listSortBy,
		Reverse:   listReverse,
	}

	ctx := getContext()
	resp, err := listService.Execute(ctx, req)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to list notes"))
		return err
	}

	// Handle empty results
	if resp.Total == 0 {
		if listTagFilter != "" {
			fmt.Println(ui.FormatWarning("No notes found with tag: " + listTagFilter))
		} else {
			fmt.Println(ui.FormatWarning("No notes found"))
			fmt.Println(ui.FormatInfo("Create your first note with: lx new \"My Note\""))
		}
		return nil
	}

	// Print header
	if listTagFilter != "" {
		fmt.Println(ui.FormatTitle(fmt.Sprintf("Notes (filtered by tag: %s)", listTagFilter)))
	} else {
		fmt.Println(ui.FormatTitle("Notes"))
	}
	fmt.Println()

	// Create table
	table := ui.NewTable([]ui.TableColumn{
		{Header: "Title", Width: 40, Align: "left"},
		{Header: "Date", Width: 12, Align: "left"},
		{Header: "Tags", Width: 30, Align: "left"},
		{Header: "Slug", Width: 25, Align: "left"},
	})

	// Add rows
	for _, note := range resp.Notes {
		table.AddRow([]string{
			truncate(note.Title, 40),
			note.GetDisplayDate(),
			truncate(note.GetTagsString(), 30),
			note.Slug,
		})
	}

	// Render table
	fmt.Print(table.Render())
	fmt.Println()

	// Print summary
	fmt.Println(ui.FormatMuted(fmt.Sprintf("Total: %d notes", resp.Total)))

	return nil
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
