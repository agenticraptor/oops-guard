package analyzer

import "strings"

// flags is a lenient parse of a command's argv into the short letters, long
// names, and positional arguments rules care about. It is not a full getopt: it
// deliberately accepts unknown flags and never errors, because its only job is
// to recognize danger.
type flags struct {
	short map[rune]bool
	long  map[string]bool
	pos   []string
}

// parseFlags parses words[1:] (argv without the program name). Combined short
// flags (-rf) are split into individual letters; --long[=value] flags record
// the name; everything after a bare "--" is positional.
func parseFlags(words []string) flags {
	f := flags{short: map[rune]bool{}, long: map[string]bool{}}
	end := false
	for i, w := range words {
		if i == 0 {
			continue
		}
		switch {
		case end:
			f.pos = append(f.pos, w)
		case w == "--":
			end = true
		case strings.HasPrefix(w, "--"):
			name := strings.TrimPrefix(w, "--")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				name = name[:eq]
			}
			f.long[name] = true
		case len(w) > 1 && w[0] == '-' && !looksNumeric(w):
			for _, r := range w[1:] {
				f.short[r] = true
			}
		default:
			f.pos = append(f.pos, w)
		}
	}
	return f
}

func (f flags) hasShort(r rune) bool  { return f.short[r] }
func (f flags) hasLong(s string) bool { return f.long[s] }
func (f flags) anyShort(rs ...rune) bool {
	for _, r := range rs {
		if f.short[r] {
			return true
		}
	}
	return false
}

// looksNumeric reports whether a "-1" style token is a negative number rather
// than a flag (so chmod's mode and dd's counts aren't mistaken for flags).
func looksNumeric(w string) bool {
	if len(w) < 2 || w[0] != '-' {
		return false
	}
	for _, r := range w[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// contains reports whether any word equals target.
func contains(words []string, target string) bool {
	for _, w := range words {
		if w == target {
			return true
		}
	}
	return false
}
