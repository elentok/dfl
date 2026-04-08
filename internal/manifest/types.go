package manifest

type InstallManifest struct {
	Name     string            `yaml:"name"`
	Kind     string            `yaml:"kind"`
	When     When              `yaml:"when"`
	Symlinks map[string]string `yaml:"symlinks"`
	Copies   map[string]string `yaml:"copies"`
	Mkdirs   []string          `yaml:"mkdirs"`
	Packages PackageGroups     `yaml:"packages"`
	Steps    []StepSpec        `yaml:"steps"`
}

type SetupManifest struct {
	When         When           `yaml:"when"`
	RepoDefaults RepoDefaults   `yaml:"repo_defaults"`
	Components   []ComponentRef `yaml:"components"`
	Packages     PackageGroups  `yaml:"packages"`
	Repos        []RepoSpec     `yaml:"repos"`
	Steps        []StepSpec     `yaml:"steps"`
}

type When struct {
	OS []string `yaml:"os"`
}

type PackageGroups struct {
	Brew  []PackageSpec `yaml:"brew"`
	Apt   []PackageSpec `yaml:"apt"`
	NPM   []PackageSpec `yaml:"npm"`
	Pipx  []PackageSpec `yaml:"pipx"`
	Cargo []PackageSpec `yaml:"cargo"`
	Snap  []PackageSpec `yaml:"snap"`
}

type PackageSpec struct {
	Names           []string `yaml:"names"`
	Tap             string   `yaml:"tap"`
	Cask            bool     `yaml:"cask"`
	WhenOS          []string `yaml:"when_os"`
	WhenLinuxDistro []string `yaml:"when_linux_distro"`
	WhenFeatures    []string `yaml:"when_features"`
}

type StepSpec struct {
	Name  string   `yaml:"name"`
	OS    []string `yaml:"os"`
	If    string   `yaml:"if"`
	IfNot string   `yaml:"if_not"`
	CWD   string   `yaml:"cwd"`
	Sudo  bool     `yaml:"sudo"`
	Run   string   `yaml:"run"`
}

type RepoDefaults struct {
	Transport string `yaml:"transport"`
}

type ComponentRef struct {
	Names           []string `yaml:"names"`
	WhenOS          []string `yaml:"when_os"`
	WhenLinuxDistro []string `yaml:"when_linux_distro"`
	WhenFeatures    []string `yaml:"when_features"`
}

type RepoSpec struct {
	Name            string   `yaml:"name"`
	Path            string   `yaml:"path"`
	GitHub          string   `yaml:"github"`
	URL             string   `yaml:"url"`
	Transport       string   `yaml:"transport"`
	WhenOS          []string `yaml:"when_os"`
	WhenLinuxDistro []string `yaml:"when_linux_distro"`
	WhenFeatures    []string `yaml:"when_features"`
}

type MachineContext struct {
	OS           string
	LinuxDistro  string
	FeatureFlags map[string]bool
}
