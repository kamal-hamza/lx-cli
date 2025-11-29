package cmd

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/kamal-hamza/lx-cli/pkg/ui"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show vault statistics",
	Long: `Display summary statistics for the vault.

Shows:
  - Note and asset counts
  - Storage usage
  - Tag distribution`,
	RunE: runStats,
}

func runStats(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// 1. Fetch Data
	headers, err := noteRepo.ListHeaders(ctx)
	if err != nil {
		return err
	}

	// 2. Note Statistics
	totalNotes := len(headers)
	totalWords := 0
	tagCounts := make(map[string]int)

	var lastUpdated time.Time
	var lastNoteTitle string

	for _, h := range headers {
		for _, t := range h.Tags {
			tagCounts[t]++
		}

		noteDate, _ := time.Parse("2006-01-02", h.Date)
		if noteDate.After(lastUpdated) {
			lastUpdated = noteDate
			lastNoteTitle = h.Title
		}

		path := appVault.GetNotePath(h.Filename)
		content, err := os.ReadFile(path)
		if err == nil {
			words := len(strings.Fields(string(content)))
			totalWords += words
		}
	}

	// 3. Asset Statistics
	totalAssets := 0
	totalAssetSize := int64(0)

	filepath.WalkDir(appVault.AssetsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == ".manifest.json" {
			return nil
		}
		info, _ := d.Info()
		totalAssets++
		totalAssetSize += info.Size()
		return nil
	})

	// 4. Render Output
	fmt.Println()
	fmt.Println(ui.FormatTitle("Vault Statistics"))
	fmt.Println()

	// General Stats Table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	// Notes Info
	fmt.Fprintf(w, "Notes:\t%d\n", totalNotes)
	fmt.Fprintf(w, "Words:\t%d\n", totalWords)

	avgWords := 0
	if totalNotes > 0 {
		avgWords = totalWords / totalNotes
	}
	fmt.Fprintf(w, "Avg Length:\t%d words\n", avgWords)

	// Assets Info
	assetSizeStr := formatBytes(totalAssetSize)
	fmt.Fprintf(w, "Assets:\t%d (%s)\n", totalAssets, assetSizeStr)

	// Latest Activity
	if !lastUpdated.IsZero() {
		fmt.Fprintf(w, "Last Active:\t%s (%s)\n", lastUpdated.Format("2006-01-02"), lastNoteTitle)
	}

	w.Flush()
	fmt.Println()

	// Top Tags
	renderTopTags(tagCounts)

	return nil
}

// formatBytes converts bytes to human readable string
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func renderTopTags(counts map[string]int) {
	if len(counts) == 0 {
		return
	}

	fmt.Println("Top Tags:")

	// Sort tags by count
	type tagPair struct {
		Name  string
		Count int
	}
	var sorted []tagPair
	for k, v := range counts {
		sorted = append(sorted, tagPair{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	// Limit to top 10
	limit := 10
	if len(sorted) < limit {
		limit = len(sorted)
	}

	// Find max for scaling
	maxCount := sorted[0].Count
	barWidth := 20.0

	for i := 0; i < limit; i++ {
		t := sorted[i]

		// Calculate bar length
		length := int(math.Ceil(float64(t.Count) / float64(maxCount) * barWidth))
		bar := strings.Repeat("#", length)

		fmt.Printf("  %-15s %s (%d)\n", t.Name, bar, t.Count)
	}
}
