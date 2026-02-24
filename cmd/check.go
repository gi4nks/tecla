package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/gi4nks/tecla/gitinfo"
	"github.com/gi4nks/tecla/internal/config"
	"github.com/gi4nks/tecla/scanner"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	var (
		roots         []string
		maxDepth      int
		exclude       []string
		includeHidden bool
		workers       int
		failOnDirty   bool
		failOnBehind  bool
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check repository health and exit with non-zero if issues found",
		Long:  "Scans repositories and returns exit code 1 if any repository is dirty or behind upstream (configurable).",
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

			slog.Info("Starting health check", "roots", absRoots)

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
			}, nil)

			issuesFound := 0
			for _, repo := range infos {
				hasIssue := false
				reason := ""
				if failOnDirty && !repo.Status.Clean {
					hasIssue = true
					reason = "dirty"
				}
				if failOnBehind && repo.Behind > 0 {
					hasIssue = true
					if reason != "" {
						reason += ", "
					}
					reason += "behind"
				}
				if repo.Error != "" {
					hasIssue = true
					if reason != "" {
						reason += ", "
					}
					reason += "error"
				}

				if hasIssue {
					issuesFound++
					slog.Warn("Issue found", "repo", repo.Path, "reason", reason, "error", repo.Error)
					status := "clean"
					if !repo.Status.Clean {
						status = "dirty"
					}
					fmt.Printf("FAIL: %s (status: %s, behind: %d, error: %s)\n", repo.Path, status, repo.Behind, repo.Error)
				}
			}

			if issuesFound > 0 {
				slog.Error("Health check failed", "issues", issuesFound)
				fmt.Printf("\nFound %d repositories with issues.\n", issuesFound)
				os.Exit(1)
			}

			slog.Info("Health check passed")
			fmt.Println("All repositories are healthy.")
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&roots, "root", nil, "Root path(s) to scan")
	cmd.Flags().IntVar(&maxDepth, "max-depth", -1, "Maximum scan depth (-1 for unlimited)")
	cmd.Flags().StringArrayVar(&exclude, "exclude", nil, "Exclude path patterns (repeatable)")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "Include hidden folders")
	cmd.Flags().IntVar(&workers, "workers", runtime.NumCPU(), "Number of workers for Git inspection")
	cmd.Flags().BoolVar(&failOnDirty, "dirty", true, "Fail if any repository is dirty")
	cmd.Flags().BoolVar(&failOnBehind, "behind", false, "Fail if any repository is behind upstream")

	return cmd
}
