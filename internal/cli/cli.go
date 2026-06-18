// Package cli wires the oops-guard command-line interface together.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/agenticraptor/oops-guard/internal/buildinfo"
	"github.com/agenticraptor/oops-guard/internal/render"
)

// declineExitCode is the exit status `guard` returns when the user explicitly
// declines a command. The shell hooks block a command ONLY on this exact code,
// so that a missing binary, a parse error, or any other failure can never block
// your shell — oops-guard always fails open.
const declineExitCode = 10

// exitCodeError lets a leaf command request a specific process exit code
// without printing an "error:" line (its message is empty).
type exitCodeError struct{ code int }

func (e exitCodeError) Error() string { return "" }

// Execute runs the root command and returns a process exit code.
func Execute() int {
	if err := newRootCmd().Execute(); err != nil {
		var ec exitCodeError
		if errors.As(err, &ec) {
			return ec.code
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oops-guard",
		Short: "Catch a dangerous command before you hit enter",
		Long: `oops-guard is a small shell guard. It watches the command you're about to
run, and when it spots something genuinely destructive — rm -rf, a force-push
to main, DROP TABLE, dd to a disk — it stops and tells you exactly what would
be lost before letting you continue. Everyday commands pass straight through;
this is not a confirm-everything nag.

Get started:

  oops-guard check 'rm -rf build'     see what oops-guard thinks of a command
  oops-guard init zsh                 print the shell hook to install
  oops-guard rules                    list everything it watches for

Install the hook (bash/zsh):

  eval "$(oops-guard init zsh)"       # add to ~/.zshrc`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buildinfo.Version,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")
	cmd.AddCommand(
		newCheckCmd(),
		newGuardCmd(),
		newInitCmd(),
		newExplainCmd(),
		newRulesCmd(),
		newDoctorCmd(),
		newConfigCmd(),
		newVersionCmd(),
	)
	return cmd
}

// useColor reports whether styled output should be used for the given file.
func useColor(noColor bool, f *os.File) bool {
	if noColor || os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// parseFormat validates a --format value.
func parseFormat(s string) (render.Format, error) {
	switch render.Format(s) {
	case render.FormatTerm, render.FormatPlain, render.FormatJSON:
		return render.Format(s), nil
	default:
		return "", fmt.Errorf("invalid --format %q (want term, plain, or json)", s)
	}
}

// ttyWriter returns a writer aimed at the controlling terminal so the guard's
// warning is visible even when invoked from a shell hook with redirected
// stdio. It falls back to stderr.
func ttyWriter() (io.Writer, func()) {
	if f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		return f, func() { _ = f.Close() }
	}
	return os.Stderr, func() {}
}
