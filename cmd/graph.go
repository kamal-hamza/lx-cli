package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kamal-hamza/lx-cli/pkg/ui"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:     "graph",
	Aliases: []string{"gg"},
	Short:   "Generate an interactive knowledge graph (alias: gg)",
	Long: `Analyze your vault and generate a visual graph.

This command forces a fresh scan of your notes to ensure accuracy,
updates the cache, and produces a 'graph.html' file.`,
	RunE: runGraph,
}

func runGraph(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	fmt.Println(ui.FormatRocket("Generating knowledge graph..."))

	// 1. Fetch Data
	// Use global graphService (already initialized with indexer)
	data, err := graphService.GetGraph(ctx, true) // force refresh
	if err != nil {
		return fmt.Errorf("failed to generate graph: %w", err)
	}

	if len(data.Nodes) == 0 {
		fmt.Println(ui.FormatWarning("Graph is empty."))
		return nil
	}

	// 2. Prepare Data
	slugToTitle := make(map[string]string)
	var echartsNodes []opts.GraphNode
	var echartsLinks []opts.GraphLink

	for _, n := range data.Nodes {
		slugToTitle[n.ID] = n.Title

		echartsNodes = append(echartsNodes, opts.GraphNode{
			Name:       n.Title,
			Value:      float32(n.Value),
			SymbolSize: calculateSymbolSize(n.Value),
			Tooltip:    &opts.Tooltip{Show: opts.Bool(true), Formatter: types.FuncStr(fmt.Sprintf("{b}<br/>(%s)", n.ID))},
		})
	}

	for _, l := range data.Links {
		sourceTitle := slugToTitle[l.Source]
		targetTitle := slugToTitle[l.Target]

		if sourceTitle != "" && targetTitle != "" {
			echartsLinks = append(echartsLinks, opts.GraphLink{
				Source: sourceTitle,
				Target: targetTitle,
			})
		}
	}

	// 3. Configure Chart
	graph := charts.NewGraph()

	graph.SetGlobalOptions(
		// Removed TitleOpts entirely
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
		charts.WithInitializationOpts(opts.Initialization{
			PageTitle: "LX Knowledge Graph",
			Width:     "100%",
			Height:    "100vh",
		}),
	)

	graph.AddSeries("notes", echartsNodes, echartsLinks).
		SetSeriesOptions(
			charts.WithGraphChartOpts(opts.GraphChart{
				Layout:             "force",
				Roam:               opts.Bool(true),
				FocusNodeAdjacency: opts.Bool(true),
				Force: &opts.GraphForce{
					Repulsion:  800,
					Gravity:    0.05,
					EdgeLength: 150,
					InitLayout: "circular",
				},
				Draggable: opts.Bool(true),
			}),
			charts.WithLabelOpts(opts.Label{
				Show:     opts.Bool(true),
				Color:    "black",
				Position: "right",
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color:     "source",
				Curveness: 0.2,
				Opacity:   opts.Float(0.6),
			}),
		)

	// 4. Render
	outputPath := filepath.Join(appVault.RootPath, "graph.html")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create graph file: %w", err)
	}
	defer f.Close()

	if err := graph.Render(f); err != nil {
		return fmt.Errorf("failed to render graph: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.FormatSuccess("Graph generated successfully!"))
	fmt.Println(ui.RenderKeyValue("Location", outputPath))

	fmt.Println(ui.FormatInfo("Opening in browser..."))
	return OpenFile(outputPath, "")
}

func calculateSymbolSize(connections int) float32 {
	base := float32(20.0)
	cap := float32(100.0)
	size := base + (float32(connections) * 3.0)
	if size > cap {
		return cap
	}
	return size
}
