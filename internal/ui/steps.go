package ui

import (
	"fmt"
	"io"

	runtimectx "dfl/internal/runtime"
)

const (
	colorBlue  = "\x1b[34m"
	colorGreen = "\x1b[32m"
	colorGray  = "\x1b[90m"
	colorRed   = "\x1b[31m"
	colorReset = "\x1b[0m"
)

func StepStart(w io.Writer, message string) error {
	_, err := fmt.Fprintf(w, "\n%s◆ %s%s\n", colorBlue, message, colorReset)
	return err
}

func StepEnd(w io.Writer, status runtimectx.ResultStatus, message string) error {
	color, icon := statusStyle(status)
	if message == "" {
		_, err := fmt.Fprintf(w, "%s%s %s%s\n", color, icon, status, colorReset)
		return err
	}

	_, err := fmt.Fprintf(w, "%s%s %s: %s%s\n", color, icon, status, message, colorReset)
	return err
}

func statusStyle(status runtimectx.ResultStatus) (string, string) {
	switch status {
	case runtimectx.StatusSuccess:
		return colorGreen, "✓"
	case runtimectx.StatusSkipped:
		return colorGray, "•"
	case runtimectx.StatusFailed:
		return colorRed, "✗"
	default:
		return colorGray, "•"
	}
}

func Step(w io.Writer, message string, fn func() (runtimectx.ResultStatus, string, error)) error {
	if err := StepStart(w, message); err != nil {
		return err
	}

	status, detail, err := fn()
	if err != nil {
		return err
	}

	return StepEnd(w, status, detail)
}
