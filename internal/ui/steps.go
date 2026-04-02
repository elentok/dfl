package ui

import (
	"fmt"
	"io"

	runtimectx "dfl/internal/runtime"
)

func StepStart(w io.Writer, message string) error {
	_, err := fmt.Fprintf(w, "==> %s\n", message)
	return err
}

func StepEnd(w io.Writer, status runtimectx.ResultStatus, message string) error {
	if message == "" {
		_, err := fmt.Fprintf(w, "[%s]\n", status)
		return err
	}

	_, err := fmt.Fprintf(w, "[%s] %s\n", status, message)
	return err
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
