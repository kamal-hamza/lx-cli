package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kamal-hamza/lx-cli/pkg/ui"
	"github.com/spf13/cobra"

	"github.com/atotto/clipboard"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

var (
	assetsGallery bool
)

// AssetSearchResult represents a unified result for display
type AssetSearchResult struct {
	Filename    string
	Description string
	UsedIn      []string // Slugs of notes using this asset
	Path        string
	ModTime     time.Time
}

var assetsCmd = &cobra.Command{
	Use:   "assets [query]",
	Short: "Search for attachments by context or description",
	Long: `Search for assets using:
1. Metadata: Matches filename, original name, or description.
2. Context: Finds notes matching the query and lists their images.

If no query is provided, opens an interactive fuzzy finder.
Use --gallery to view thumbnails in a browser.`,
	RunE: runAssets,
}

func init() {
	assetsCmd.Flags().BoolVarP(&assetsGallery, "gallery", "g", false, "View visual gallery")
	rootCmd.AddCommand(assetsCmd)
}

func runAssets(cmd *cobra.Command, args []string) error {
	ctx := getContext()
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	// Load Index
	index, err := indexerService.LoadIndex()
	if err != nil {
		return fmt.Errorf("failed to load index (try running 'lx reindex'): %w", err)
	}

	// Matched assets set (by filename)
	matchedAssets := make(map[string]bool)

	// 1. Metadata Search (using AssetRepo)
	if query != "" {
		metaMatches, err := assetRepo.Search(ctx, query)
		if err == nil {
			for _, m := range metaMatches {
				matchedAssets[m.Filename] = true
			}
		}
	}

	// 2. Context Search (using Grep + Index)
	if query != "" {
		// Find notes containing the text
		grepMatches, _ := grepService.Execute(ctx, query)
		for _, m := range grepMatches {
			// Find assets used in those notes
			if note, exists := index.GetNote(m.Slug); exists {
				for _, asset := range note.Assets {
					matchedAssets[asset] = true
				}
			}
		}
	} else {
		// No query? Collect EVERYTHING for the fuzzy finder
		for _, note := range index.Notes {
			for _, asset := range note.Assets {
				matchedAssets[asset] = true
			}
		}
		// Also include assets from manifest that might not be in notes yet
		if allAssets, err := assetRepo.Search(ctx, ""); err == nil {
			for _, a := range allAssets {
				matchedAssets[a.Filename] = true
			}
		}
	}

	// 3. Build Result Set
	var results []AssetSearchResult

	for filename := range matchedAssets {
		fullPath := appVault.GetAssetPath(filename)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue // Asset file missing
		}

		// Get description from repo
		desc := ""
		if meta, err := assetRepo.Get(ctx, filename); err == nil {
			desc = meta.Description
		}

		// Find usage (Reverse lookup in Index)
		var usedIn []string
		for slug, note := range index.Notes {
			for _, a := range note.Assets {
				if a == filename {
					usedIn = append(usedIn, slug)
					break
				}
			}
		}

		results = append(results, AssetSearchResult{
			Filename:    filename,
			Description: desc,
			UsedIn:      usedIn,
			Path:        fullPath,
			ModTime:     info.ModTime(),
		})
	}

	if len(results) == 0 {
		fmt.Println(ui.FormatWarning("No matching assets found."))
		return nil
	}

	// --- Interactive Mode (No Query) ---
	if query == "" {
		return runInteractiveAssetSearch(results)
	}

	// --- Standard Mode ---

	// 4. Output
	if assetsGallery {
		return generateGallery(results, query)
	}

	fmt.Println()
	for _, res := range results {
		fmt.Printf("%s\n", ui.StyleBold.Render(res.Filename))

		if res.Description != "" {
			fmt.Printf("   %s\n", ui.StyleMuted.Render(res.Description))
		}

		if len(res.UsedIn) > 0 {
			usage := strings.Join(res.UsedIn, ", ")
			// Truncate if too long
			if len(usage) > 60 {
				usage = usage[:57] + "..."
			}
			fmt.Printf("   Used in: %s\n", ui.StyleInfo.Render(usage))
		}

		fmt.Println()
	}

	return nil
}

