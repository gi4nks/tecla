package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/gi4nks/tecla/internal/config"
	"github.com/spf13/cobra"
)

func newIgnoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ignore [path]",
		Short: "Add a path to the ignore list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if cfg.AddIgnore(absPath) {
				if err := config.Save(cfg); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				fmt.Printf("Added %s to ignore list\n", absPath)
			} else {
				fmt.Printf("%s is already in the ignore list\n", absPath)
			}

			return nil
		},
	}

	return cmd
}
