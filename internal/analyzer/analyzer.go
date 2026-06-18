// Package analyzer decides whether a shell command line is destructive. It is
// the heart of oops-guard: a fast, deterministic, dependency-free classifier
// that flags genuinely dangerous commands (rm -rf, force-push, DROP TABLE, dd,
// mkfs, …) and — crucially — explains *what specifically* would be lost, rather
// than nagging on everything. It never needs a network or a model.
package analyzer

import (
	"sort"
	"strings"
)

// Finding is one reason a command line was flagged.
type Finding struct {
	Rule     string         `json:"rule"`     // stable identifier, e.g. "rm-recursive"
	Severity Severity       `json:"severity"` // how bad it is
	Title    string         `json:"title"`    // short headline, e.g. "Recursive force-delete"
	Detail   string         `json:"detail"`   // one or two plain-English sentences
	Command  string         `json:"command"`  // the simple command this came from
	Targets  []string       `json:"targets,omitempty"`
	Impact   []TargetImpact `json:"impact,omitempty"` // precise preview of what would be lost
}

// TargetImpact is a precise, read-only preview of what a single target holds —
// the "what exactly will be lost" that separates oops-guard from a blind
// confirm-everything prompt.
type TargetImpact struct {
	Path   string `json:"path"`           // the target as written
	Exists bool   `json:"exists"`         // whether it resolves to something on disk
	IsDir  bool   `json:"is_dir"`         // directory vs file
	Files  int    `json:"files"`          // files counted under the target
	Bytes  int64  `json:"bytes"`          // total size counted
	Capped bool   `json:"capped"`         // the walk hit its safety limit (counts are a floor)
	Git    string `json:"git,omitempty"`  // git state worth knowing, e.g. "3 uncommitted changes"
	Note   string `json:"note,omitempty"` // extra warning, e.g. "outside the current directory"
}

// Assessment is the verdict for a whole command line.
type Assessment struct {
	Input    string    `json:"input"`
	Findings []Finding `json:"findings"`
}

// Max returns the highest severity among the findings (Safe if there are none).
func (a Assessment) Max() Severity {
	m := Safe
	for _, f := range a.Findings {
		if f.Severity > m {
			m = f.Severity
		}
	}
	return m
}

// Dangerous reports whether the assessment is at or above the given threshold.
func (a Assessment) Dangerous(threshold Severity) bool {
	return len(a.Findings) > 0 && a.Max() >= threshold
}

// Options configures a single analysis.
type Options struct {
	// Cwd is the directory the command would run in. Empty means the process's
	// current directory. It anchors relative paths in the impact preview.
	Cwd string
	// NoImpact disables filesystem/git inspection, keeping analysis pure and
	// instant (used in tests and for untrusted input).
	NoImpact bool
}

// Analyze classifies a command line and returns its assessment. With the
// default options it also enriches findings with a precise impact preview
// (file counts, sizes, and git state) by reading — never modifying — the
// filesystem.
func Analyze(line string, opts Options) Assessment {
	a := Assessment{Input: line}
	pl := Parse(line)

	for _, cmd := range pl.Commands {
		for _, rule := range commandRules {
			a.Findings = append(a.Findings, rule(cmd)...)
		}
	}
	for _, rule := range pipelineRules {
		a.Findings = append(a.Findings, rule(pl)...)
	}

	a.Findings = dedupe(a.Findings)
	if !opts.NoImpact {
		enrich(a.Findings, opts.Cwd)
	}
	sortFindings(a.Findings)
	return a
}

// commandRule inspects a single simple command and returns any findings.
type commandRule func(Command) []Finding

// pipelineRule inspects the whole pipeline (for cross-command patterns such as
// piping a download straight into a shell).
type pipelineRule func(Pipeline) []Finding

func dedupe(in []Finding) []Finding {
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, f := range in {
		key := f.Rule + "\x00" + f.Command + "\x00" + strings.Join(f.Targets, ",")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, f)
	}
	return out
}

func sortFindings(f []Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		return f[i].Severity > f[j].Severity
	})
}
