package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/agenticraptor/oops-guard/internal/analyzer"
)

func newRulesCmd() *cobra.Command {
	var asJSON bool
	var noColor bool
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "List everything oops-guard watches for",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cat := analyzer.Catalog()
			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(cat)
			}
			color := useColor(noColor, os.Stdout)
			fmt.Printf("oops-guard watches for %d kinds of destructive command:\n\n", len(cat))
			for _, r := range cat {
				tag := fmt.Sprintf("%-8s", r.Severity.Label())
				if color {
					tag = lipgloss.NewStyle().Bold(true).Foreground(catColor(r.Severity)).Render(tag)
				}
				fmt.Printf("  %s  %-22s %s\n", tag, r.ID, r.Summary)
			}
			fmt.Printf("\nSeverity decides what interrupts your shell (default: danger and above).\n")
			fmt.Printf("Tune it in %s.\n", "`oops-guard config`")
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output the catalog as JSON")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	return cmd
}

func catColor(s analyzer.Severity) lipgloss.Color {
	switch s {
	case analyzer.Critical:
		return lipgloss.Color("196")
	case analyzer.Danger:
		return lipgloss.Color("203")
	case analyzer.Caution:
		return lipgloss.Color("214")
	default:
		return lipgloss.Color("42")
	}
}
