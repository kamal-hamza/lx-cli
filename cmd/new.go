package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"lx/internal/core/services"
	"lx/pkg/ui"
)

var (
	newTemplateName string
	newTags         []string
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new [title]",
	Short: "Create a new LaTeX note",
	Long: `Create a new LaTeX note with optional template and tags.

Examples:
  lx new "Graph Theory Notes"
  lx new "Chemistry Lab" --template homework --tags science,lab
  lx new "Calculus Chapter 3" -t math-common --tags math,calculus`,
	Args: cobra.ExactArgs(1),
	RunE: runNew,
}

func init() {
	newCmd.Flags().StringVarP(&newTemplateName, "template", "t", "", "Template to use (e.g., homework, ieee)")
	newCmd.Flags().StringSliceVar(&newTags, "tags", []string{}, "Tags for the note (comma-separated)")
}

func runNew(cmd *cobra.Command, args []string) error {
	title := args[0]

	// Create the note
	req := services.CreateNoteRequest{
		Title:        title,
		Tags:         newTags,
		TemplateName: newTemplateName,
	}

	ctx := getContext()
	resp, err := createNoteService.Execute(ctx, req)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to create note"))
		return err
	}

	// Success message
	fmt.Println(ui.FormatSuccess("Note created successfully!"))
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Title", resp.Note.Header.Title))
	fmt.Println(ui.RenderKeyValue("Slug", resp.Note.Header.Slug))
	fmt.Println(ui.RenderKeyValue("File", resp.FilePath))
	if len(resp.Note.Header.Tags) > 0 {
		fmt.Println(ui.RenderKeyValue("Tags", strings.Join(resp.Note.Header.Tags, ", ")))
	}
	if newTemplateName != "" {
		fmt.Println(ui.RenderKeyValue("Template", newTemplateName))
	}
	fmt.Println()

	// Get the full path
	notePath := appVault.GetNotePath(resp.FilePath)

	// Open in editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default fallback
	}

	fmt.Println(ui.FormatInfo("Opening in editor: " + editor))
	fmt.Println()

	// Launch editor
	editorCmd := exec.Command(editor, notePath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		fmt.Println(ui.FormatWarning("Failed to open editor: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually edit: " + notePath))
	}

	return nil
}
