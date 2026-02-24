package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

var (
	verbose bool
	logger  *slog.Logger
)

var rootCmd = &cobra.Command{
	Use:   "tecla",
	Short: "Scan folders for Git repositories and report status",
	Long:  "tecla scans a root folder, detects Git repositories, and reports their status with recommended actions.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		
		// If we are in UI mode, we might want to disable logging or log to a file
		// For now, we only log if verbose is set, otherwise we use a no-op handler
		if verbose {
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
		} else {
			f, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
			logger = slog.New(slog.NewTextHandler(f, nil))
		}
		slog.SetDefault(logger)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDefaultUI()
	},
}

func Execute() error {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	rootCmd.AddCommand(newScanCmd())
	rootCmd.AddCommand(newUICmd())
	rootCmd.AddCommand(newCheckCmd())
	rootCmd.AddCommand(newIgnoreCmd())
	rootCmd.AddCommand(newVersionCmd())
	return rootCmd.Execute()
}

func versionString() string {
	if Commit == "" && Date == "" {
		return Version
	}
	if Date == "" {
		return fmt.Sprintf("%s (%s)", Version, Commit)
	}
	if Commit == "" {
		return fmt.Sprintf("%s (%s)", Version, Date)
	}
	return fmt.Sprintf("%s (%s, %s)", Version, Commit, Date)
}
