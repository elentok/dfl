package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeJSONCommandMergesFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(`{"permissions":{"allow":["x"]}}`), 0o644); err != nil {
		t.Fatalf("WriteFile a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(`{"permissions":{"allow":["y"]}}`), 0o644); err != nil {
		t.Fatalf("WriteFile b: %v", err)
	}
	out := filepath.Join(dir, "out.json")

	app := NewApp()
	var stdout, stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"merge-json", filepath.Join(dir, "a.json"), filepath.Join(dir, "b.json"), out})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Merging 2 files into "+out) {
		t.Fatalf("stdout = %q, want merge step label", stdout.String())
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile out: %v", err)
	}
	if !strings.Contains(string(data), `"x"`) || !strings.Contains(string(data), `"y"`) {
		t.Fatalf("merged output missing unioned entries: %q", string(data))
	}
}

func TestMergeJSONCommandRejectsTooFewArgs(t *testing.T) {
	dir := t.TempDir()
	app := NewApp()
	var stdout, stderr bytes.Buffer
	app.SetStdout(&stdout)
	app.SetStderr(&stderr)

	code, err := app.Run([]string{"merge-json", filepath.Join(dir, "a.json"), filepath.Join(dir, "out.json")})
	if err == nil {
		t.Fatal("expected error for fewer than 3 args")
	}
	if code == 0 {
		t.Fatal("expected non-zero exit code for arg validation failure")
	}
}
