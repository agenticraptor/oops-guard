// Package shell serves the embedded shell-integration snippets used by
// `oops-guard init <shell>`. Each snippet hooks the shell so a destructive
// command is inspected before it runs.
package shell

import (
	"embed"
	"fmt"
	"sort"
)

//go:embed hooks/*
var hooks embed.FS

var files = map[string]string{
	"bash": "hooks/oops-guard.bash",
	"zsh":  "hooks/oops-guard.zsh",
	"fish": "hooks/oops-guard.fish",
}

// Supported returns the shells oops-guard can hook, sorted.
func Supported() []string {
	out := make([]string, 0, len(files))
	for k := range files {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Snippet returns the integration script for the named shell. The result is
// meant to be evaluated by that shell (e.g. eval "$(oops-guard init bash)" or,
// for fish, oops-guard init fish | source).
func Snippet(shell string) (string, error) {
	path, ok := files[shell]
	if !ok {
		return "", fmt.Errorf("unsupported shell %q (supported: %v)", shell, Supported())
	}
	data, err := hooks.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Instructions returns the one-line install snippet the user should add to
// their shell's startup file.
func Instructions(shell string) string {
	switch shell {
	case "fish":
		return "oops-guard init fish | source   # add to ~/.config/fish/config.fish"
	case "zsh":
		return `eval "$(oops-guard init zsh)"   # add to ~/.zshrc`
	default:
		return `eval "$(oops-guard init bash)"   # add to ~/.bashrc`
	}
}
