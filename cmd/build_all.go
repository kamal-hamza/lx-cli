package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lx/internal/core/services"
	"lx/pkg/ui"
)

var (
	buildAllJobs int
)

// buildAllCmd represents the build-all command
var buildAllCmd = &cobra.Command{
	Use:   "build-all",
	Short: "Build all LaTeX notes concurrently",
	Long: `Build all LaTeX notes to PDF using concurrent workers.

This command uses a worker pool to compile multiple notes in parallel,
dramatically reducing the total build time for large collections.

Examples:
  lx build-all
  lx build-all --jobs 8`,
	RunE: runBuildAll,
}

func init() {
	buildAllCmd.Flags().IntVarP(&buildAllJobs, "jobs", "j", 4, "Number of concurrent workers")
}

func runBuildAll(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	// Get total count first
	listReq := services.ListRequest{}
	listResp, err := listService.Execute(ctx, listReq)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to list notes"))
		return err
	}

	if listResp.Total == 0 {
		fmt.Println(ui.FormatWarning("No notes to build"))
		return nil
	}

	// Show build info
	fmt.Println(ui.FormatRocket("Building all notes..."))
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Total notes", fmt.Sprintf("%d", listResp.Total)))
	fmt.Println(ui.RenderKeyValue("Workers", fmt.Sprintf("%d", buildAllJobs)))
	fmt.Println()

	// Create progress channel
	progressChan := make(chan services.BuildProgress, listResp.Total)

	// Execute build in goroutine
	resultChan := make(chan *services.BuildAllResponse, 1)
	errorChan := make(chan error, 1)

	go func() {
		req := services.BuildAllRequest{
			MaxWorkers: buildAllJobs,
		}
		resp, err := buildService.ExecuteAllWithProgress(ctx, req, progressChan)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- resp
	}()

	// Display progress
	for progress := range progressChan {
		status := ui.FormatSuccess("✓")
		if !progress.Success {
			status = ui.FormatError("✗")
		}

		percentage := float64(progress.Current) / float64(progress.Total) * 100
		progressBar := createProgressBar(percentage, 30)

		fmt.Printf("\r%s [%d/%d] %s %s",
			progressBar,
			progress.Current,
			progress.Total,
			status,
			truncate(progress.Slug, 30),
		)
	}

	// Wait for completion
	var response *services.BuildAllResponse
	select {
	case err := <-errorChan:
		fmt.Println()
		fmt.Println(ui.FormatError("Build failed"))
		return err
	case response = <-resultChan:
		// Continue to show results
	}

	// Final newline after progress
	fmt.Println()
	fmt.Println()

	// Show summary
	fmt.Println(ui.FormatSuccess("Build completed!"))
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Total", fmt.Sprintf("%d", response.Total)))
	fmt.Println(ui.RenderKeyValue("Succeeded", ui.StyleSuccess.Render(fmt.Sprintf("%d", response.Succeeded))))
	if response.Failed > 0 {
		fmt.Println(ui.RenderKeyValue("Failed", ui.StyleError.Render(fmt.Sprintf("%d", response.Failed))))
		fmt.Println()

		// Show failed builds
		fmt.Println(ui.FormatWarning("Failed builds:"))
		for _, result := range response.Results {
			if !result.Success {
				fmt.Println(ui.FormatMuted("  • " + result.Slug + ": " + result.Error.Error()))
			}
		}
	}

	return nil
}

// createProgressBar creates an ASCII progress bar
func createProgressBar(percentage float64, width int) string {
	filled := int(percentage / 100.0 * float64(width))
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return ui.StyleAccent.Render(bar)
}
