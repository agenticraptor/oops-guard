// Package render turns an analyzer.Assessment into something a human (or a
// script) can read: a colored, boxed terminal report, plain text, or JSON.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/agenticraptor/oops-guard/internal/analyzer"
)

// Format selects the output style.
type Format string

// Supported formats.
const (
	FormatTerm  Format = "term"
	FormatPlain Format = "plain"
	FormatJSON  Format = "json"
)

// Options configures a render.
type Options struct {
	Format Format
	Color  bool
	Width  int // wrap width for detail text; 0 picks a sensible default
}

// Render writes the assessment to w in the requested format.
func Render(w io.Writer, a analyzer.Assessment, opts Options) error {
	switch opts.Format {
	case FormatJSON:
		return renderJSON(w, a)
	default:
		return renderText(w, a, opts)
	}
}

func renderJSON(w io.Writer, a analyzer.Assessment) error {
	type out struct {
		analyzer.Assessment
		MaxSeverity analyzer.Severity `json:"max_severity"`
		Dangerous   bool              `json:"dangerous"`
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out{Assessment: a, MaxSeverity: a.Max(), Dangerous: a.Dangerous(analyzer.Danger)})
}

func renderText(w io.Writer, a analyzer.Assessment, opts Options) error {
	width := opts.Width
	if width <= 0 {
		width = 76
	}
	p := painter{color: opts.Color}
	var b strings.Builder

	if len(a.Findings) == 0 {
		b.WriteString(p.ok("✓ oops-guard") + "  nothing destructive detected\n")
		if a.Input != "" {
			b.WriteString("  " + p.dim(a.Input) + "\n")
		}
		_, err := io.WriteString(w, b.String())
		return err
	}

	b.WriteString(p.sev(a.Max(), p.icon(a.Max())+" oops-guard") + "  this looks destructive\n\n")
	if a.Input != "" {
		b.WriteString("  " + p.bold(a.Input) + "\n\n")
	}

	for _, f := range a.Findings {
		tag := p.sev(f.Severity, fmt.Sprintf("%s %-8s", p.icon(f.Severity), f.Severity.Label()))
		b.WriteString("  " + tag + "  " + p.bold(f.Title) + "\n")
		for _, ln := range wrap(f.Detail, width-5) {
			b.WriteString("     " + ln + "\n")
		}
		for _, ti := range f.Impact {
			b.WriteString("     " + p.dim(impactLine(ti)) + "\n")
		}
		b.WriteString("\n")
	}

	_, err := io.WriteString(w, strings.TrimRight(b.String(), "\n")+"\n")
	return err
}

// impactLine renders one TargetImpact as a compact, human line.
func impactLine(ti analyzer.TargetImpact) string {
	var parts []string
	if ti.Exists && ti.Files > 0 {
		count := fmt.Sprintf("%d file%s", ti.Files, plural(ti.Files))
		if ti.Capped {
			count = "≥ " + count
		}
		parts = append(parts, count)
		if ti.Bytes > 0 {
			parts = append(parts, analyzer.HumanBytes(ti.Bytes))
		}
	}
	if ti.Git != "" {
		parts = append(parts, ti.Git)
	}
	if ti.Note != "" {
		parts = append(parts, ti.Note)
	}
	label := ti.Path
	if label == "" {
		label = "(target)"
	}
	if len(parts) == 0 {
		return label
	}
	return fmt.Sprintf("%s — %s", label, strings.Join(parts, " · "))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// wrap breaks text into lines no longer than width, on word boundaries.
func wrap(text string, width int) []string {
	if width < 20 {
		width = 20
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := words[0]
	for _, word := range words[1:] {
		if len(cur)+1+len(word) > width {
			lines = append(lines, cur)
			cur = word
			continue
		}
		cur += " " + word
	}
	return append(lines, cur)
}

// painter applies (or skips) ANSI styling.
type painter struct{ color bool }

func (p painter) style(s lipgloss.Style, text string) string {
	if !p.color {
		return text
	}
	return s.Render(text)
}

func (p painter) bold(s string) string { return p.style(lipgloss.NewStyle().Bold(true), s) }
func (p painter) dim(s string) string {
	return p.style(lipgloss.NewStyle().Foreground(lipgloss.Color("245")), s)
}
func (p painter) ok(s string) string {
	return p.style(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")), s)
}

func (p painter) sev(s analyzer.Severity, text string) string {
	return p.style(lipgloss.NewStyle().Bold(true).Foreground(sevColor(s)), text)
}

func (p painter) icon(s analyzer.Severity) string {
	switch s {
	case analyzer.Critical:
		return "☢"
	case analyzer.Danger:
		return "✖"
	case analyzer.Caution:
		return "▲"
	default:
		return "✓"
	}
}

func sevColor(s analyzer.Severity) lipgloss.Color {
	switch s {
	case analyzer.Critical:
		return lipgloss.Color("196") // bright red
	case analyzer.Danger:
		return lipgloss.Color("203") // red-orange
	case analyzer.Caution:
		return lipgloss.Color("214") // amber
	default:
		return lipgloss.Color("42") // green
	}
}
