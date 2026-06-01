package actions

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"dfl/internal/runctx"
	"dfl/internal/ui"
)

func (o Runner) HasCommand(name string) (bool, error) {
	_, err := exec.LookPath(name)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, exec.ErrNotFound) {
		return false, nil
	}

	var execErr *exec.Error
	if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
		return false, nil
	}

	return false, err
}

func (o Runner) Ask(question, defaultValue string) (string, error) {
	promptWriter := o.stderr()
	if _, err := fmt.Fprintf(promptWriter, "%s ", question); err != nil {
		return "", err
	}
	if defaultValue != "" {
		if _, err := fmt.Fprintf(promptWriter, "[%s] ", defaultValue); err != nil {
			return "", err
		}
	}

	reader := bufio.NewReader(o.stdin())
	reply, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if _, err := fmt.Fprintln(promptWriter); err != nil {
		return "", err
	}

	reply = strings.TrimSpace(reply)
	if reply == "" {
		reply = defaultValue
	}
	return reply, nil
}

func (o Runner) StepStart(message string) error {
	return ui.StepStart(o.stdout(), message)
}

func (o Runner) StepEnd(status runctx.ResultStatus, message string) error {
	return ui.StepEnd(o.stdout(), status, message)
}
