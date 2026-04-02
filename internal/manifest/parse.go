package manifest

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

func ParseInstallFile(path string) (InstallManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return InstallManifest{}, err
	}
	return ParseInstallBytes(data)
}

func ParseInstallBytes(data []byte) (InstallManifest, error) {
	var manifest InstallManifest
	if err := decodeStrict(data, &manifest); err != nil {
		return InstallManifest{}, err
	}
	if err := ValidateInstall(manifest); err != nil {
		return InstallManifest{}, err
	}
	return manifest, nil
}

func ParseSetupFile(path string) (SetupManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SetupManifest{}, err
	}
	return ParseSetupBytes(data)
}

func ParseSetupBytes(data []byte) (SetupManifest, error) {
	var manifest SetupManifest
	if err := decodeStrict(data, &manifest); err != nil {
		return SetupManifest{}, err
	}
	if err := ValidateSetup(manifest); err != nil {
		return SetupManifest{}, err
	}
	return manifest, nil
}

func decodeStrict(data []byte, target any) error {
	md, err := toml.Decode(string(data), target)
	if err != nil {
		return err
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		parts := make([]string, 0, len(undecoded))
		for _, item := range undecoded {
			parts = append(parts, item.String())
		}
		return fmt.Errorf("unknown manifest fields: %s", strings.Join(parts, ", "))
	}
	return nil
}
