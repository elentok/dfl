package packagemgr

import "strings"

func (r Runner) findMissingAPT(packages []string) ([]string, error) {
	var missing []string
	for _, pkg := range packages {
		output, err := r.exec().Output("dpkg-query", "-W", "-f=${db:Status-Status}", pkg)
		if err != nil || strings.TrimSpace(string(output)) != "installed" {
			missing = append(missing, pkg)
		}
	}
	return missing, nil
}

func (r Runner) installAPTPkgs(missing []string) error {
	args := append([]string{"apt", "install", "-y"}, missing...)
	return r.exec().Run(r.stdout(), r.stderr(), "sudo", args...)
}
