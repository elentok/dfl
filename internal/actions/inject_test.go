package actions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dfl/internal/runctx"
)

func TestInjectAppendsManagedBlock(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source.md")
	target := filepath.Join(tempDir, "target.md")
	if err := os.WriteFile(source, []byte("this is the dotfiles' AGENTS.md\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	if err := os.WriteFile(target, []byte("this is the user-scope Codex AGENTS.md\n"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	status, message, err := Runner{}.Inject(runctx.Context{}, tempDir, source, target, false)
	if err != nil {
		t.Fatalf("Inject returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if message != "done" {
		t.Fatalf("message = %q, want done", message)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}

	want := "this is the user-scope Codex AGENTS.md\n\n<!-- dfl:inject:start source=" + source + " -->\nthis is the dotfiles' AGENTS.md\n<!-- dfl:inject:end -->\n"
	if string(data) != want {
		t.Fatalf("target = %q, want %q", string(data), want)
	}
}

func TestInjectReplacesExistingManagedBlock(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source.md")
	target := filepath.Join(tempDir, "target.md")
	if err := os.WriteFile(source, []byte("second version\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	initial := "base text\n\n<!-- dfl:inject:start source=/old/source.md -->\nfirst version\n<!-- dfl:inject:end -->\n"
	if err := os.WriteFile(target, []byte(initial), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	status, message, err := Runner{}.Inject(runctx.Context{}, tempDir, source, target, false)
	if err != nil {
		t.Fatalf("Inject returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if message != "done" {
		t.Fatalf("message = %q, want done", message)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}
	output := string(data)
	if strings.Count(output, injectEndMarker) != 1 {
		t.Fatalf("target = %q, want exactly one managed block", output)
	}
	if !strings.Contains(output, "second version\n") {
		t.Fatalf("target = %q, want updated source contents", output)
	}
	if strings.Contains(output, "first version\n") {
		t.Fatalf("target = %q, want stale source contents removed", output)
	}
}

func TestInjectDryRunDoesNotModifyTarget(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source.md")
	target := filepath.Join(tempDir, "target.md")
	if err := os.WriteFile(source, []byte("injected text\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	if err := os.WriteFile(target, []byte("base text\n"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	status, message, err := Runner{}.Inject(runctx.Context{DryRun: true}, tempDir, source, target, false)
	if err != nil {
		t.Fatalf("Inject returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if !strings.Contains(message, "would inject") {
		t.Fatalf("message = %q, want dry-run output", message)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}
	if string(data) != "base text\n" {
		t.Fatalf("target = %q, want unchanged", string(data))
	}
}

func TestInjectLinkModeWritesReferencePayload(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source with spaces.md")
	target := filepath.Join(tempDir, "target.md")
	if err := os.WriteFile(source, []byte("ignored in link mode\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	if err := os.WriteFile(target, []byte("base text\n"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	status, message, err := Runner{}.Inject(runctx.Context{}, tempDir, source, target, true)
	if err != nil {
		t.Fatalf("Inject returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if message != "done" {
		t.Fatalf("message = %q, want done", message)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}

	want := "base text\n\n<!-- dfl:inject:start source=" + source + " -->\n@" + source + "\n<!-- dfl:inject:end -->\n"
	if string(data) != want {
		t.Fatalf("target = %q, want %q", string(data), want)
	}
}

func TestInjectLinkModeCollapsesHomeInPayload(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sourceDir := filepath.Join(home, ".dotfiles", "core", "ai")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	source := filepath.Join(sourceDir, "AGENTS.md")
	target := filepath.Join(home, "target.md")
	if err := os.WriteFile(source, []byte("ignored in link mode\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	if err := os.WriteFile(target, []byte("base text\n"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	status, message, err := Runner{}.Inject(runctx.Context{}, home, source, target, true)
	if err != nil {
		t.Fatalf("Inject returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if message != "done" {
		t.Fatalf("message = %q, want done", message)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}

	want := "base text\n\n<!-- dfl:inject:start source=~/.dotfiles/core/ai/AGENTS.md -->\n@~/.dotfiles/core/ai/AGENTS.md\n<!-- dfl:inject:end -->\n"
	if string(data) != want {
		t.Fatalf("target = %q, want %q", string(data), want)
	}
}

func TestInjectDryRunLinkModeMessage(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source.md")
	target := filepath.Join(tempDir, "target.md")
	if err := os.WriteFile(source, []byte("injected text\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	if err := os.WriteFile(target, []byte("base text\n"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	status, message, err := Runner{}.Inject(runctx.Context{DryRun: true}, tempDir, source, target, true)
	if err != nil {
		t.Fatalf("Inject returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if !strings.Contains(message, "would inject link") {
		t.Fatalf("message = %q, want dry-run link output", message)
	}
}

func TestInjectSwitchesBetweenContentAndLinkModes(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source.md")
	target := filepath.Join(tempDir, "target.md")
	if err := os.WriteFile(source, []byte("source content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}
	if err := os.WriteFile(target, []byte("base text\n"), 0o644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}

	if _, _, err := (Runner{}).Inject(runctx.Context{}, tempDir, source, target, false); err != nil {
		t.Fatalf("Inject (content) returned error: %v", err)
	}
	if _, _, err := (Runner{}).Inject(runctx.Context{}, tempDir, source, target, true); err != nil {
		t.Fatalf("Inject (link) returned error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}
	output := string(data)
	if strings.Contains(output, "source content\n") {
		t.Fatalf("target = %q, want content payload removed after switch to link", output)
	}
	if !strings.Contains(output, "@"+source+"\n") {
		t.Fatalf("target = %q, want link payload after switch", output)
	}
}
