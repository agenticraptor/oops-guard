// Package config loads oops-guard's optional configuration. Everything has a
// sensible default, so the tool works with no config file at all; the file
// exists only to let you tune the prompt threshold, allow-list commands you
// trust, and point `explain` at a model.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
)

// Config is the on-disk configuration.
type Config struct {
	// MinSeverity is the lowest severity at which the shell guard interrupts you
	// ("caution", "danger", or "critical"). Default: "danger".
	MinSeverity string `toml:"min_severity"`
	// Allow is a list of regular expressions; a command line matching any of
	// them is never flagged. Use it to silence a pattern you trust.
	Allow []string `toml:"allow"`
	// AI configures the optional model used by `oops-guard explain`.
	AI AI `toml:"ai"`
}

// AI holds the optional language-model settings used only by `explain`.
type AI struct {
	Enabled  bool   `toml:"enabled"`  // allow `explain` to call a model
	Provider string `toml:"provider"` // anthropic | openai | ollama (empty = auto-detect)
	Model    string `toml:"model"`    // empty = provider default
}

// Default returns the configuration used when no file is present.
func Default() Config {
	return Config{MinSeverity: "danger"}
}

// Path returns the configuration file path, honoring XDG_CONFIG_HOME.
func Path() string {
	if dir := os.Getenv("OOPS_GUARD_CONFIG"); dir != "" {
		return dir
	}
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "oops-guard", "config.toml")
}

// Load reads the configuration file, falling back to defaults for anything not
// set. A missing file is not an error.
func Load() (Config, error) {
	cfg := Default()
	path := Path()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.MinSeverity == "" {
		cfg.MinSeverity = "danger"
	}
	return cfg, nil
}

// Allowed reports whether the command line matches any allow-list pattern.
// Invalid patterns are ignored rather than failing the whole run.
func (c Config) Allowed(line string) bool {
	for _, pat := range c.Allow {
		if re, err := regexp.Compile(pat); err == nil && re.MatchString(line) {
			return true
		}
	}
	return false
}

const starter = `# oops-guard configuration
# Everything here is optional; delete this file to return to defaults.

# Interrupt your shell when a command is at least this dangerous.
# One of: "caution", "danger", "critical".
min_severity = "danger"

# Command lines matching any of these regular expressions are never flagged.
# Example: allow your own backup script that calls rm under the hood.
#   allow = ['^bin/clean-tmp\b']
allow = []

[ai]
# Used only by ` + "`oops-guard explain`" + `, never by the shell guard.
enabled  = false
provider = ""   # anthropic | openai | ollama (empty = auto-detect from env)
model    = ""   # empty = provider default
`

// WriteStarter creates a documented configuration file at Path if one does not
// already exist, returning the path written.
func WriteStarter() (string, error) {
	path := Path()
	if _, err := os.Stat(path); err == nil {
		return path, fmt.Errorf("config already exists at %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, err
	}
	if err := os.WriteFile(path, []byte(starter), 0o644); err != nil {
		return path, err
	}
	return path, nil
}
