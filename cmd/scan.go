package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gi4nks/tecla/gitinfo"
	"github.com/gi4nks/tecla/internal/config"
	"github.com/gi4nks/tecla/report"
	"github.com/gi4nks/tecla/scanner"
	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	var (
		roots         []string
		format        string
		output        string
		progress      bool
		maxDepth      int
		exclude       []string
		includeHidden bool
		workers       int
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan root folder(s) for Git repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				roots = append(roots, args...)
			}
			if len(roots) == 0 {
				roots = []string{"."}
			}
			var absRoots []string
			for _, r := range roots {
				abs, err := filepath.Abs(r)
				if err != nil {
					absRoots = append(absRoots, r)
				} else {
					absRoots = append(absRoots, abs)
				}
			}

			slog.Info("Starting scan", "roots", absRoots, "maxDepth", maxDepth)

			format = strings.ToLower(strings.TrimRight(strings.TrimSpace(format), ".,;:"))
			switch format {
			case "table", "json", "markdown":
			default:
				return fmt.Errorf("unsupported format %q", format)
			}

			cfg, err := config.Load()
			if err == nil {
				exclude = append(exclude, cfg.IgnoredPaths...)
			} else {
				cfg = &config.Config{
					DefaultIgnoredDirs: []string{"node_modules", "dist", "build", ".cache", ".venv", "target", ".terraform"},
					StaleThresholdDays: 30,
				}
			}

			repos, scanErrs := scanner.Scan(scanner.Options{
				Roots:              absRoots,
				IncludeHidden:      includeHidden,
				ExcludePatterns:    exclude,
				DefaultIgnoredDirs: cfg.DefaultIgnoredDirs,
				MaxDepth:           maxDepth,
			})

			for _, e := range scanErrs {
				slog.Error("Scan error", "error", e)
			}

			sort.Strings(repos)
			slog.Debug("Repositories found", "count", len(repos))

			workerCount := workers
			if workerCount <= 0 {
				workerCount = runtime.NumCPU()
			}

			var progressFn gitinfo.ProgressFunc
			if progress {
				progressFn = progressPrinter(cmd.ErrOrStderr(), len(repos))
			}

			var customRecs []gitinfo.CustomRecommendation
			for _, cr := range cfg.CustomRecommendations {
				customRecs = append(customRecs, gitinfo.CustomRecommendation{
					Condition: cr.Condition,
					Text:      cr.Text,
					Command:   cr.Command,
				})
			}

			slog.Info("Collecting Git information", "workers", workerCount)
			infos := gitinfo.Collect(context.Background(), repos, gitinfo.Options{
				Timeout:               3 * time.Second,
				Workers:               workerCount,
				StaleThresholdDays:    cfg.StaleThresholdDays,
				CustomRecommendations: customRecs,
			}, progressFn)

			sort.Slice(infos, func(i, j int) bool {
				return infos[i].Path < infos[j].Path
			})

			rep := report.Report{
				Roots:       absRoots,
				GeneratedAt: time.Now().UTC(),
				Repos:       infos,
				ScanErrors:  scanErrs,
			}

			writer := cmd.OutOrStdout()
			if output != "" {
				file, err := os.Create(output)
				if err != nil {
					return err
				}
				defer file.Close()
				writer = file
			}

			switch format {
			case "table":
				return report.RenderTable(writer, rep)
			case "json":
				return report.RenderJSON(writer, rep)
			case "markdown":
				return report.RenderMarkdown(writer, rep)
			default:
				return errors.New("unknown format")
			}
		},
	}

	cmd.Flags().StringSliceVar(&roots, "root", nil, "Root path(s) to scan")
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json, markdown")
	cmd.Flags().StringVar(&output, "output", "", "Write output to file")
	cmd.Flags().BoolVar(&progress, "progress", false, "Show progress while collecting repo data")
	cmd.Flags().IntVar(&maxDepth, "max-depth", -1, "Maximum scan depth (-1 for unlimited)")
	cmd.Flags().StringArrayVar(&exclude, "exclude", nil, "Exclude path patterns (repeatable)")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "Include hidden folders")
	cmd.Flags().IntVar(&workers, "workers", runtime.NumCPU(), "Number of workers for Git inspection")

	return cmd
}

func progressPrinter(w io.Writer, total int) gitinfo.ProgressFunc {
	var done int64
	return func() {
		count := atomic.AddInt64(&done, 1)
		fmt.Fprintf(w, "\rProcessed %d/%d", count, total)
		if int(count) == total {
			fmt.Fprintln(w)
		}
	}
}
