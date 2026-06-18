package analyzer

import "strings"

// Command is one simple command extracted from a command line: its argv plus
// the raw text it came from. Redirection operators (>, >>, <) are preserved as
// their own tokens so rules can reason about them.
type Command struct {
	Raw  string   // the original text of this simple command
	Args []string // tokenized arguments, with quotes removed
}

// Name returns the command's program name (argv[0]), or "" if empty. A leading
// environment assignment (FOO=bar cmd) and common prefixes (sudo, command, env,
// time, nice, nohup, xargs) are skipped so the real verb is returned.
func (c Command) Name() string {
	args := c.skipPrefixes()
	if len(args) == 0 {
		return ""
	}
	return base(args[0])
}

// Words returns the argv with leading env-assignments and wrapper prefixes
// removed, so rule matchers see the real command and its flags.
func (c Command) Words() []string {
	return c.skipPrefixes()
}

// wrapperCommands run another command given as their argument, so the real verb
// a danger rule cares about is what follows them (and their options).
var wrapperCommands = map[string]bool{
	"sudo": true, "doas": true, "command": true, "env": true, "time": true,
	"nice": true, "ionice": true, "nohup": true, "stdbuf": true, "setsid": true,
	"exec": true, "builtin": true, "timeout": true, "xargs": true,
}

// wrapperValueFlags are wrapper options that consume the following token as a
// value, so it isn't mistaken for the wrapped command.
var wrapperValueFlags = map[string]bool{
	"-u": true, "--user": true, // sudo
	"-n": true, "--adjustment": true, // nice
	"-c": true, "--class": true, "--classdata": true, // ionice
	"-s": true, "--signal": true, "-k": true, "--kill-after": true, // timeout
}

func (c Command) skipPrefixes() []string {
	args := c.Args
	for len(args) > 0 {
		w := args[0]
		if isAssignment(w) {
			args = args[1:]
			continue
		}
		if !wrapperCommands[w] {
			return args
		}
		args = args[1:]
		// Consume the wrapper's own options (and any values they take).
		for len(args) > 0 && len(args[0]) > 1 && args[0][0] == '-' && args[0] != "--" {
			flag := args[0]
			args = args[1:]
			if strings.Contains(flag, "=") {
				continue // --flag=value carries its own value
			}
			if wrapperValueFlags[flag] && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
				args = args[1:]
			}
		}
		if len(args) > 0 && args[0] == "--" {
			args = args[1:]
		}
		// timeout takes a leading DURATION positional before the command.
		if w == "timeout" && len(args) > 0 && looksLikeDuration(args[0]) {
			args = args[1:]
		}
	}
	return args
}

// looksLikeDuration reports whether s is a number with an optional time suffix,
// e.g. "5", "0.5", "30s", "2m" — the leading argument timeout(1) expects.
func looksLikeDuration(s string) bool {
	s = strings.TrimRight(s, "smhd")
	if s == "" {
		return false
	}
	dot := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r == '.' && !dot:
			dot = true
		default:
			return false
		}
	}
	return true
}

// Pipeline is the parsed form of a whole command line: the sequence of simple
// commands plus whether any two were joined by a pipe (used by curl|sh rules).
type Pipeline struct {
	Raw      string // the original, unmodified command line
	Commands []Command
	Piped    bool // true if at least one '|' joined two commands
}

// Parse splits a raw command line into its simple commands, honoring single
// quotes, double quotes and backslash escapes, and breaking on the shell
// operators ; & && || | and newlines. It is intentionally tolerant: anything it
// cannot make sense of is still surfaced as best-effort tokens.
func Parse(line string) Pipeline {
	var (
		pl       Pipeline
		args     []string
		cur      strings.Builder
		rawStart int
		hasTok   bool
	)
	pl.Raw = line
	runes := []rune(line)

	flushTok := func() {
		if hasTok {
			args = append(args, cur.String())
			cur.Reset()
			hasTok = false
		}
	}
	flushCmd := func(end int, piped bool) {
		flushTok()
		raw := strings.TrimSpace(string(runes[rawStart:end]))
		if len(args) > 0 {
			pl.Commands = append(pl.Commands, Command{Raw: raw, Args: args})
		}
		args = nil
		rawStart = end + 1
		if piped {
			pl.Piped = true
		}
	}

	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		switch ch {
		case '\'':
			hasTok = true
			for i++; i < len(runes) && runes[i] != '\''; i++ {
				cur.WriteRune(runes[i])
			}
		case '"':
			hasTok = true
			for i++; i < len(runes) && runes[i] != '"'; i++ {
				if runes[i] == '\\' && i+1 < len(runes) {
					i++
				}
				cur.WriteRune(runes[i])
			}
		case '\\':
			hasTok = true
			if i+1 < len(runes) {
				i++
				cur.WriteRune(runes[i])
			}
		case ' ', '\t':
			flushTok()
		case '|':
			if i+1 < len(runes) && runes[i+1] == '|' {
				flushCmd(i, false)
				i++
			} else {
				flushCmd(i, true)
			}
			rawStart = i + 1
		case '&':
			if i+1 < len(runes) && runes[i+1] == '&' {
				i++
			}
			flushCmd(i, false)
			rawStart = i + 1
		case ';', '\n':
			flushCmd(i, false)
			rawStart = i + 1
		case '>', '<':
			// Preserve redirections as standalone tokens.
			flushTok()
			op := string(ch)
			if ch == '>' && i+1 < len(runes) && runes[i+1] == '>' {
				op = ">>"
				i++
			}
			args = append(args, op)
		default:
			hasTok = true
			cur.WriteRune(ch)
		}
	}
	flushCmd(len(runes), false)
	return pl
}

func isAssignment(w string) bool {
	eq := strings.IndexByte(w, '=')
	if eq <= 0 {
		return false
	}
	for i, r := range w[:eq] {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

// base returns the last path element of a program name (so /bin/rm -> rm).
func base(s string) string {
	if i := strings.LastIndexAny(s, "/\\"); i >= 0 {
		return s[i+1:]
	}
	return s
}
