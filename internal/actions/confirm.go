package actions

import (
	"fmt"
	"os"

	"github.com/charmbracelet/x/term"
)

// confirmDecision is the outcome of interpreting a single input byte.
type confirmDecision int

const (
	// confirmIgnore means the key has no meaning here; keep waiting.
	confirmIgnore confirmDecision = iota
	confirmYes
	confirmNo
	// confirmDefault resolves to whatever the configured default is (Enter).
	confirmDefault
	// confirmCancel is Esc / Ctrl-C / Ctrl-D, all of which resolve to "no".
	confirmCancel
)

// confirmKey maps a single raw input byte to a decision. Pure and table-tested.
// Any Esc (0x1b) is treated as cancel, so escape sequences (arrow keys, etc.)
// cancel rather than being ignored — acceptable for a yes/no prompt.
func confirmKey(b byte) confirmDecision {
	switch b {
	case 'y', 'Y':
		return confirmYes
	case 'n', 'N':
		return confirmNo
	case '\r', '\n':
		return confirmDefault
	case 0x1b, 0x03, 0x04: // Esc, Ctrl-C, Ctrl-D
		return confirmCancel
	default:
		return confirmIgnore
	}
}

// Confirm prints "question (y/N)? " (or "(Y/n)?" when defaultYes) to stderr and
// reads a single keypress in raw mode. It returns true for yes, false for no.
// When stdin is not a terminal it silently resolves to the default.
func (o Runner) Confirm(question string, defaultYes bool) (bool, error) {
	suffix := "(y/N)"
	if defaultYes {
		suffix = "(Y/n)"
	}
	if _, err := fmt.Fprintf(o.stderr(), "%s %s? ", question, suffix); err != nil {
		return false, err
	}

	file, ok := o.stdin().(*os.File)
	if !ok || !term.IsTerminal(file.Fd()) {
		// Non-interactive: resolve to the default without blocking. Plain "\n"
		// so redirected output / files don't get stray carriage returns.
		return o.finishConfirm(defaultYes, "\n")
	}

	state, err := term.MakeRaw(file.Fd())
	if err != nil {
		return false, err
	}
	defer term.Restore(file.Fd(), state)

	// Raw mode disables output post-processing (ONLCR), so a bare "\n" only
	// moves down a row without returning to column 0. Emit "\r\n" ourselves.
	buf := make([]byte, 1)
	for {
		n, err := file.Read(buf)
		if err != nil {
			// Read error / EOF: treat as cancel (no).
			return o.finishConfirm(false, "\r\n")
		}
		if n == 0 {
			continue
		}
		switch confirmKey(buf[0]) {
		case confirmYes:
			return o.finishConfirm(true, "\r\n")
		case confirmNo, confirmCancel:
			return o.finishConfirm(false, "\r\n")
		case confirmDefault:
			return o.finishConfirm(defaultYes, "\r\n")
		case confirmIgnore:
			continue
		}
	}
}

// finishConfirm echoes the resolved answer (y/n) and eol to stderr, since raw
// mode does not echo keypresses, then returns the result.
func (o Runner) finishConfirm(result bool, eol string) (bool, error) {
	letter := "n"
	if result {
		letter = "y"
	}
	if _, err := fmt.Fprintf(o.stderr(), "%s%s", letter, eol); err != nil {
		return result, err
	}
	return result, nil
}
