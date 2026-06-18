package cli

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// run executes the root command with args, capturing everything written to
// os.Stdout, and returns the output plus the command error.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	cmd := newRootCmd()
	cmd.SetArgs(args)
	runErr := cmd.Execute()

	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out), runErr
}

func TestCheckReportsCritical(t *testing.T) {
	out, err := run(t, "check", "--no-color", "sudo rm -rf /")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"CRITICAL", "filesystem"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestCheckSafeCommand(t *testing.T) {
	out, err := run(t, "check", "--no-color", "ls -la")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "nothing destructive") {
		t.Errorf("expected a clean report, got:\n%s", out)
	}
}

func TestCheckExitCodeFlag(t *testing.T) {
	_, err := run(t, "check", "--no-color", "--exit-code", "rm -rf /")
	var ec exitCodeError
	if !errors.As(err, &ec) || ec.code != 2 {
		t.Fatalf("expected exit code 2, got %v", err)
	}
}

func TestRulesLists(t *testing.T) {
	out, err := run(t, "rules", "--no-color")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"rm-catastrophic", "sql-drop-database", "fork-bomb"} {
		if !strings.Contains(out, want) {
			t.Errorf("rules output missing %q", want)
		}
	}
}

func TestInitEmitsHook(t *testing.T) {
	out, err := run(t, "init", "bash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "oops-guard guard --shell bash") {
		t.Errorf("init bash did not emit the guard hook:\n%s", out)
	}
}

func TestVersion(t *testing.T) {
	out, err := run(t, "version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "oops-guard") {
		t.Errorf("version output = %q", out)
	}
}
