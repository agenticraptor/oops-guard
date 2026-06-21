// Package confirm asks the user to approve a dangerous command on the
// controlling terminal. It reads directly from /dev/tty (not stdin), so it
// works when invoked from a shell hook whose stdin is the script being run. To
// behave correctly whether the shell left the terminal in cooked or raw mode
// (zsh's line editor uses raw mode), it does its own minimal echo and editing.
package confirm

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// CriticalPhrase is what the user must type in full to proceed past a Critical
// warning. The friction is the point.
const CriticalPhrase = "yes, do it"

// ErrNoTTY is returned when there is no terminal to prompt on.
var ErrNoTTY = errors.New("no controlling terminal available to confirm on")

var errCanceled = errors.New("canceled")

// Ask prompts for approval. When critical is true it requires the user to type
// CriticalPhrase in full; otherwise it is a single-key y/N defaulting to No.
// It returns true only if the user explicitly approves.
func Ask(prompt string, critical bool) (bool, error) {
	f, closeFn, err := openTTY()
	if err != nil {
		return false, ErrNoTTY
	}
	defer closeFn()

	if critical {
		fmt.Fprintf(f, "%s\n  type %q to proceed (anything else cancels): ", prompt, CriticalPhrase)
		line, err := readLineRaw(f)
		if err != nil {
			return false, nil // canceled or unreadable → treat as "no"
		}
		return strings.TrimSpace(line) == CriticalPhrase, nil
	}

	fmt.Fprint(f, prompt+" [y/N] ")
	c, err := readKeyRaw(f)
	if err != nil {
		fmt.Fprintln(f)
		return false, nil
	}
	yes := c == 'y' || c == 'Y'
	if yes {
		fmt.Fprintln(f, "y")
	} else {
		fmt.Fprintln(f, "n")
	}
	return yes, nil
}

func openTTY() (*os.File, func(), error) {
	if f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		return f, func() { _ = f.Close() }, nil
	}
	// Fall back to stdin (e.g. on Windows there is no /dev/tty).
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return os.Stdin, func() {}, nil
	}
	return nil, func() {}, ErrNoTTY
}

func readKeyRaw(f *os.File) (byte, error) {
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return 0, ErrNoTTY
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	defer func() { _ = term.Restore(fd, old) }()
	var b [1]byte
	if _, err := f.Read(b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func readLineRaw(f *os.File) (string, error) {
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return "", ErrNoTTY
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(fd, old) }()

	var sb strings.Builder
	var b [1]byte
	for {
		if _, err := f.Read(b[:]); err != nil {
			return sb.String(), err
		}
		switch c := b[0]; {
		case c == '\r' || c == '\n':
			_, _ = f.Write([]byte("\r\n"))
			return sb.String(), nil
		case c == 3: // Ctrl-C
			_, _ = f.Write([]byte("\r\n"))
			return "", errCanceled
		case c == 127 || c == 8: // backspace / delete
			if s := sb.String(); s != "" {
				sb.Reset()
				_, _ = sb.WriteString(s[:len(s)-1])
				_, _ = f.Write([]byte("\b \b"))
			}
		default:
			_ = sb.WriteByte(c)
			_, _ = f.Write(b[:])
		}
	}
}
