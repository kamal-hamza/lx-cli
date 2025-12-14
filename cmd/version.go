package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kamal-hamza/lx-cli/pkg/ui"
)

// Version information - these can be set during build with ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Display version information",
	Aliases: []string{"v"},
	Long:    `Display the current version of lx along with build information. (alias: v)`,
	Run:     runVersion,
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Println(ui.StyleTitle.Render("LX") + " - LaTeX Notes Manager")
	fmt.Println()
	fmt.Println(ui.RenderKeyValue("Version", Version))
	fmt.Println(ui.RenderKeyValue("Commit", GitCommit))
	fmt.Println(ui.RenderKeyValue("Build Date", BuildDate))
}
