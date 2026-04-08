package manifest

type PackageGroup struct {
	Manager string
	Spec    PackageSpec
}

func (p PackageGroups) All() map[string][]PackageSpec {
	return map[string][]PackageSpec{
		"brew":  p.Brew,
		"apt":   p.Apt,
		"npm":   p.NPM,
		"pipx":  p.Pipx,
		"cargo": p.Cargo,
		"snap":  p.Snap,
	}
}

func (p PackageGroups) Managers() []string {
	return []string{"brew", "apt", "npm", "pipx", "cargo", "snap"}
}

func (p PackageGroups) Entries() []PackageGroup {
	var entries []PackageGroup
	for _, manager := range p.Managers() {
		for _, pkg := range p.All()[manager] {
			entries = append(entries, PackageGroup{Manager: manager, Spec: pkg})
		}
	}
	return entries
}
