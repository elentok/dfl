package selfmgr

import (
	"bytes"
	"strings"
	"testing"
)

func TestUpdateDryRunDoesNotRequireInstalledBinary(t *testing.T) {
	repoRoot := t.TempDir()

	var stdout bytes.Buffer
	updater := Updater{
		Stdout: &stdout,
		DryRun: true,
	}

	code, err := updater.Run(repoRoot)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if code != 0 {
		t.Fatalf("Run returned code %d, want 0", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "would install dfl latest release to") && !strings.Contains(output, "latest version already installed") {
		t.Fatalf("stdout = %q, want dry-run install output or latest-version skip output", output)
	}
	if !strings.Contains(output, "would update "+repoRoot) {
		t.Fatalf("stdout = %q, want dry-run repo update output", output)
	}
	if !strings.Contains(output, "would run dfl setup --repo "+repoRoot+" --dry-run") {
		t.Fatalf("stdout = %q, want dry-run setup output", output)
	}
}
