package actions

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmKeyDecisionTable(t *testing.T) {
	cases := []struct {
		name string
		b    byte
		want confirmDecision
	}{
		{"lower yes", 'y', confirmYes},
		{"upper yes", 'Y', confirmYes},
		{"lower no", 'n', confirmNo},
		{"upper no", 'N', confirmNo},
		{"enter cr", '\r', confirmDefault},
		{"enter lf", '\n', confirmDefault},
		{"escape", 0x1b, confirmCancel},
		{"ctrl-c", 0x03, confirmCancel},
		{"ctrl-d", 0x04, confirmCancel},
		{"space ignored", ' ', confirmIgnore},
		{"other letter ignored", 'q', confirmIgnore},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := confirmKey(tc.b); got != tc.want {
				t.Fatalf("confirmKey(%q) = %d, want %d", tc.b, got, tc.want)
			}
		})
	}
}

func TestConfirmNonTTYUsesDefaultNo(t *testing.T) {
	var stderr bytes.Buffer
	runner := Runner{Stdin: strings.NewReader(""), Stderr: &stderr}

	ok, err := runner.Confirm("Proceed?", false)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if ok {
		t.Fatal("Confirm = true, want false (default no)")
	}
	if !strings.Contains(stderr.String(), "Proceed? (y/N)? ") {
		t.Fatalf("stderr = %q, want prompt with (y/N)", stderr.String())
	}
	if !strings.HasSuffix(stderr.String(), "n\n") {
		t.Fatalf("stderr = %q, want echoed n", stderr.String())
	}
}

func TestConfirmNonTTYUsesDefaultYes(t *testing.T) {
	var stderr bytes.Buffer
	runner := Runner{Stdin: strings.NewReader(""), Stderr: &stderr}

	ok, err := runner.Confirm("Proceed?", true)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if !ok {
		t.Fatal("Confirm = false, want true (default yes)")
	}
	if !strings.Contains(stderr.String(), "Proceed? (Y/n)? ") {
		t.Fatalf("stderr = %q, want prompt with (Y/n)", stderr.String())
	}
	if !strings.HasSuffix(stderr.String(), "y\n") {
		t.Fatalf("stderr = %q, want echoed y", stderr.String())
	}
}
