package analyzer

import (
	"regexp"
	"strings"
)

// catastrophicTarget returns a human label when a path refers to the whole
// filesystem, the user's home, or a top-level system directory — the targets
// where a recursive delete is catastrophic rather than merely dangerous. It
// returns "" for ordinary paths (including "." and "*", which are dangerous but
// scoped and handled with a precise impact preview instead).
func catastrophicTarget(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "/" || raw == "/*" {
		return "the entire filesystem (/)"
	}
	// Normalize a trailing "/" or "/*" so "~/", "~/*" and "$HOME/" all collapse.
	t := strings.TrimSuffix(raw, "*")
	t = strings.TrimRight(t, "/")
	switch t {
	case "~", "$HOME", "${HOME}":
		return "your home directory"
	}
	for _, d := range systemRoots {
		if t == d {
			return "the system directory " + d
		}
	}
	return ""
}

var systemRoots = []string{
	"/etc", "/usr", "/var", "/bin", "/sbin", "/lib", "/lib64", "/boot",
	"/opt", "/root", "/home", "/Users", "/System", "/Library", "/Applications",
	"/private", "/dev", "/proc", "/sys",
}

// isSystemDir reports whether a path is, or lives directly under, a top-level
// system directory — used to escalate chmod/chown.
func isSystemDir(raw string) bool {
	t := "/" + strings.Trim(strings.TrimSpace(raw), "/")
	for _, d := range systemRoots {
		if t == d || strings.HasPrefix(t, d+"/") {
			return true
		}
	}
	return false
}

var blockDevice = regexp.MustCompile(`^(/dev/(sd[a-z]|nvme\d+n\d+|mmcblk\d+|vd[a-z]|hd[a-z]|loop\d+|x?vd[a-z]|r?disk\d+)(p?\d+)?|\\\\\.\\PhysicalDrive\d+)$`)

// isBlockDevice reports whether a path looks like a raw disk/partition device.
func isBlockDevice(p string) bool {
	return blockDevice.MatchString(strings.TrimSpace(p))
}

// hasGlob reports whether any target contains a shell wildcard.
func hasGlob(targets []string) bool {
	for _, t := range targets {
		if strings.ContainsAny(t, "*?[") {
			return true
		}
	}
	return false
}

// humanList renders a small set of items as readable English.
func humanList(items []string) string {
	switch len(items) {
	case 0:
		return "the target"
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}
