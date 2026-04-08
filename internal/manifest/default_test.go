package manifest

import (
	"path/filepath"
	"testing"
)

func TestDefaultSetupManifestParses(t *testing.T) {
	path := filepath.Join("..", "..", "setup", "default.yaml")
	if _, err := ParseSetupFile(path); err != nil {
		t.Fatalf("ParseSetupFile(%s): %v", path, err)
	}
}
