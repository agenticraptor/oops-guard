package analyzer

import (
	"reflect"
	"testing"
)

func TestParseSplitsOperators(t *testing.T) {
	pl := Parse("a foo && b | c ; d")
	if len(pl.Commands) != 4 {
		t.Fatalf("want 4 commands, got %d: %+v", len(pl.Commands), pl.Commands)
	}
	if !pl.Piped {
		t.Error("expected Piped to be true")
	}
	got := []string{pl.Commands[0].Name(), pl.Commands[1].Name(), pl.Commands[2].Name(), pl.Commands[3].Name()}
	want := []string{"a", "b", "c", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
}

func TestParseQuotes(t *testing.T) {
	pl := Parse(`rm -rf "my dir" 'other dir'`)
	if len(pl.Commands) != 1 {
		t.Fatalf("want 1 command, got %d", len(pl.Commands))
	}
	want := []string{"rm", "-rf", "my dir", "other dir"}
	if !reflect.DeepEqual(pl.Commands[0].Args, want) {
		t.Errorf("args = %v, want %v", pl.Commands[0].Args, want)
	}
}

func TestParseRedirectToken(t *testing.T) {
	pl := Parse("cat x > /dev/sda")
	args := pl.Commands[0].Args
	if !contains(args, ">") {
		t.Errorf("expected '>' preserved as a token, got %v", args)
	}
}

func TestCommandNameSkipsPrefixes(t *testing.T) {
	cases := map[string]string{
		"sudo rm -rf /":       "rm",
		"FOO=bar rm file":     "rm",
		"env X=1 git push":    "git",
		"/usr/bin/rm -rf x":   "rm",
		"sudo -u root rm x":   "rm",
		"time make":           "make",
		"command rm -rf x":    "rm",
		"nohup node server":   "node",
		"nice -n 19 rm -rf x": "rm",
		"timeout 5 rm -rf x":  "rm",
		"timeout 30s git gc":  "git",
		"xargs rm -rf":        "rm",
		"ionice -c 3 rm x":    "rm",
	}
	for line, want := range cases {
		if got := Parse(line).Commands[0].Name(); got != want {
			t.Errorf("Name(%q) = %q, want %q", line, got, want)
		}
	}
}

func TestParseEmpty(t *testing.T) {
	if pl := Parse("   "); len(pl.Commands) != 0 {
		t.Errorf("blank line should yield no commands, got %+v", pl.Commands)
	}
}
