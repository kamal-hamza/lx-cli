package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"lx/internal/core/services"
	"lx/pkg/ui"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types" // <--- ADD THIS IMPORT
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Generate an interactive knowledge graph",
	Long: `Analyze your vault and generate a visual graph.

This command forces a fresh scan of your notes to ensure accuracy,
updates the cache, and produces a 'graph.html' file.`,
	RunE: runGraph,
}

func runGraph(cmd *cobra.Command, args []string) error {
	ctx := getContext()

	fmt.Println(ui.FormatRocket("Analyzing knowledge graph..."))

	// 1. Initialize Service
	// FIX: Use correct arguments (Repository, RootPath)
	graphSvc := services.NewGraphService(noteRepo, appVault.RootPath)

	// 2. Fetch Data
	data, err := graphSvc.GetGraph(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to generate graph: %w", err)
	}

	if len(data.Nodes) == 0 {
		fmt.Println(ui.FormatWarning("Graph is empty."))
		return nil
	}

	// 3. Prepare Data
	slugToTitle := make(map[string]string)
	var echartsNodes []opts.GraphNode
	var echartsLinks []opts.GraphLink

	for _, n := range data.Nodes {
		slugToTitle[n.ID] = n.Title

		echartsNodes = append(echartsNodes, opts.GraphNode{
			Name:       n.Title,
			Value:      float32(n.Value),
			SymbolSize: calculateSymbolSize(n.Value),
			// FIX: Cast string to types.FuncStr
			Tooltip: &opts.Tooltip{Show: opts.Bool(true), Formatter: types.FuncStr(fmt.Sprintf("{b}<br/>(%s)", n.ID))},
			// Note: Draggable and Label are controlled by Series options in v2
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

	// 4. Configure Chart
	graph := charts.NewGraph()

	graph.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "LX Knowledge Graph",
			Subtitle: fmt.Sprintf("%d Notes • %d Connections", len(data.Nodes), len(data.Links)),
			Left:     "center",
		}),
		// FIX: Use opts.Bool(true)
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
		charts.WithInitializationOpts(opts.Initialization{
			PageTitle: "LX Knowledge Graph",
			Width:     "100%",
			Height:    "95vh",
		}),
	)

	graph.AddSeries("notes", echartsNodes, echartsLinks).
		SetSeriesOptions(
			charts.WithGraphChartOpts(opts.GraphChart{
				Layout:             "force",
				Roam:               opts.Bool(true), // FIX: opts.Bool
				FocusNodeAdjacency: opts.Bool(true), // FIX: opts.Bool
				Force: &opts.GraphForce{
					Repulsion:  800,
					Gravity:    0.05,
					EdgeLength: 150,
					InitLayout: "circular",
				},
				Draggable: opts.Bool(true), // FIX: Draggable goes here in Series options
			}),
			charts.WithLabelOpts(opts.Label{
				Show:     opts.Bool(true), // FIX: opts.Bool
				Color:    "black",
				Position: "right",
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color:     "source",
				Curveness: 0.2,             // Curveness usually accepts float direct, if error persists use opts.Float(0.2)
				Opacity:   opts.Float(0.6), // Opacity usually accepts float direct
			}),
		)

	// 5. Render
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
	return OpenFileWithDefaultApp(outputPath)
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
