package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/agenticraptor/oops-guard/internal/analyzer"
)

func sample() analyzer.Assessment {
	return analyzer.Assessment{
		Input: "rm -rf build/",
		Findings: []analyzer.Finding{{
			Rule:     "rm-recursive",
			Severity: analyzer.Danger,
			Title:    "Recursive force-delete",
			Detail:   "Permanently removes build/.",
			Targets:  []string{"build/"},
			Impact:   []analyzer.TargetImpact{{Path: "build/", Exists: true, IsDir: true, Files: 812, Bytes: 1153433}},
		}},
	}
}

func TestRenderPlainText(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sample(), Options{Format: FormatPlain, Color: false}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"oops-guard", "DANGER", "Recursive force-delete", "812 files", "build/"} {
		if !strings.Contains(out, want) {
			t.Errorf("plain output missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("plain output must not contain ANSI escapes:\n%s", out)
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sample(), Options{Format: FormatJSON}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{`"max_severity": "danger"`, `"dangerous": true`, `"rule": "rm-recursive"`, `"severity": "danger"`} {
		if !strings.Contains(out, want) {
			t.Errorf("json output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderSafe(t *testing.T) {
	var buf bytes.Buffer
	a := analyzer.Assessment{Input: "ls -la"}
	if err := Render(&buf, a, Options{Format: FormatPlain}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "nothing destructive") {
		t.Errorf("expected a clean bill of health, got:\n%s", buf.String())
	}
}

func TestWrap(t *testing.T) {
	const width = 24 // at or above the renderer's minimum wrap width
	lines := wrap("one two three four five six seven eight nine ten", width)
	if len(lines) < 2 {
		t.Fatalf("expected wrapping into multiple lines, got %v", lines)
	}
	for _, ln := range lines {
		if len(ln) > width {
			t.Errorf("line exceeds width %d: %q", width, ln)
		}
	}
}
