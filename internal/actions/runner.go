package actions

import (
	"errors"
	"io"
	"os"
	"strings"
)

type Runner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type OutputError struct {
	Err    error
	Output string
}

func (e *OutputError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *OutputError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func OutputFromError(err error) string {
	var outputErr *OutputError
	if errors.As(err, &outputErr) {
		return strings.TrimSpace(outputErr.Output)
	}
	return ""
}

func combinedOutput(stdout, stderr string) string {
	switch {
	case strings.TrimSpace(stdout) == "":
		return strings.TrimSpace(stderr)
	case strings.TrimSpace(stderr) == "":
		return strings.TrimSpace(stdout)
	default:
		return strings.TrimSpace(stdout) + "\n" + strings.TrimSpace(stderr)
	}
}

func (o Runner) stdout() io.Writer {
	if o.Stdout == nil {
		return io.Discard
	}
	return o.Stdout
}

func (o Runner) stderr() io.Writer {
	if o.Stderr == nil {
		return io.Discard
	}
	return o.Stderr
}

func (o Runner) stdin() io.Reader {
	if o.Stdin == nil {
		return os.Stdin
	}
	return o.Stdin
}
