package analyzer

import "testing"

func TestCatastrophicTarget(t *testing.T) {
	yes := map[string]string{
		"/":       "the entire filesystem (/)",
		"/*":      "the entire filesystem (/)",
		"~":       "your home directory",
		"~/":      "your home directory",
		"$HOME":   "your home directory",
		"${HOME}": "your home directory",
		"/etc":    "the system directory /etc",
		"/usr/":   "the system directory /usr",
	}
	for in, want := range yes {
		if got := catastrophicTarget(in); got != want {
			t.Errorf("catastrophicTarget(%q) = %q, want %q", in, got, want)
		}
	}
	for _, in := range []string{".", "..", "*", "build", "./dist", "~/projects/app", "/etc/nginx"} {
		if got := catastrophicTarget(in); got != "" {
			t.Errorf("catastrophicTarget(%q) = %q, want empty", in, got)
		}
	}
}

func TestIsBlockDevice(t *testing.T) {
	devs := []string{"/dev/sda", "/dev/sda1", "/dev/nvme0n1", "/dev/nvme0n1p2", "/dev/disk2", "/dev/mmcblk0", "/dev/vdb"}
	for _, d := range devs {
		if !isBlockDevice(d) {
			t.Errorf("isBlockDevice(%q) = false, want true", d)
		}
	}
	notDevs := []string{"/dev/null", "/dev/random", "disk.img", "/home/me/file", "/devices/x"}
	for _, d := range notDevs {
		if isBlockDevice(d) {
			t.Errorf("isBlockDevice(%q) = true, want false", d)
		}
	}
}

func TestIsSystemDir(t *testing.T) {
	if !isSystemDir("/etc/nginx") {
		t.Error("/etc/nginx should be a system dir")
	}
	if isSystemDir("project/etc") {
		t.Error("project/etc should not be a system dir")
	}
}

func TestHumanList(t *testing.T) {
	if got := humanList([]string{"a"}); got != "a" {
		t.Errorf("got %q", got)
	}
	if got := humanList([]string{"a", "b"}); got != "a and b" {
		t.Errorf("got %q", got)
	}
	if got := humanList([]string{"a", "b", "c"}); got != "a, b, and c" {
		t.Errorf("got %q", got)
	}
}

func TestParseSeverityRoundTrip(t *testing.T) {
	for _, s := range []Severity{Safe, Caution, Danger, Critical} {
		if got := ParseSeverity(s.String()); got != s {
			t.Errorf("ParseSeverity(%q) = %v, want %v", s.String(), got, s)
		}
	}
	if ParseSeverity("nonsense") != Danger {
		t.Error("unknown severity should default to Danger")
	}
}
