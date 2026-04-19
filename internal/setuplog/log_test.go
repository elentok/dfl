package setuplog

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	runtimectx "dfl/internal/runtime"
)

func TestReadPairsStepStartAndEnd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "setup.jsonl")

	if err := AppendStart(path, "create directory X"); err != nil {
		t.Fatalf("AppendStart: %v", err)
	}
	if err := AppendEnd(path, runtimectx.StatusSkipped, "already exists"); err != nil {
		t.Fatalf("AppendEnd: %v", err)
	}

	steps, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("len(steps) = %d, want 1", len(steps))
	}
	if steps[0].Text != "create directory X" || steps[0].Status != runtimectx.StatusSkipped || steps[0].Message != "already exists" {
		t.Fatalf("step = %#v", steps[0])
	}
}

func TestReadIncludesAtomicStepResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "setup.jsonl")

	if err := AppendResult(path, "git-clone this repo", runtimectx.StatusFailed, "failed", "stdout\nstderr\n"); err != nil {
		t.Fatalf("AppendResult: %v", err)
	}

	steps, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("len(steps) = %d, want 1", len(steps))
	}
	if steps[0].Output != "stdout\nstderr\n" {
		t.Fatalf("output = %q, want captured output", steps[0].Output)
	}
}

func TestRenderSummaryIncludesFailedOutput(t *testing.T) {
	var out bytes.Buffer

	err := RenderSummary(&out, []Step{
		{Text: "create directory X", Status: runtimectx.StatusSkipped, Message: "already exists"},
		{Text: "git-clone this repo", Status: runtimectx.StatusFailed, Message: "failed", Output: "line 1\nline 2\n"},
	})
	if err != nil {
		t.Fatalf("RenderSummary: %v", err)
	}

	text := out.String()
	if !strings.Contains(text, "Setup Summary") {
		t.Fatalf("summary = %q, want section header", text)
	}
	if !strings.Contains(text, "Setup Summary \n\n") {
		t.Fatalf("summary = %q, want blank line after header", text)
	}
	if !strings.Contains(text, "create directory X... already exists") {
		t.Fatalf("summary = %q, want skipped step", text)
	}
	if !strings.Contains(text, "git-clone this repo... failed") {
		t.Fatalf("summary = %q, want failed step", text)
	}
	if !strings.Contains(text, "    line 1") || !strings.Contains(text, "    line 2") {
		t.Fatalf("summary = %q, want indented failure output", text)
	}
	if !strings.Contains(text, "\n\n✗ dfl setup failed: 1 of 2 steps failed\n") {
		t.Fatalf("summary = %q, want final failure summary", text)
	}
}

func TestRenderSummaryIncludesSuccessFinalLine(t *testing.T) {
	var out bytes.Buffer

	err := RenderSummary(&out, []Step{
		{Text: "create directory X", Status: runtimectx.StatusSkipped, Message: "already exists"},
		{Text: "create directory Y", Status: runtimectx.StatusSuccess, Message: "done"},
	})
	if err != nil {
		t.Fatalf("RenderSummary: %v", err)
	}

	text := out.String()
	if !strings.Contains(text, "\n\n✓ dfl setup completed successfully\n") {
		t.Fatalf("summary = %q, want final success summary", text)
	}
}
