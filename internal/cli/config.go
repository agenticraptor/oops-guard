package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/oops-guard/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <init|path|show>",
		Short: "Manage the optional configuration file",
		Long: `oops-guard works with no configuration at all. The config file only lets
you change the severity threshold that interrupts your shell, allow-list
commands you trust, and point the explain command at a model.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "path":
				fmt.Println(config.Path())
				return nil
			case "init":
				path, err := config.WriteStarter()
				if err != nil {
					return err
				}
				fmt.Printf("Wrote a starter config to %s\n", path)
				return nil
			case "show":
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				fmt.Printf("config file: %s\n", config.Path())
				if _, err := os.Stat(config.Path()); os.IsNotExist(err) {
					fmt.Println("(none on disk — showing defaults)")
				}
				fmt.Printf("min_severity: %s\n", cfg.MinSeverity)
				fmt.Printf("allow:        %d pattern(s)\n", len(cfg.Allow))
				fmt.Printf("ai.enabled:   %t\n", cfg.AI.Enabled)
				if cfg.AI.Provider != "" {
					fmt.Printf("ai.provider:  %s\n", cfg.AI.Provider)
				}
				return nil
			default:
				return fmt.Errorf("unknown subcommand %q (want init, path, or show)", args[0])
			}
		},
	}
	return cmd
}
