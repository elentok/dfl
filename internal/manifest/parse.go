package manifest

import (
	"bytes"
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
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
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}
	return nil
}
