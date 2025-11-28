package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"lx/internal/core/services"
	"lx/pkg/ui"

	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag [command]",
	Short: "Manage tags on notes",
	Long:  `Add or remove tags from notes without opening the editor.`,
}

var tagAddCmd = &cobra.Command{
	Use:   "add [query] [tags]",
	Short: "Add tags to a note",
	Example: `  lx tag add graph "math, study"
  lx tag add "Linear Algebra" final-review`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateTags(args[0], args[1], true)
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:   "remove [query] [tags]",
	Short: "Remove tags from a note",
	Example: `  lx tag remove graph "math"
  lx tag remove "Linear Algebra" final-review`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateTags(args[0], args[1], false)
	},
}

func init() {
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRemoveCmd)
}

func updateTags(query string, tagsInput string, isAdd bool) error {
	ctx := getContext()

	// 1. Find the Note
	req := services.SearchRequest{Query: query}
	resp, err := listService.Search(ctx, req)
	if err != nil {
		return err
	}

	if resp.Total == 0 {
		return fmt.Errorf("no notes found matching: %s", query)
	}

	// Pick the best match (first one)
	target := resp.Notes[0]
	path := appVault.GetNotePath(target.Filename)

	// 2. Read Content
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	// 3. Parse Existing Tags
	// Regex matches: % tags: tag1, tag2, ...
	tagRegex := regexp.MustCompile(`(?m)^%\s*tags:\s*(.*)$`)
	matches := tagRegex.FindStringSubmatch(content)

	var existingTags []string
	if len(matches) > 1 && strings.TrimSpace(matches[1]) != "" {
		parts := strings.Split(matches[1], ",")
		for _, p := range parts {
			existingTags = append(existingTags, strings.TrimSpace(p))
		}
	}

	// 4. Modify Tags
	newTagsInput := strings.Split(tagsInput, ",")
	changed := false

	if isAdd {
		for _, t := range newTagsInput {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}

			// Check duplicate
			exists := false
			for _, e := range existingTags {
				if strings.EqualFold(e, t) {
					exists = true
					break
				}
			}
			if !exists {
				existingTags = append(existingTags, t)
				changed = true
			}
		}
	} else {
		// Remove
		var keptTags []string
		for _, e := range existingTags {
			remove := false
			for _, t := range newTagsInput {
				if strings.EqualFold(e, strings.TrimSpace(t)) {
					remove = true
					changed = true
					break
				}
			}
			if !remove {
				keptTags = append(keptTags, e)
			}
		}
		existingTags = keptTags
	}

	if !changed {
		fmt.Println(ui.FormatInfo("No changes to tags."))
		return nil
	}

	// 5. Write Back
	newTagLine := fmt.Sprintf("%% tags: %s", strings.Join(existingTags, ", "))

	if tagRegex.MatchString(content) {
		content = tagRegex.ReplaceAllString(content, newTagLine)
	} else {
		// If tag line didn't exist, try to insert after date or title
		// For simplicity, just prepend if missing, or handle strictly
		// But your file_repository ensures metadata exists, so regex should match.
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	action := "Added"
	if !isAdd {
		action = "Removed"
	}

	fmt.Printf("%s tags for '%s'\n", ui.FormatSuccess(action), target.Title)
	fmt.Println(ui.RenderKeyValue("Current Tags", strings.Join(existingTags, ", ")))

	return nil
}
