package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/oops-guard/internal/analyzer"
	"github.com/agenticraptor/oops-guard/internal/config"
	"github.com/agenticraptor/oops-guard/internal/confirm"
	"github.com/agenticraptor/oops-guard/internal/render"
)

func newGuardCmd() *cobra.Command {
	var shell string
	cmd := &cobra.Command{
		Use:   "guard -- <command>",
		Short: "Inspect a command for a shell hook (exit non-zero blocks it)",
		Long: `guard is the entry point used by the shell integration. It analyzes the
command, and if it crosses your configured severity threshold it prints the
warning and asks for confirmation on your terminal. It exits 0 to allow the
command and non-zero to block it.

You normally don't run this yourself — install the hook with:

  oops-guard init <bash|zsh|fish>`,
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			line := strings.TrimSpace(strings.Join(args, " "))
			if line == "" {
				return nil
			}
			cfg, _ := config.Load()
			if cfg.Allowed(line) {
				return nil // explicitly trusted
			}

			// Fast path: classify with no filesystem access at all. The vast
			// majority of commands are safe and return here in microseconds. Only
			// when a command actually crosses the threshold do we pay for the
			// (read-only) impact preview.
			threshold := analyzer.ParseSeverity(cfg.MinSeverity)
			if !analyzer.Analyze(line, analyzer.Options{NoImpact: true}).Dangerous(threshold) {
				return nil
			}
			a := analyzer.Analyze(line, analyzer.Options{}) // enrich for the warning

			out, closeFn := ttyWriter()
			defer closeFn()
			fmt.Fprintln(out)
			_ = render.Render(out, a, render.Options{Format: render.FormatTerm, Color: true})

			critical := a.Max() >= analyzer.Critical
			ok, err := confirm.Ask("Proceed?", critical)
			if err != nil {
				// No terminal to confirm on: surface the warning but don't block,
				// so non-interactive contexts aren't broken.
				fmt.Fprintln(out, "oops-guard: no terminal to confirm on — allowing.")
				return nil
			}
			if ok {
				return nil
			}
			fmt.Fprintln(out, "oops-guard: cancelled.")
			return exitCodeError{declineExitCode}
		},
	}
	cmd.Flags().StringVar(&shell, "shell", "", "calling shell (bash|zsh|fish); accepted from the hook")
	_ = shell // accepted so the hooks can pass it; reserved for future shell-specific behavior
	cmd.Flags().SetInterspersed(false)
	return cmd
}
