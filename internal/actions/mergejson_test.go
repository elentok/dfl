package actions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dfl/internal/runctx"
)

func TestMergeJSONWritesUnionedResult(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.json", `{"permissions":{"allow":["x"]},"model":"sonnet"}`)
	writeFile(t, dir, "b.json", `{"permissions":{"allow":["y"]},"model":"opus"}`)
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	out := filepath.Join(dir, "out.json")

	status, message, err := Runner{}.MergeJSON(runctx.Context{}, dir, []string{a, b}, out)
	if err != nil {
		t.Fatalf("MergeJSON returned error: %v", err)
	}
	if status != runctx.StatusSuccess || message != "done" {
		t.Fatalf("status/message = %q/%q, want success/done", status, message)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile out: %v", err)
	}
	want := "{\n  \"model\": \"opus\",\n  \"permissions\": {\n    \"allow\": [\n      \"x\",\n      \"y\"\n    ]\n  }\n}\n"
	if string(data) != want {
		t.Fatalf("out = %q, want %q", string(data), want)
	}
}

func TestMergeJSONInPlaceOutputEqualsInput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "settings.json", `{"permissions":{"allow":["runtime"]},"model":"sonnet"}`)
	writeFile(t, dir, "base.json", `{"permissions":{"allow":["managed"]},"model":"opus"}`)
	live := filepath.Join(dir, "settings.json")
	base := filepath.Join(dir, "base.json")

	// live first, base last so base wins on scalar; output overlaps first input.
	status, _, err := Runner{}.MergeJSON(runctx.Context{}, dir, []string{live, base}, live)
	if err != nil {
		t.Fatalf("MergeJSON returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}

	data, err := os.ReadFile(live)
	if err != nil {
		t.Fatalf("ReadFile live: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"runtime"`) || !strings.Contains(got, `"managed"`) {
		t.Fatalf("allow not unioned in place: %q", got)
	}
	if !strings.Contains(got, `"opus"`) {
		t.Fatalf("base scalar did not win: %q", got)
	}
}

func TestMergeJSONSkipsWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.json", `{"a":1}`)
	writeFile(t, dir, "b.json", `{"b":2}`)
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	out := filepath.Join(dir, "settings.json")

	if _, _, err := (Runner{}).MergeJSON(runctx.Context{}, dir, []string{a, b}, out); err != nil {
		t.Fatalf("first MergeJSON error: %v", err)
	}

	status, message, err := Runner{}.MergeJSON(runctx.Context{}, dir, []string{a, b}, out)
	if err != nil {
		t.Fatalf("second MergeJSON error: %v", err)
	}
	if status != runctx.StatusSkipped || message != "already up to date" {
		t.Fatalf("status/message = %q/%q, want skipped/already up to date", status, message)
	}
}

func TestMergeJSONDryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.json", `{"a":1}`)
	writeFile(t, dir, "b.json", `{"b":2}`)
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	out := filepath.Join(dir, "out.json")

	status, message, err := Runner{}.MergeJSON(runctx.Context{DryRun: true}, dir, []string{a, b}, out)
	if err != nil {
		t.Fatalf("MergeJSON returned error: %v", err)
	}
	if status != runctx.StatusSuccess {
		t.Fatalf("status = %q, want success", status)
	}
	if !strings.Contains(message, "would merge 2 files into") {
		t.Fatalf("message = %q, want dry-run output", message)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("output file should not exist after dry-run")
	}
}

func TestMergeJSONErrorsOnMissingInput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.json", `{"a":1}`)
	a := filepath.Join(dir, "a.json")

	status, _, err := Runner{}.MergeJSON(runctx.Context{}, dir, []string{a, filepath.Join(dir, "missing.json")}, filepath.Join(dir, "out.json"))
	if err == nil {
		t.Fatal("expected error for missing input")
	}
	if status != runctx.StatusFailed {
		t.Fatalf("status = %q, want failed", status)
	}
}

func TestMergeJSONErrorsOnInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.json", `{"a":1}`)
	writeFile(t, dir, "b.json", `{not valid json`)
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")

	status, _, err := Runner{}.MergeJSON(runctx.Context{}, dir, []string{a, b}, filepath.Join(dir, "out.json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if status != runctx.StatusFailed {
		t.Fatalf("status = %q, want failed", status)
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Fatalf("error = %v, want parsing context", err)
	}
}
