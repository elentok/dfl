package manifest

import (
	"fmt"
	"strings"
)

var supportedPackageManagers = map[string]bool{
	"brew":  true,
	"apt":   true,
	"npm":   true,
	"pipx":  true,
	"cargo": true,
	"snap":  true,
}

var supportedOS = map[string]bool{
	"mac":   true,
	"linux": true,
	"wsl":   true,
}

var supportedKinds = map[string]bool{
	"":      true,
	"core":  true,
	"extra": true,
}

var supportedTransport = map[string]bool{
	"":        true,
	"inherit": true,
	"ssh":     true,
	"https":   true,
}

func ValidateInstall(m InstallManifest) error {
	if !supportedKinds[m.Kind] {
		return fmt.Errorf("unsupported manifest kind %q", m.Kind)
	}
	if err := validateWhen(m.When); err != nil {
		return err
	}
	for _, pkg := range m.Packages {
		if err := validatePackage(pkg); err != nil {
			return err
		}
	}
	for _, step := range m.Steps {
		if err := validateStep(step); err != nil {
			return err
		}
	}
	return nil
}

func ValidateSetup(m SetupManifest) error {
	if err := validateWhen(m.When); err != nil {
		return err
	}
	if !supportedTransport[m.RepoDefaults.Transport] {
		return fmt.Errorf("unsupported repo default transport %q", m.RepoDefaults.Transport)
	}
	for _, component := range m.Components {
		if len(component.Names) == 0 {
			return fmt.Errorf("setup component names are required")
		}
		for _, name := range component.Names {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("setup component names must not be empty")
			}
		}
		if err := validateConditionalLists(component.WhenOS, component.WhenLinuxDistro, component.WhenFeatures); err != nil {
			return err
		}
	}
	for _, pkg := range m.Packages {
		if err := validatePackage(pkg); err != nil {
			return err
		}
	}
	for _, repo := range m.Repos {
		if err := validateRepo(repo); err != nil {
			return err
		}
	}
	for _, step := range m.Steps {
		if err := validateStep(step); err != nil {
			return err
		}
	}
	return nil
}

func validatePackage(pkg PackageSpec) error {
	if !supportedPackageManagers[pkg.Manager] {
		return fmt.Errorf("unsupported package manager %q", pkg.Manager)
	}
	if len(pkg.Names) == 0 {
		return fmt.Errorf("package names are required for manager %q", pkg.Manager)
	}
	for _, name := range pkg.Names {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("package names must not be empty")
		}
	}
	if pkg.Tap != "" && pkg.Manager != "brew" {
		return fmt.Errorf("tap is only supported for brew packages")
	}
	if pkg.Cask && pkg.Manager != "brew" {
		return fmt.Errorf("cask is only supported for brew packages")
	}
	return validateConditionalLists(pkg.WhenOS, pkg.WhenLinuxDistro, pkg.WhenFeatures)
}

func validateStep(step StepSpec) error {
	if step.Name == "" {
		return fmt.Errorf("step name is required")
	}
	if err := validateOSList(step.OS); err != nil {
		return err
	}
	return nil
}

func validateRepo(repo RepoSpec) error {
	if repo.Name == "" {
		return fmt.Errorf("repo name is required")
	}
	if repo.Path == "" {
		return fmt.Errorf("repo path is required")
	}
	if (repo.GitHub == "" && repo.URL == "") || (repo.GitHub != "" && repo.URL != "") {
		return fmt.Errorf("repo %q must define exactly one of github or url", repo.Name)
	}
	if !supportedTransport[repo.Transport] {
		return fmt.Errorf("unsupported repo transport %q", repo.Transport)
	}
	return validateConditionalLists(repo.WhenOS, repo.WhenLinuxDistro, repo.WhenFeatures)
}

func validateWhen(when When) error {
	return validateOSList(when.OS)
}

func validateConditionalLists(osList, distroList, featureList []string) error {
	if err := validateOSList(osList); err != nil {
		return err
	}
	for _, value := range distroList {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("linux distro values must not be empty")
		}
	}
	for _, value := range featureList {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("feature values must not be empty")
		}
	}
	return nil
}

func validateOSList(values []string) error {
	for _, value := range values {
		if !supportedOS[value] {
			return fmt.Errorf("unsupported os value %q", value)
		}
	}
	return nil
}
