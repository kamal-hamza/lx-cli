package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

var (
	listTagFilter string
	listSortBy    string
	listReverse   bool
	listTemplates bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all LaTeX notes or templates",
	Aliases: []string{"ls"},
	Long: `List all LaTeX notes in a table format, or list templates.

Examples:
  # List notes
  lx list
  lx list --tag math
  lx list --sort title
  lx list --tag science --reverse

  # List templates
  lx list -t
  lx list --template`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&listTagFilter, "tag", "", "Filter notes by tag")
	// Sort defaults to "date", but we handle config override in runListNotes
	listCmd.Flags().StringVar(&listSortBy, "sort", "date", "Sort by field (date, title)")
	listCmd.Flags().BoolVar(&listReverse, "reverse", false, "Reverse sort order")
	listCmd.Flags().BoolVarP(&listTemplates, "template", "t", false, "List templates instead of notes")
}

func runList(cmd *cobra.Command, args []string) error {
	// Check if template flag is set
	if listTemplates {
		return runListTemplates(cmd, args)
	}

	// Otherwise, list notes
	return runListNotes(cmd, args)
}

func runListNotes(cmd *cobra.Command, args []string) error {
	// Determine sort order
	// If the flag was NOT changed by the user, use the config default
	if !cmd.Flags().Changed("sort") {
		listSortBy = appConfig.DefaultSort
	}
	// If the flag was NOT changed by the user, use the config default
	if !cmd.Flags().Changed("reverse") {
		listReverse = appConfig.ReverseSort
	}

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

func runListTemplates(cmd *cobra.Command, args []string) error {
	ctx := getContext()
	templates, err := templateRepo.List(ctx)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to list templates"))
		return err
	}

	if len(templates) == 0 {
		fmt.Println(ui.FormatWarning("No templates found"))
		fmt.Println(ui.FormatInfo("Create a template with: lx new template \"title\""))
		return nil
	}

	// Display templates
	fmt.Println(ui.FormatTitle(fmt.Sprintf("Templates (%d)", len(templates))))
	fmt.Println()

	for _, template := range templates {
		fmt.Printf("%s %s\n",
			ui.StyleAccent.Render("•"),
			ui.StyleBold.Render(template.Name))
	}
	fmt.Println()

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
