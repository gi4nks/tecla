package cmd

import (
	"fmt"
	"strings"

	"github.com/gi4nks/tecla/internal/config"
	"github.com/spf13/cobra"
)

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile [name]",
		Short: "Set or show the active profile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				if cfg.ActiveProfile == "" {
					fmt.Println("No active profile.")
				} else {
					fmt.Printf("Active profile: %s\n", cfg.ActiveProfile)
				}
				fmt.Println("Available profiles:")
				for _, p := range cfg.Profiles {
					fmt.Printf("  - %s: %s\n", p.Name, strings.Join(p.Roots, ", "))
				}
				return nil
			}

			name := args[0]
			found := false
			for _, p := range cfg.Profiles {
				if p.Name == name {
					found = true
					break
				}
			}

			if !found && name != "none" {
				return fmt.Errorf("profile %q not found", name)
			}

			if name == "none" {
				cfg.ActiveProfile = ""
				fmt.Println("Cleared active profile.")
			} else {
				cfg.ActiveProfile = name
				fmt.Printf("Set active profile to: %s\n", name)
			}

			return config.Save(cfg)
		},
	}
	return cmd
}
