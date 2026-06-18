package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/oops-guard/internal/buildinfo"
	"github.com/agenticraptor/oops-guard/internal/config"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check your environment and configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			runDoctor(cmd.Context())
			return nil
		},
	}
}

func runDoctor(ctx context.Context) {
	ok := func(label, detail string) { fmt.Printf("  \033[32m✓\033[0m %-20s %s\n", label, detail) }
	warn := func(label, detail string) { fmt.Printf("  \033[33m!\033[0m %-20s %s\n", label, detail) }

	fmt.Print("oops-guard doctor\n\n")
	ok("version", buildinfo.String())

	// Shell detection + whether the hook is likely installed.
	shell := filepath.Base(os.Getenv("SHELL"))
	if shell == "" || shell == "." {
		warn("shell", "could not detect $SHELL")
	} else {
		ok("shell", shell)
	}
	if rc, hooked := hookInstalled(shell); hooked {
		ok("hook", "installed in "+rc)
	} else {
		want := "oops-guard init " + nonEmpty(shell, "zsh")
		warn("hook", "not detected — add: eval \"$("+want+")\"")
	}

	// git is needed for the richest impact previews (uncommitted-change counts).
	if out, err := exec.CommandContext(ctx, "git", "--version").Output(); err == nil {
		ok("git", strings.TrimSpace(string(out)))
	} else {
		warn("git", "not found — git-aware previews will be skipped")
	}

	// Config.
	path := config.Path()
	if _, err := os.Stat(path); err == nil {
		cfg, err := config.Load()
		if err != nil {
			warn("config", fmt.Sprintf("%s (parse error: %v)", path, err))
		} else {
			ok("config", fmt.Sprintf("%s (threshold: %s)", path, cfg.MinSeverity))
		}
	} else {
		ok("config", "using defaults (threshold: danger) — run `oops-guard config init` to customize")
	}

	// Model availability (only matters for `explain`).
	switch {
	case os.Getenv("ANTHROPIC_API_KEY") != "":
		ok("model", "Anthropic key detected (used only by `explain`)")
	case os.Getenv("OPENAI_API_KEY") != "":
		ok("model", "OpenAI key detected (used only by `explain`)")
	default:
		warn("model", "no API key — `explain` will try a local Ollama; the guard never needs a model")
	}
}

// hookInstalled does a best-effort scan of the shell rc file for the snippet.
func hookInstalled(shell string) (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	candidates := map[string][]string{
		"zsh":  {".zshrc"},
		"bash": {".bashrc", ".bash_profile"},
		"fish": {".config/fish/config.fish"},
	}
	for _, rel := range candidates[shell] {
		p := filepath.Join(home, rel)
		data, err := os.ReadFile(p)
		if err == nil && strings.Contains(string(data), "oops-guard init") {
			return p, true
		}
	}
	return "", false
}

func nonEmpty(s, fallback string) string {
	if s == "" || s == "." {
		return fallback
	}
	return s
}
