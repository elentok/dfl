package packagemgr

import "strings"

func (r Runner) findMissingPipx(packages []string) ([]string, error) {
	output, err := r.exec().Output("pipx", "list", "--short")
	if err != nil {
		return nil, err
	}

	installed := map[string]bool{}
	for _, line := range splitLines(output) {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			installed[fields[0]] = true
		}
	}

	var missing []string
	for _, pkg := range packages {
		if !installed[pkg] {
			missing = append(missing, pkg)
		}
	}
	return missing, nil
}

func (r Runner) installPipxPkgs(missing []string) error {
	for _, pkg := range missing {
		if err := r.exec().Run(r.stdout(), r.stderr(), "pipx", "install", pkg); err != nil {
			return err
		}
	}
	return nil
}