// runInteractiveAssetSearch launches the fuzzy finder for assets
func runInteractiveAssetSearch(results []AssetSearchResult) error {
	idx, err := fuzzyfinder.Find(
		results,
		func(i int) string {
			r := results[i]
			// Construct a searchable string containing all metadata
			usage := ""
			if len(r.UsedIn) > 0 {
				usage = fmt.Sprintf("[used: %s]", strings.Join(r.UsedIn, ", "))
			}
			return fmt.Sprintf("%s  %s  %s  %s",
				r.Filename,
				r.Description,
				usage,
				r.ModTime.Format("2006-01-02"),
			)
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			r := results[i]

			// Build Preview
			var s strings.Builder
			s.WriteString(fmt.Sprintf("File: %s\n", ui.StyleBold.Render(r.Filename)))
			s.WriteString(fmt.Sprintf("Date: %s\n", r.ModTime.Format("Jan 02, 2006 15:04")))
			s.WriteString("\n")

			if r.Description != "" {
				s.WriteString(ui.StyleHeader.Render("Description") + "\n")
				s.WriteString(r.Description + "\n\n")
			}

			if len(r.UsedIn) > 0 {
				s.WriteString(ui.StyleHeader.Render("Used In") + "\n")
				for _, slug := range r.UsedIn {
					s.WriteString(fmt.Sprintf("• %s\n", slug))
				}
			} else {
				s.WriteString(ui.FormatMuted("(Not used in any notes yet)"))
			}

			return s.String()
		}),
	)

	if err != nil {
		fmt.Println(ui.FormatInfo("Selection cancelled."))
		return nil
	}

	// Action: Copy LaTeX snippet
	selected := results[idx]
	latexSnippet := fmt.Sprintf("\\includegraphics[width=0.8\\linewidth]{%s}", selected.Filename)

	fmt.Println(ui.FormatSuccess("Selected: " + selected.Filename))
	fmt.Println()
	fmt.Println(ui.FormatInfo("LaTeX Code (Copied):"))
	fmt.Println(ui.StyleBold.Render(latexSnippet))

	if err := clipboard.WriteAll(latexSnippet); err != nil {
		fmt.Println(ui.FormatMuted("(Clipboard access failed)"))
	}

	return nil
}

func generateGallery(results []AssetSearchResult, query string) error {
	html := strings.Builder{}
	html.WriteString(`<!DOCTYPE html><html><head><title>LX Gallery</title>
	<style>
		body { font-family: sans-serif; background: #1a1b26; color: #a9b1d6; padding: 20px; }
		.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 20px; }
		.card { background: #24283b; border-radius: 8px; padding: 10px; }
		.card img { width: 100%; height: 150px; object-fit: contain; background: #000; }
		.title { color: #7aa2f7; font-weight: bold; display: block; margin-top: 5px; }
		.desc { font-size: 0.9em; margin-top: 5px; display: block; }
	</style></head><body><h1>Results: "` + query + `"</h1><div class="grid">`)

	for _, res := range results {
		html.WriteString(fmt.Sprintf(`
		<div class="card">
			<a href="file://%s" target="_blank"><img src="file://%s" loading="lazy"></a>
			<span class="title">%s</span>
			<span class="desc">%s</span>
		</div>`, res.Path, res.Path, res.Filename, res.Description))
	}
	html.WriteString(`</div></body></html>`)

	tmpPath := filepath.Join(appVault.CachePath, "gallery.html")
	if err := os.WriteFile(tmpPath, []byte(html.String()), 0644); err != nil {
		return err
	}

	fmt.Println(ui.FormatRocket("Opening gallery..."))
	return OpenFile(tmpPath, "")
}
