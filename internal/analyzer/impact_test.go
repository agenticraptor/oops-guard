package analyzer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewPathCountsTree(t *testing.T) {
	dir := t.TempDir()
	// 3 files, one in a subdirectory.
	mustWrite(t, filepath.Join(dir, "a.txt"), "hello")
	mustWrite(t, filepath.Join(dir, "b.txt"), "world!!")
	sub := filepath.Join(dir, "nested")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(sub, "c.txt"), "x")

	ti := previewPath("target", filepath.Dir(dir)) // resolve relative to parent
	_ = ti                                         // (relative form covered below)

	got := previewPath(dir, dir)
	if !got.Exists || !got.IsDir {
		t.Fatalf("expected an existing directory, got %+v", got)
	}
	if got.Files != 3 {
		t.Errorf("Files = %d, want 3", got.Files)
	}
	if got.Bytes != int64(len("hello")+len("world!!")+len("x")) {
		t.Errorf("Bytes = %d, want %d", got.Bytes, len("hello")+len("world!!")+len("x"))
	}
}

func TestPreviewPathMissing(t *testing.T) {
	ti := previewPath("does-not-exist", t.TempDir())
	if ti.Exists {
		t.Errorf("missing path should not exist: %+v", ti)
	}
	if ti.Note == "" {
		t.Error("missing path should carry an explanatory note")
	}
}

func TestPreviewPathCatastrophicNotWalked(t *testing.T) {
	ti := previewPath("/", t.TempDir())
	if ti.Files != 0 || ti.Note == "" {
		t.Errorf("catastrophic target must not be walked, got %+v", ti)
	}
}

func TestPreviewPathWildcardNotWalked(t *testing.T) {
	ti := previewPath("*.go", t.TempDir())
	if ti.Exists || ti.Note == "" {
		t.Errorf("wildcard target should be annotated, not walked, got %+v", ti)
	}
}

func TestEnrichAttachesImpact(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "f1"), "12345")
	a := Analyze("rm -rf "+dir, Options{Cwd: dir})
	var found bool
	for _, f := range a.Findings {
		if len(f.Impact) > 0 && f.Impact[0].Files == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected enrichment to attach a file count, got %+v", a.Findings)
	}
}

func TestUnder(t *testing.T) {
	if !under("/a/b/c", "/a/b") {
		t.Error("/a/b/c should be under /a/b")
	}
	if under("/a/x", "/a/b") {
		t.Error("/a/x should not be under /a/b")
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{0: "0 B", 512: "512 B", 1024: "1.0 KB", 1572864: "1.5 MB"}
	for in, want := range cases {
		if got := HumanBytes(in); got != want {
			t.Errorf("HumanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}

// TestGitStatusCountHardened confirms the security-hardened git invocation
// still functions on a real repository (the -c overrides and restricted env
// must not break the read-only queries we rely on).
func TestGitStatusCountHardened(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	gitEnv := append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Env = gitEnv
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed: %v\n%s", err, out)
	}
	mustWrite(t, filepath.Join(dir, "new.txt"), "hi")

	if s := gitStatusCount(dir); !strings.Contains(s, "uncommitted") {
		t.Errorf("expected uncommitted changes from hardened git, got %q", s)
	}
	if _, ok := gitCleanCount(dir); !ok {
		t.Error("expected git clean -n to succeed under the hardened invocation")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
