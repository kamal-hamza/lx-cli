package cmd

import (
	"fmt"
	"os"
	"sort"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"

	"lx/internal/core/domain"
	"lx/internal/core/services"
	"lx/pkg/ui"
)

var (
	graphDotFormat bool
	graphBacklink  bool
	graphOutgoing  bool
)

var graphCmd = &cobra.Command{
	Use:   "graph [query]",
	Short: "Browse the knowledge graph interactively",
	Long: `Browse your knowledge graph interactively in the terminal.

Interactive mode (default):
  lx graph                         # Browse all notes
  lx graph "graph-theory"          # Start from a specific note

Generate DOT file for visualization:
  lx graph --dot > graph.dot
  dot -Tpng graph.dot -o graph.png

Navigation:
  ↑↓ or j/k  - Move cursor
  Enter or l - Open selected note
  Backspace/h - Go back in history
  q or Esc   - Quit

The graph is automatically updated before viewing to ensure freshness.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().BoolVar(&graphDotFormat, "dot", false, "Output DOT format instead of interactive mode")
	graphCmd.Flags().BoolVarP(&graphBacklink, "backlink", "b", false, "Show only backlinks (DOT mode only)")
	graphCmd.Flags().BoolVarP(&graphOutgoing, "outgoing", "o", false, "Show only outgoing links (DOT mode only)")
}

func runGraph(cmd *cobra.Command, args []string) error {
	// If --dot flag is set, use DOT output mode
	if graphDotFormat {
		if len(args) > 0 {
			return runGraphConnections(cmd, args)
		}
		return runGraphVisualization(cmd, args)
	}

	// Default: Interactive mode
	if len(args) > 0 {
		return runGraphInteractive(cmd, args)
	}

	// No query - show all notes interactively
	return runGraphInteractiveAll(cmd)
}

func runGraphVisualization(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// Generate graph
	req := services.GenerateRequest{
		Format: "dot",
	}

	resp, err := graphService.Execute(ctx, req)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.FormatError("Failed to generate graph"))
		return err
	}

	// Output to stdout (can be piped to file)
	fmt.Print(resp.Output)

	return nil
}

func runGraphInteractive(cmd *cobra.Command, args []string) error {
	ctx := getContext()
	query := args[0]

	// Load or ensure index exists
	var index *domain.Index
	var err error

	if !indexerService.IndexExists() {
		fmt.Println(ui.FormatInfo("Index not found. Building..."))
		if _, err := indexerService.Execute(ctx, services.ReindexRequest{}); err != nil {
			return err
		}
	}

	index, err = indexerService.LoadIndex()
	if err != nil {
		return err
	}

	// Search for the note
	searchReq := services.SearchRequest{Query: query}
	searchResp, err := listService.Search(ctx, searchReq)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to search notes"))
		return err
	}

	if searchResp.Total == 0 {
		fmt.Println(ui.FormatWarning("No notes found matching: " + query))
		return nil
	}

	// Select note
	var selectedNote *domain.NoteHeader
	if searchResp.Total == 1 {
		selectedNote = &searchResp.Notes[0]
	} else {
		idx, err := fuzzyfinder.Find(
			searchResp.Notes,
			func(i int) string {
				return searchResp.Notes[i].Title
			},
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				note := searchResp.Notes[i]
				entry, exists := index.GetNote(note.Slug)
				preview := fmt.Sprintf("Title: %s\nSlug: %s\nDate: %s",
					note.Title, note.Slug, note.GetDisplayDate())
				if exists {
					preview += fmt.Sprintf("\n\nConnections: %d",
						len(entry.Backlinks)+len(entry.OutgoingLinks))
					preview += fmt.Sprintf("\nBacklinks: %d", len(entry.Backlinks))
					preview += fmt.Sprintf("\nOutgoing: %d", len(entry.OutgoingLinks))
				}
				return preview
			}),
		)
		if err != nil {
			fmt.Println(ui.FormatInfo("Operation cancelled."))
			return nil
		}
		selectedNote = &searchResp.Notes[idx]
	}

	// Launch interactive viewer
	viewer, err := NewInteractiveGraphView(index, selectedNote.Slug)
	if err != nil {
		return fmt.Errorf("failed to create viewer: %w", err)
	}

	return viewer.Run()
}

func runGraphInteractiveAll(cmd *cobra.Command) error {
	ctx := getContext()

	// Load or ensure index exists
	var index *domain.Index
	var err error

	if !indexerService.IndexExists() {
		fmt.Println(ui.FormatInfo("Index not found. Building..."))
		if _, err := indexerService.Execute(ctx, services.ReindexRequest{}); err != nil {
			return err
		}
	}

	index, err = indexerService.LoadIndex()
	if err != nil {
		return err
	}

	// Get all notes sorted by connections
	type noteWithConnections struct {
		slug        string
		title       string
		connections int
	}

	var notes []noteWithConnections
	for slug, entry := range index.Notes {
		notes = append(notes, noteWithConnections{
			slug:        slug,
			title:       entry.Title,
			connections: len(entry.Backlinks) + len(entry.OutgoingLinks),
		})
	}

	// Sort by most connected
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].connections > notes[j].connections
	})

	if len(notes) == 0 {
		fmt.Println(ui.FormatWarning("No notes found in vault"))
		return nil
	}

	// Let user pick starting note
	idx, err := fuzzyfinder.Find(
		notes,
		func(i int) string {
			return fmt.Sprintf("%s (%d connections)", notes[i].title, notes[i].connections)
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			note := notes[i]
			entry, _ := index.GetNote(note.slug)
			preview := fmt.Sprintf("Title: %s\nSlug: %s\n\nTotal Connections: %d\nBacklinks: %d\nOutgoing: %d",
				entry.Title, note.slug, note.connections, len(entry.Backlinks), len(entry.OutgoingLinks))

			if len(entry.Backlinks) > 0 {
				preview += "\n\nTop Backlinks:"
				for i, bl := range entry.Backlinks {
					if i >= 5 {
						break
					}
					blEntry, exists := index.GetNote(bl)
					if exists {
						preview += "\n  ← " + blEntry.Title
					}
				}
			}

			return preview
		}),
	)
	if err != nil {
		fmt.Println(ui.FormatInfo("Operation cancelled."))
		return nil
	}

	// Launch interactive viewer
	viewer, err := NewInteractiveGraphView(index, notes[idx].slug)
	if err != nil {
		return fmt.Errorf("failed to create viewer: %w", err)
	}

	return viewer.Run()
}

func runGraphConnections(cmd *cobra.Command, args []string) error {
	ctx := getContext()
	query := args[0]

	// Search for the note
	searchReq := services.SearchRequest{
		Query: query,
	}

	searchResp, err := listService.Search(ctx, searchReq)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to search notes"))
		return err
	}

	if searchResp.Total == 0 {
		fmt.Println(ui.FormatWarning("No notes found matching: " + query))
		return nil
	}

	// Select note
	var selectedNote *domain.NoteHeader
	if searchResp.Total == 1 {
		selectedNote = &searchResp.Notes[0]
	} else {
		// Use fuzzy finder for selection
		idx, err := fuzzyfinder.Find(
			searchResp.Notes,
			func(i int) string {
				return searchResp.Notes[i].Title
			},
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				note := searchResp.Notes[i]
				return fmt.Sprintf("Title: %s\nSlug: %s\nDate: %s",
					note.Title,
					note.Slug,
					note.GetDisplayDate())
			}),
		)
		if err != nil {
			fmt.Println(ui.FormatInfo("Operation cancelled."))
			return nil
		}
		selectedNote = &searchResp.Notes[idx]
	}

	// Get connections
	var backlinks []string
	var outgoingLinks []string

	if !graphOutgoing {
		backlinks, err = graphService.GetBacklinks(ctx, selectedNote.Slug)
		if err != nil {
			return err
		}
	}

	if !graphBacklink {
		outgoingLinks, err = graphService.GetOutgoingLinks(ctx, selectedNote.Slug)
		if err != nil {
			return err
		}
	}

	// Display results
	fmt.Println(ui.FormatTitle("Connections for: " + selectedNote.Title))
	fmt.Println()

	if !graphOutgoing {
		fmt.Println(ui.StyleHeader.Render("← Backlinks") + ui.StyleMuted.Render(fmt.Sprintf(" (%d)", len(backlinks))))
		if len(backlinks) == 0 {
			fmt.Println(ui.FormatMuted("  No notes link to this note"))
		} else {
			for _, slug := range backlinks {
				fmt.Printf("  %s %s\n", ui.StyleAccent.Render("←"), slug)
			}
		}
		fmt.Println()
	}

	if !graphBacklink {
		fmt.Println(ui.StyleHeader.Render("→ Outgoing Links") + ui.StyleMuted.Render(fmt.Sprintf(" (%d)", len(outgoingLinks))))
		if len(outgoingLinks) == 0 {
			fmt.Println(ui.FormatMuted("  This note doesn't link to others"))
		} else {
			for _, slug := range outgoingLinks {
				fmt.Printf("  %s %s\n", ui.StyleAccent.Render("→"), slug)
			}
		}
		fmt.Println()
	}

	// Show quick stats
	totalConnections := len(backlinks) + len(outgoingLinks)
	if totalConnections > 0 {
		fmt.Println(ui.FormatMuted(fmt.Sprintf("Total connections: %d", totalConnections)))
	}

	return nil
}
