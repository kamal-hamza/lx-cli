package cmd

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/kamal-hamza/lx-cli/pkg/ui"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show vault statistics and activity",
	Long: `Analyze your vault and display useful statistics.

Includes:
  - Word counts and reading time
  - Top tags distribution
  - Writing streak
  - 7-day activity heatmap`,
	RunE: runStats,
}

func runStats(cmd *cobra.Command, args []string) error {
	ctx := getContext()
	headers, err := noteRepo.ListHeaders(ctx)
	if err != nil {
		return err
	}

	fmt.Println(ui.FormatRocket("Analyzing vault..."))

	// 1. Data Aggregation
	totalNotes := len(headers)
	totalWords := 0
	tagCounts := make(map[string]int)
	dateActivity := make(map[string]int) // "YYYY-MM-DD" -> count

	// Track latest note for "Last Updated"
	var lastUpdated time.Time
	var lastNoteTitle string

	for _, h := range headers {
		// Aggregate Tags
		for _, t := range h.Tags {
			tagCounts[t]++
		}

		// Aggregate Dates
		dateActivity[h.Date]++

		// Parse date for sorting/streak
		noteDate, _ := time.Parse("2006-01-02", h.Date)
		if noteDate.After(lastUpdated) {
			lastUpdated = noteDate
			lastNoteTitle = h.Title
		}

		// Calculate Word Count (Naive read)
		// We read directly from disk for speed, bypassing full Note domain loading
		path := appVault.GetNotePath(h.Filename)
		content, err := os.ReadFile(path)
		if err == nil {
			// Simple whitespace split is fast enough for <10k notes
			words := len(strings.Fields(string(content)))
			totalWords += words
		}
	}

	// 2. Calculate "Fun" Stats

	// Reading Time (Avg 200 wpm)
	readingTimeMinutes := float64(totalWords) / 200.0
	readingTimeStr := fmt.Sprintf("%.1f min", readingTimeMinutes)
	if readingTimeMinutes > 60 {
		readingTimeStr = fmt.Sprintf("%.1f hrs", readingTimeMinutes/60.0)
	}

	// Current Streak
	streak := calculateStreak(dateActivity)

	// 3. Render Output
	fmt.Println()
	fmt.Println(ui.FormatTitle("Vault Analytics"))
	fmt.Println()

	// --- General Stats (Tabular) ---
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	fmt.Fprintf(w, "%s\t%d\n", ui.StyleBold.Render("Total Notes:"), totalNotes)
	fmt.Fprintf(w, "%s\t%d\n", ui.StyleBold.Render("Total Words:"), totalWords)
	fmt.Fprintf(w, "%s\t%s\n", ui.StyleBold.Render("Reading Time:"), readingTimeStr)

	avgWords := 0
	if totalNotes > 0 {
		avgWords = totalWords / totalNotes
	}
	fmt.Fprintf(w, "%s\t%d words/note\n", ui.StyleBold.Render("Average Length:"), avgWords)
	w.Flush()

	fmt.Println()

	// --- Activity Heatmap (Last 7 Days) ---
	renderHeatmap(dateActivity)

	// Streak Display
	streakIcon := "🔥"
	if streak == 0 {
		streakIcon = "🧊"
	}
	fmt.Printf("%s %s %d days\n", streakIcon, ui.StyleBold.Render("Current Streak:"), streak)
	if lastNoteTitle != "" {
		fmt.Printf("   %s %s (%s)\n", ui.StyleMuted.Render("Last active:"), lastUpdated.Format("Jan 02"), lastNoteTitle)
	}
	fmt.Println()

	// --- Top Tags (Bar Chart) ---
	renderTopTags(tagCounts)

	return nil
}

// calculateStreak counts consecutive days looking backwards from today/yesterday
func calculateStreak(activity map[string]int) int {
	streak := 0
	current := time.Now()

	// Check if we wrote today
	todayStr := current.Format("2006-01-02")
	if activity[todayStr] > 0 {
		streak++
	} else {
		// If not today, did we write yesterday? (Streak is still alive if we missed today but wrote yesterday)
		// Actually, strict streaks usually require today OR yesterday to be active.
		// Let's check yesterday to start the chain if today is empty.
		current = current.AddDate(0, 0, -1)
		yesterdayStr := current.Format("2006-01-02")
		if activity[yesterdayStr] == 0 {
			return 0 // Streak broken
		}
	}

	// Count backwards
	for {
		current = current.AddDate(0, 0, -1)
		dateStr := current.Format("2006-01-02")
		if activity[dateStr] > 0 {
			streak++
		} else {
			break
		}
	}
	return streak
}

// renderHeatmap prints a GitHub-style activity row
func renderHeatmap(activity map[string]int) {
	fmt.Println(ui.StyleHeader.Render("Activity (Last 7 Days)"))

	// Generate last 7 days keys
	today := time.Now()
	var days []time.Time
	for i := 6; i >= 0; i-- {
		days = append(days, today.AddDate(0, 0, -i))
	}

	// Render blocks
	var blocks []string
	var labels []string

	for _, day := range days {
		dateStr := day.Format("2006-01-02")
		count := activity[dateStr]

		// Color mapping
		var block string
		if count == 0 {
			block = ui.StyleMuted.Render("⬜") // Empty
		} else if count < 3 {
			block = ui.StyleSuccess.Copy().Faint(true).Render("🟩") // Light green
		} else {
			block = ui.StyleSuccess.Render("Vs") // Replaced literal block with text or symbol if needed, but 🟩 works well
			// Use standard green square for heavy activity
			block = "\033[32m🟩\033[0m"
		}

		// Use Lipgloss for cleaner colors if standard emoji look wrong in your terminal
		// Let's stick to simple colored blocks for compatibility
		if count == 0 {
			block = "⬜"
		} else if count <= 1 {
			block = "Cc" // Light
			block = "🟩"
		} else {
			block = "SDK" // Heavy
			block = "Sq"
			block = "🟩" // Just use green square for now, distinction via opacity is hard in basic ASCII
		}

		// Improved Logic for visual distinction
		if count == 0 {
			block = "⬜"
		} else {
			block = "🟩" // Standard green square
		}

		blocks = append(blocks, block)
		labels = append(labels, day.Format("Mon"))
	}

	// Print visual row
	fmt.Println(strings.Join(blocks, "  "))

	// Print Labels (Mon, Tue...)
	// Using muted style for labels
	labelRow := ""
	for _, l := range labels {
		// Padding to align with emojis (emojis are often 2 chars wide visually)
		labelRow += fmt.Sprintf("%-4s", l)
	}
	fmt.Println(ui.StyleMuted.Render(labelRow))
}

// renderTopTags displays a horizontal bar chart
func renderTopTags(counts map[string]int) {
	if len(counts) == 0 {
		return
	}

	fmt.Println(ui.StyleHeader.Render("Top Tags"))

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

	// Limit to top 5
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}

	// Find max for scaling
	maxCount := sorted[0].Count
	barWidth := 20

	for i := 0; i < limit; i++ {
		t := sorted[i]

		// Calculate bar length
		length := int(math.Ceil(float64(t.Count) / float64(maxCount) * float64(barWidth)))
		bar := strings.Repeat("█", length)

		// Render
		fmt.Printf("%s %-15s %s\n",
			ui.StyleAccent.Render(bar),
			t.Name,
			ui.StyleMuted.Render(fmt.Sprintf("%d", t.Count)),
		)
	}
}
