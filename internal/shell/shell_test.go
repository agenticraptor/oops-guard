package shell

import (
	"strings"
	"testing"
)

func TestSnippetsPresentAndInvokeGuard(t *testing.T) {
	for _, sh := range Supported() {
		snip, err := Snippet(sh)
		if err != nil {
			t.Fatalf("Snippet(%q) error: %v", sh, err)
		}
		if !strings.Contains(snip, "oops-guard guard --shell "+sh) {
			t.Errorf("%s snippet does not invoke the guard for its shell:\n%s", sh, snip)
		}
		if !strings.Contains(snip, "_OOPS_GUARD") {
			t.Errorf("%s snippet missing a re-entrancy guard", sh)
		}
	}
}

// TestSnippetsFailOpen verifies the safety-critical contract: every hook must
// (a) only block on the explicit decline exit code 10, and (b) skip the guard
// entirely when the binary isn't installed — so oops-guard can never brick a
// shell.
func TestSnippetsFailOpen(t *testing.T) {
	existenceCheck := map[string]string{
		"bash": "command -v oops-guard",
		"zsh":  "$+commands[oops-guard]",
		"fish": "command -q oops-guard",
	}
	for _, sh := range Supported() {
		snip, err := Snippet(sh)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(snip, "10") {
			t.Errorf("%s snippet must block only on the decline exit code 10", sh)
		}
		if !strings.Contains(snip, existenceCheck[sh]) {
			t.Errorf("%s snippet must check the binary exists before invoking it (want %q)", sh, existenceCheck[sh])
		}
	}
}

func TestSnippetUnknownShell(t *testing.T) {
	if _, err := Snippet("powershell"); err == nil {
		t.Error("expected an error for an unsupported shell")
	}
}

func TestSupportedSorted(t *testing.T) {
	got := Supported()
	want := []string{"bash", "fish", "zsh"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Supported() = %v, want %v", got, want)
	}
}
