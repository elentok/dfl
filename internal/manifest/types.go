package manifest

type InstallManifest struct {
	Name     string            `toml:"name"`
	Kind     string            `toml:"kind"`
	When     When              `toml:"when"`
	Symlinks map[string]string `toml:"symlinks"`
	Copies   map[string]string `toml:"copies"`
	Mkdirs   []string          `toml:"mkdirs"`
	Packages []PackageSpec     `toml:"packages"`
	Steps    []StepSpec        `toml:"steps"`
}

type SetupManifest struct {
	When         When           `toml:"when"`
	RepoDefaults RepoDefaults   `toml:"repo_defaults"`
	Components   []ComponentRef `toml:"components"`
	Packages     []PackageSpec  `toml:"packages"`
	Repos        []RepoSpec     `toml:"repos"`
	Steps        []StepSpec     `toml:"steps"`
}

type When struct {
	OS []string `toml:"os"`
}

type PackageSpec struct {
	Manager         string   `toml:"manager"`
	Names           []string `toml:"names"`
	Tap             string   `toml:"tap"`
	Cask            bool     `toml:"cask"`
	WhenOS          []string `toml:"when_os"`
	WhenLinuxDistro []string `toml:"when_linux_distro"`
	WhenFeatures    []string `toml:"when_features"`
}

type StepSpec struct {
	Name  string   `toml:"name"`
	OS    []string `toml:"os"`
	If    string   `toml:"if"`
	IfNot string   `toml:"if_not"`
	CWD   string   `toml:"cwd"`
	Sudo  bool     `toml:"sudo"`
	Run   string   `toml:"run"`
}

type RepoDefaults struct {
	Transport string `toml:"transport"`
}

type ComponentRef struct {
	Name            string   `toml:"name"`
	WhenOS          []string `toml:"when_os"`
	WhenLinuxDistro []string `toml:"when_linux_distro"`
	WhenFeatures    []string `toml:"when_features"`
}

type RepoSpec struct {
	Name            string   `toml:"name"`
	Path            string   `toml:"path"`
	GitHub          string   `toml:"github"`
	URL             string   `toml:"url"`
	Transport       string   `toml:"transport"`
	WhenOS          []string `toml:"when_os"`
	WhenLinuxDistro []string `toml:"when_linux_distro"`
	WhenFeatures    []string `toml:"when_features"`
}

type MachineContext struct {
	OS           string
	LinuxDistro  string
	FeatureFlags map[string]bool
}
