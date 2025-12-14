package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/internal/core/services"
	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

var (
	newTemplateName string
	newTags         []string
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:     "new [title|template] [title]",
	Short:   "Create a new LaTeX note or template",
	Aliases: []string{"n", "create"},
	Long: `Create a new LaTeX note with optional template and tags, or create a new template.

Examples:
  # Create a note
  lx new "Graph Theory Notes"
  lx new "Chemistry Lab" --template homework --tags science,lab
  lx new "Calculus Chapter 3" -t math-common --tags math,calculus

  # Create a template
  lx new template "My Custom Template"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runNewDispatcher,
}

func init() {
	newCmd.Flags().StringVarP(&newTemplateName, "template", "t", "", "Template to use (e.g., homework, ieee)")
	newCmd.Flags().StringSliceVar(&newTags, "tags", []string{}, "Tags for the note (comma-separated)")
}

// runNewDispatcher determines whether to create a note or template
func runNewDispatcher(cmd *cobra.Command, args []string) error {
	// Check if first arg is "template" OR "t"
	if args[0] == "template" || args[0] == "t" {
		if len(args) < 2 {
			fmt.Println(ui.FormatError("Template title is required"))
			return fmt.Errorf("usage: lx new template \"title\"")
		}
		return runNewTemplate(cmd, args)
	}

	// Otherwise, create a note
	return runNew(cmd, args)
}

func runNew(cmd *cobra.Command, args []string) error {
	title := args[0]

	// Create the note
	req := services.CreateNoteRequest{
		Title:        title,
		Tags:         newTags,
		TemplateName: newTemplateName,
		DateFormat:   appConfig.DateFormat, // Use configured date format
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

	// Use preferred editor
	editor := GetPreferredEditor()
	fmt.Println(ui.FormatInfo("Opening in editor: " + editor))
	fmt.Println()

	if err := OpenEditorAtLine(notePath, 1); err != nil {
		fmt.Println(ui.FormatWarning("Failed to open editor: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually edit: " + notePath))
	}

	return nil
}

func runNewTemplate(cmd *cobra.Command, args []string) error {
	title := args[1]

	// Create the template
	req := services.CreateTemplateRequest{
		Title:   title,
		Content: "", // Empty content, user will edit in editor
	}

	ctx := getContext()
	resp, err := createTemplateService.Execute(ctx, req)
	if err != nil {
		fmt.Println(ui.FormatError("Failed to create template"))
		return err
	}

	// Success message
	fmt.Println(ui.FormatSuccess("Template created successfully!"))
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Title", resp.Template.Header.Title))
	fmt.Println(ui.RenderKeyValue("Slug", resp.Template.Header.Slug))
	fmt.Println(ui.RenderKeyValue("File", resp.FilePath))
	fmt.Println()

	// Get the full path
	templatePath := appVault.GetTemplatePath(resp.FilePath)

	// Use preferred editor
	editor := GetPreferredEditor()
	fmt.Println(ui.FormatInfo("Opening in editor: " + editor))
	fmt.Println()

	if err := OpenEditorAtLine(templatePath, 1); err != nil {
		fmt.Println(ui.FormatWarning("Failed to open editor: " + err.Error()))
		fmt.Println(ui.FormatInfo("You can manually edit: " + templatePath))
	}

	return nil
}
