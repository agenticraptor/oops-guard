package analyzer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Limits keep the impact preview fast enough to run inside a shell hook: the
// walk stops early and reports a floor ("≥ N") rather than ever stalling your
// prompt.
const (
	maxWalkFiles = 20000
	walkBudget   = 400 * time.Millisecond
	gitTimeout   = 1000 * time.Millisecond
)

// enrich fills in the precise impact preview for findings that destroy files or
// git state. It only ever reads — it never modifies anything — and degrades to
// the generic finding if the filesystem or git can't be inspected.
func enrich(findings []Finding, cwd string) {
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	for i := range findings {
		f := &findings[i]
		switch {
		case strings.HasPrefix(f.Rule, "rm") || f.Rule == "shred":
			for _, t := range f.Targets {
				f.Impact = append(f.Impact, previewPath(t, cwd))
			}
		case f.Rule == "git-reset-hard":
			if s := gitStatusCount(cwd); s != "" {
				f.Impact = append(f.Impact, TargetImpact{Path: "working tree", Git: s})
			}
		case f.Rule == "git-clean":
			if n, ok := gitCleanCount(cwd); ok {
				f.Impact = append(f.Impact, TargetImpact{Path: "untracked files", Files: n, IsDir: true,
					Git: fmt.Sprintf("%d path%s would be removed", n, plural(n))})
			}
		}
	}
}

// previewPath inspects a single delete target and returns what it holds.
func previewPath(raw, cwd string) TargetImpact {
	ti := TargetImpact{Path: raw}
	if catastrophicTarget(raw) != "" {
		ti.Note = "catastrophic scope — not measured"
		return ti
	}
	if strings.ContainsAny(raw, "*?[") {
		ti.Note = "wildcard — matches whatever is there now"
		return ti
	}
	p := resolvePath(raw, cwd)
	fi, err := os.Lstat(p)
	if err != nil {
		ti.Note = "not found — nothing to delete here"
		return ti
	}
	ti.Exists = true
	if fi.Mode()&os.ModeSymlink != 0 {
		ti.Note = "symlink — removes the link, not its target"
		return ti
	}
	if !under(p, cwd) {
		ti.Note = "outside the current directory"
	}
	if fi.IsDir() {
		ti.IsDir = true
		ti.Files, ti.Bytes, ti.Capped = walkDir(p)
		if g := gitStatusCount(p); g != "" {
			ti.Git = g
		}
	} else {
		ti.Files, ti.Bytes = 1, fi.Size()
	}
	return ti
}

func resolvePath(raw, cwd string) string {
	p := raw
	switch {
	case p == "~":
		p = home()
	case strings.HasPrefix(p, "~/"):
		p = filepath.Join(home(), p[2:])
	}
	p = strings.ReplaceAll(p, "${HOME}", home())
	p = strings.ReplaceAll(p, "$HOME", home())
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	return filepath.Clean(p)
}

func home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

// under reports whether p is at or below dir.
func under(p, dir string) bool {
	rel, err := filepath.Rel(dir, p)
	if err != nil {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}

// walkDir counts files and bytes beneath root, stopping at the safety limits.
func walkDir(root string) (files int, total int64, capped bool) {
	deadline := time.Now().Add(walkBudget)
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip what we cannot read; never abort the whole walk
		}
		if files >= maxWalkFiles || time.Now().After(deadline) {
			capped = true
			return filepath.SkipAll
		}
		if d.IsDir() {
			return nil
		}
		files++
		if info, e := d.Info(); e == nil {
			total += info.Size()
		}
		return nil
	})
	return files, total, capped
}

func gitStatusCount(dir string) string {
	out, err := gitRun(dir, "status", "--porcelain")
	if err != nil {
		return "" // not a git repo, or git missing — stay quiet
	}
	n := nonEmptyLines(out)
	if n == 0 {
		return "git repo, nothing uncommitted"
	}
	return fmt.Sprintf("git repo with %d uncommitted change%s", n, plural(n))
}

func gitCleanCount(dir string) (int, bool) {
	out, err := gitRun(dir, "clean", "-nd")
	if err != nil {
		return 0, false
	}
	return nonEmptyLines(out), true
}

func gitRun(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	// SECURITY: oops-guard inspects directories the user merely *named* (e.g. an
	// rm target), which may be an untrusted, attacker-supplied repository. Git
	// executes certain config keys on read — notably core.fsmonitor — so a
	// malicious .git/config could run code just because we ran `git status`
	// there. We neutralize those vectors: command-line -c overrides any repo
	// config, and the environment disables prompts, optional locks, and external
	// helpers. We never run git against a remote here, only local read-only
	// queries (status / clean -n).
	full := append([]string{
		"-c", "core.fsmonitor=",
		"-c", "core.hooksPath=/dev/null",
		"-c", "core.pager=cat",
		"-C", dir,
	}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(os.Environ(),
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_PAGER=cat",
		"GIT_ALLOW_PROTOCOL=file:builtin",
	)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func nonEmptyLines(s string) int {
	n := 0
	for _, ln := range strings.Split(s, "\n") {
		if strings.TrimSpace(ln) != "" {
			n++
		}
	}
	return n
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// HumanBytes renders a byte count in friendly units (used by renderers).
func HumanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
