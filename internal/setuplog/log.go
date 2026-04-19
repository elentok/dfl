package setuplog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	runtimectx "dfl/internal/runtime"
	"dfl/internal/ui"
)

const (
	summaryIndent             = "  "
	recordTypeComponentHeader = "component_header"
	recordTypeStepStart       = "step_start"
	recordTypeStepEnd         = "step_end"
	recordTypeStepResult      = "step_result"
	maxSummaryOutputLines     = 80
)

type Record struct {
	Type    string                  `json:"type"`
	Text    string                  `json:"text,omitempty"`
	Status  runtimectx.ResultStatus `json:"status,omitempty"`
	Message string                  `json:"message,omitempty"`
	Output  string                  `json:"output,omitempty"`
}

type Step struct {
	IsHeader bool
	Text     string
	Status   runtimectx.ResultStatus
	Message  string
	Output   string
}

func AppendComponentHeader(path, text string) error {
	if path == "" || text == "" {
		return nil
	}
	return appendRecord(path, Record{
		Type: recordTypeComponentHeader,
		Text: text,
	})
}

func AppendStart(path, text string) error {
	if path == "" || text == "" {
		return nil
	}
	return appendRecord(path, Record{
		Type: recordTypeStepStart,
		Text: text,
	})
}

func AppendEnd(path string, status runtimectx.ResultStatus, message string) error {
	if path == "" {
		return nil
	}
	return appendRecord(path, Record{
		Type:    recordTypeStepEnd,
		Status:  status,
		Message: message,
	})
}

func AppendResult(path, text string, status runtimectx.ResultStatus, message, output string) error {
	if path == "" || text == "" {
		return nil
	}
	return appendRecord(path, Record{
		Type:    recordTypeStepResult,
		Text:    text,
		Status:  status,
		Message: message,
		Output:  output,
	})
}

func appendRecord(path string, record Record) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = file.Write(data)
	return err
}

func Read(path string) ([]Step, error) {
	if path == "" {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var (
		steps []Step
		stack []string
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}

		switch record.Type {
		case recordTypeComponentHeader:
			if record.Text == "" {
				continue
			}
			steps = append(steps, Step{
				IsHeader: true,
				Text:     record.Text,
			})
		case recordTypeStepStart:
			if record.Text != "" {
				stack = append(stack, record.Text)
			}
		case recordTypeStepEnd:
			if len(stack) == 0 {
				continue
			}
			text := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			steps = append(steps, Step{
				Text:    text,
				Status:  record.Status,
				Message: record.Message,
				Output:  record.Output,
			})
		case recordTypeStepResult:
			if record.Text == "" {
				continue
			}
			steps = append(steps, Step{
				Text:    record.Text,
				Status:  record.Status,
				Message: record.Message,
				Output:  record.Output,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return steps, nil
}

func RenderSummary(w io.Writer, steps []Step) error {
	if len(steps) == 0 {
		return nil
	}

	if err := ui.SectionHeader(w, "Setup Summary"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	failedCount := 0
	stepCount := 0
	for _, step := range steps {
		if step.IsHeader {
			if err := ui.StepStartWithIndent(w, step.Text, "", false); err != nil {
				return err
			}
			continue
		}

		stepCount++
		message := step.Message
		if message == "" {
			message = string(step.Status)
		}
		if err := ui.StepEndWithIndent(w, step.Status, fmt.Sprintf("%s... %s", step.Text, message), summaryIndent); err != nil {
			return err
		}
		if step.Status == runtimectx.StatusFailed && strings.TrimSpace(step.Output) != "" {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
			for _, line := range summarizeOutput(step.Output) {
				if _, err := fmt.Fprintf(w, "    %s\n", line); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if step.Status == runtimectx.StatusFailed {
			failedCount++
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if failedCount == 0 {
		return ui.StepEndWithIndent(w, runtimectx.StatusSuccess, "dfl setup completed successfully", "")
	}

	return ui.StepEndWithIndent(w, runtimectx.StatusFailed, fmt.Sprintf("dfl setup failed: %d of %d steps failed", failedCount, stepCount), "")
}

func summarizeOutput(output string) []string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) <= maxSummaryOutputLines {
		return lines
	}

	tail := append([]string(nil), lines[len(lines)-maxSummaryOutputLines:]...)
	omitted := len(lines) - maxSummaryOutputLines
	return append([]string{fmt.Sprintf("... %d earlier lines omitted ...", omitted)}, tail...)
}
