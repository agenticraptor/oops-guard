package config

import (
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	if Default().MinSeverity != "danger" {
		t.Errorf("default min_severity = %q, want danger", Default().MinSeverity)
	}
}

func TestAllowed(t *testing.T) {
	cfg := Config{Allow: []string{`^bin/clean\b`, `rm -rf .*\.cache`}}
	if !cfg.Allowed("bin/clean --all") {
		t.Error("expected the allow-list to match bin/clean")
	}
	if !cfg.Allowed("rm -rf ~/.cache") {
		t.Error("expected the allow-list to match the cache pattern")
	}
	if cfg.Allowed("rm -rf /etc") {
		t.Error("did not expect /etc to be allow-listed")
	}
}

func TestAllowedIgnoresBadPattern(t *testing.T) {
	cfg := Config{Allow: []string{"("}} // invalid regex
	if cfg.Allowed("anything") {
		t.Error("an invalid pattern should never match")
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	t.Setenv("OOPS_GUARD_CONFIG", filepath.Join(t.TempDir(), "nope.toml"))
	cfg, err := Load()
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cfg.MinSeverity != "danger" {
		t.Errorf("want default threshold, got %q", cfg.MinSeverity)
	}
}

func TestWriteStarterThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("OOPS_GUARD_CONFIG", path)

	got, err := WriteStarter()
	if err != nil {
		t.Fatalf("WriteStarter: %v", err)
	}
	if got != path {
		t.Errorf("WriteStarter wrote %q, want %q", got, path)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load after write: %v", err)
	}
	if cfg.MinSeverity != "danger" {
		t.Errorf("starter min_severity = %q, want danger", cfg.MinSeverity)
	}
	if cfg.AI.Enabled {
		t.Error("starter ai.enabled should be false")
	}
	// Writing again must refuse to clobber the existing file.
	if _, err := WriteStarter(); err == nil {
		t.Error("expected an error writing over an existing config")
	}
}
