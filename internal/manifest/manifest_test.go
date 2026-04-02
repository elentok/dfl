package manifest

import (
	"strings"
	"testing"
)

func TestParseInstallBytesParsesSupportedFields(t *testing.T) {
	data := []byte(`
name = "tmux"
kind = "core"
mkdirs = ["~/.config"]

[when]
os = ["mac", "linux"]

[symlinks]
"tmux.conf" = "~/.tmux.conf"

[copies]
"a.txt" = "~/.a.txt"

[[packages]]
manager = "brew"
names = ["tmux"]
tap = "elentok/stuff"

[[steps]]
name = "restore"
os = ["mac"]
if_not = "test -e foo"
cwd = "."
run = "./restore"
`)

	m, err := ParseInstallBytes(data)
	if err != nil {
		t.Fatalf("ParseInstallBytes returned error: %v", err)
	}
	if m.Name != "tmux" || m.Kind != "core" {
		t.Fatalf("unexpected manifest identity: %#v", m)
	}
	if m.Symlinks["tmux.conf"] != "~/.tmux.conf" {
		t.Fatalf("symlink not parsed: %#v", m.Symlinks)
	}
	if len(m.Packages) != 1 || m.Packages[0].Manager != "brew" {
		t.Fatalf("packages not parsed: %#v", m.Packages)
	}
	if len(m.Steps) != 1 || m.Steps[0].Name != "restore" {
		t.Fatalf("steps not parsed: %#v", m.Steps)
	}
}

func TestParseSetupBytesParsesSetupSpecificSections(t *testing.T) {
	data := []byte(`
[repo_defaults]
transport = "inherit"

[[components]]
name = "fish"

[[components]]
name = "osx-tuning"
when_os = ["mac"]

[[packages]]
manager = "brew"
names = ["dff"]
tap = "elentok/stuff"

[[repos]]
name = "notes"
github = "elentok/notes"
path = "~/notes"
transport = "https"

[[steps]]
name = "cache"
run = "deno cache ./**/*.ts"
`)

	m, err := ParseSetupBytes(data)
	if err != nil {
		t.Fatalf("ParseSetupBytes returned error: %v", err)
	}
	if m.RepoDefaults.Transport != "inherit" {
		t.Fatalf("repo defaults not parsed: %#v", m.RepoDefaults)
	}
	if len(m.Components) != 2 || m.Components[1].Name != "osx-tuning" {
		t.Fatalf("components not parsed: %#v", m.Components)
	}
	if len(m.Repos) != 1 || m.Repos[0].GitHub != "elentok/notes" {
		t.Fatalf("repos not parsed: %#v", m.Repos)
	}
}

func TestParseInstallBytesRejectsUnknownFields(t *testing.T) {
	_, err := ParseInstallBytes([]byte(`
name = "tmux"
unknown = "x"
`))
	if err == nil || !strings.Contains(err.Error(), "unknown manifest fields") {
		t.Fatalf("err = %v, want unknown field error", err)
	}
}

func TestValidateInstallRejectsUnsupportedPackageManager(t *testing.T) {
	_, err := ParseInstallBytes([]byte(`
[[packages]]
manager = "mason"
names = ["stylua"]
`))
	if err == nil || !strings.Contains(err.Error(), `unsupported package manager "mason"`) {
		t.Fatalf("err = %v, want unsupported manager error", err)
	}
}

func TestValidateSetupRejectsInvalidRepoShape(t *testing.T) {
	_, err := ParseSetupBytes([]byte(`
[[repos]]
name = "notes"
path = "~/notes"
github = "elentok/notes"
url = "https://github.com/elentok/notes.git"
`))
	if err == nil || !strings.Contains(err.Error(), "must define exactly one of github or url") {
		t.Fatalf("err = %v, want invalid repo error", err)
	}
}

func TestConditionMatchingUsesOSDistroAndFeatures(t *testing.T) {
	ctx := MachineContext{
		OS:          "linux",
		LinuxDistro: "ubuntu",
		FeatureFlags: map[string]bool{
			"gui": true,
		},
	}

	if !MatchesWhen(When{OS: []string{"linux"}}, ctx) {
		t.Fatal("MatchesWhen returned false, want true")
	}
	if !PackageMatches(PackageSpec{
		WhenOS:          []string{"linux"},
		WhenLinuxDistro: []string{"ubuntu"},
		WhenFeatures:    []string{"gui"},
	}, ctx) {
		t.Fatal("PackageMatches returned false, want true")
	}
	if ComponentMatches(ComponentRef{
		WhenOS:       []string{"mac"},
		WhenFeatures: []string{"gui"},
	}, ctx) {
		t.Fatal("ComponentMatches returned true, want false")
	}
}
