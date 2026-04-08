package ui

import (
	"fmt"
	"io"

	runtimectx "dfl/internal/runtime"

	"github.com/charmbracelet/lipgloss"
)

var (
	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("12")).
				Padding(0, 1)
	stepHeaderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	stepSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	stepSkippedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	stepFailedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func SectionHeader(w io.Writer, message string) error {
	_, err := fmt.Fprintf(w, "\n%s\n", sectionHeaderStyle.Render("◈ "+message))
	return err
}

func StepStart(w io.Writer, message string) error {
	_, err := fmt.Fprintf(w, "\n%s\n", stepHeaderStyle.Render("◆ "+message))
	return err
}

func StepEnd(w io.Writer, status runtimectx.ResultStatus, message string) error {
	style, icon := statusStyle(status)
	if message == "" {
		_, err := fmt.Fprintf(w, "%s\n", style.Render(fmt.Sprintf("%s %s", icon, status)))
		return err
	}

	_, err := fmt.Fprintf(w, "%s\n", style.Render(fmt.Sprintf("%s %s", icon, message)))
	return err
}

func statusStyle(status runtimectx.ResultStatus) (lipgloss.Style, string) {
	switch status {
	case runtimectx.StatusSuccess:
		return stepSuccessStyle, "✓"
	case runtimectx.StatusSkipped:
		return stepSkippedStyle, "•"
	case runtimectx.StatusFailed:
		return stepFailedStyle, "✗"
	default:
		return stepSkippedStyle, "•"
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
