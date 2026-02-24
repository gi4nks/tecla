package cmd

import (
	"path/filepath"
	"runtime"
	"time"

	"github.com/gi4nks/tecla/gitinfo"
	"github.com/gi4nks/tecla/internal/config"
	"github.com/gi4nks/tecla/tui"
	"github.com/spf13/cobra"
)

func newUICmd() *cobra.Command {
	var (
		roots         []string
		exclude       []string
		includeHidden bool
		maxDepth      int
		workers       int
	)

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the interactive UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				roots = append(roots, args...)
			}
			return runUI(roots, includeHidden, exclude, maxDepth, workers)
		},
	}

	cmd.Flags().StringSliceVar(&roots, "root", nil, "Root path(s) to scan")
	cmd.Flags().StringArrayVar(&exclude, "exclude", nil, "Exclude path patterns (repeatable)")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "Include hidden folders")
	cmd.Flags().IntVar(&maxDepth, "max-depth", -1, "Maximum scan depth (-1 for unlimited)")
	cmd.Flags().IntVar(&workers, "workers", runtime.NumCPU(), "Number of workers for Git inspection")

	return cmd
}

func runUI(roots []string, includeHidden bool, exclude []string, maxDepth int, workers int) error {
	cfg, err := config.Load()
	if err == nil {
		exclude = append(exclude, cfg.IgnoredPaths...)
	} else {
		cfg = &config.Config{
			DefaultIgnoredDirs: []string{"node_modules", "dist", "build", ".cache", ".venv", "target", ".terraform"},
			StaleThresholdDays: 30,
		}
	}

	if len(roots) == 0 {
		activeRoots := cfg.GetActiveRoots()
		if len(activeRoots) > 0 {
			roots = activeRoots
		} else {
			roots = []string{"."}
		}
	}

	var absRoots []string
	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err == nil {
			absRoots = append(absRoots, absRoot)
		} else {
			absRoots = append(absRoots, root)
		}
	}

	var customRecs []gitinfo.CustomRecommendation
	for _, cr := range cfg.CustomRecommendations {
		customRecs = append(customRecs, gitinfo.CustomRecommendation{
			Condition: cr.Condition,
			Text:      cr.Text,
			Command:   cr.Command,
		})
	}

	return tui.Run(tui.Options{
		Roots:                 absRoots,
		IncludeHidden:         includeHidden,
		Exclude:               exclude,
		DefaultIgnoredDirs:    cfg.DefaultIgnoredDirs,
		MaxDepth:              maxDepth,
		Workers:               workers,
		Timeout:               3 * time.Second,
		StaleThresholdDays:    cfg.StaleThresholdDays,
		CustomRecommendations: customRecs,
	})
}

func runDefaultUI() error {
	return runUI([]string{"."}, false, nil, -1, runtime.NumCPU())
}
