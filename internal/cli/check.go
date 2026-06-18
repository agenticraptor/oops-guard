package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/oops-guard/internal/analyzer"
	"github.com/agenticraptor/oops-guard/internal/render"
)

func newCheckCmd() *cobra.Command {
	var (
		format   string
		noColor  bool
		exitCode bool
		noImpact bool
	)
	cmd := &cobra.Command{
		Use:   "check [command...]",
		Short: "Analyze a command and report what's dangerous about it",
		Long: `check classifies a command line and prints what — if anything — is
destructive about it, including a precise preview of what would be lost. It
reads your filesystem to build that preview but never changes anything, and it
never runs the command.

Quote the command, or pass it after the options:

  oops-guard check 'rm -rf build dist'
  oops-guard check --format json 'git push --force origin main'`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := parseFormat(format)
			if err != nil {
				return err
			}
			line := strings.Join(args, " ")
			a := analyzer.Analyze(line, analyzer.Options{NoImpact: noImpact})
			if err := render.Render(os.Stdout, a, render.Options{
				Format: f,
				Color:  useColor(noColor, os.Stdout),
			}); err != nil {
				return err
			}
			if exitCode && a.Dangerous(analyzer.Danger) {
				return exitCodeError{2}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "term", "output format: term, plain, or json")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable colored output")
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "exit 2 if the command is dangerous (for scripts)")
	cmd.Flags().BoolVar(&noImpact, "no-impact", false, "skip the filesystem/git impact preview")
	cmd.Flags().SetInterspersed(false)
	return cmd
}
