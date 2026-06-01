package actions

import (
	"bytes"
	"strings"
	"testing"
)

func TestHasCommandFindsExecutable(t *testing.T) {
	ops := Runner{}
	found, err := ops.HasCommand("sh")
	if err != nil {
		t.Fatalf("HasCommand returned error: %v", err)
	}
	if !found {
		t.Fatal("HasCommand returned false, want true for sh")
	}
}

func TestAskUsesDefaultForEmptyReply(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := Runner{
		Stdin:  strings.NewReader("\n"),
		Stdout: &stdout,
		Stderr: &stderr,
	}

	reply, err := runner.Ask("What's up?", "ok")
	if err != nil {
		t.Fatalf("Ask returned error: %v", err)
	}
	if reply != "ok" {
		t.Fatalf("reply = %q, want default reply", reply)
	}
	if !strings.Contains(stderr.String(), "What's up? [ok] ") {
		t.Fatalf("stderr = %q, want prompt", stderr.String())
	}
}
