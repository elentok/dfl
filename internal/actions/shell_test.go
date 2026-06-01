package actions

import (
	"bytes"
	"strings"
	"testing"

	"dfl/internal/runctx"
)

func TestShellDryRunSkipsExecution(t *testing.T) {
	var stdout bytes.Buffer
	ops := Runner{Stdout: &stdout}

	code, err := ops.Shell(runctx.Context{DryRun: true}, "demo", []string{"echo", "hi"})
	if err != nil {
		t.Fatalf("Shell returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Shell returned code %d, want 0", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "◆ demo") {
		t.Fatalf("stdout = %q, want step start", output)
	}
	if !strings.Contains(output, "DRY-RUN: echo hi") {
		t.Fatalf("stdout = %q, want dry-run command", output)
	}
	if !strings.Contains(output, "○ dry-run") {
		t.Fatalf("stdout = %q, want skipped step end", output)
	}
}

func TestShellPassesParentEnvironment(t *testing.T) {
	t.Setenv("DFL_TEST_PARENT_ENV", "from-parent")

	var stdout bytes.Buffer
	ops := Runner{Stdout: &stdout}
	ctx := runctx.Context{}

	code, err := ops.Shell(ctx, "env", []string{
		"sh",
		"-c",
		`printf '%s\n' "$DFL_TEST_PARENT_ENV"`,
	})
	if err != nil {
		t.Fatalf("Shell returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Shell returned code %d, want 0", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "from-parent\n") {
		t.Fatalf("stdout = %q, want parent environment in child", output)
	}
}
