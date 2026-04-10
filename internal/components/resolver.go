package components

import (
	"errors"
	"os"
	"path/filepath"
)

type Kind string

const (
	KindCore  Kind = "core"
	KindExtra Kind = "extra"
)

type InstallerType string

const (
	InstallerScript InstallerType = "script"
)

type Component struct {
	Name          string
	Kind          Kind
	Root          string
	InstallerType InstallerType
	Entrypoint    string
}

var ErrComponentNotFound = errors.New("component not found")

func Resolve(repoRoot, name string) (Component, error) {
	for _, candidate := range candidates(repoRoot, name) {
		if exists(candidate.EntryPoint) {
			return Component{
				Name:          name,
				Kind:          candidate.Kind,
				Root:          candidate.Root,
				InstallerType: candidate.InstallerType,
				Entrypoint:    candidate.EntryPoint,
			}, nil
		}
	}

	return Component{}, ErrComponentNotFound
}

type candidate struct {
	Kind          Kind
	Root          string
	InstallerType InstallerType
	EntryPoint    string
}

func candidates(repoRoot, name string) []candidate {
	coreRoot := filepath.Join(repoRoot, "core", name)
	extraRoot := filepath.Join(repoRoot, "extra", name)
	coreFile := filepath.Join(repoRoot, "core", name)
	extraFile := filepath.Join(repoRoot, "extra", name)

	return []candidate{
		{
			Kind:          KindCore,
			Root:          coreRoot,
			InstallerType: InstallerScript,
			EntryPoint:    filepath.Join(coreRoot, "install"),
		},
		{
			Kind:          KindCore,
			Root:          filepath.Dir(coreFile),
			InstallerType: InstallerScript,
			EntryPoint:    coreFile,
		},
		{
			Kind:          KindExtra,
			Root:          extraRoot,
			InstallerType: InstallerScript,
			EntryPoint:    filepath.Join(extraRoot, "install"),
		},
		{
			Kind:          KindExtra,
			Root:          filepath.Dir(extraFile),
			InstallerType: InstallerScript,
			EntryPoint:    extraFile,
		},
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
