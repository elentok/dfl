package packagemgr

import "strings"

func (r Runner) findMissingSnap(packages []string) ([]string, error) {
	output, err := r.exec().Output("snap", "list")
	if err != nil {
		return nil, err
	}

	installed := map[string]bool{}
	for i, line := range splitLines(output) {
		if i == 0 {
			continue
		}
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

func (r Runner) installSnapPkgs(missing []string) error {
	for _, pkg := range missing {
		if err := r.exec().Run(r.stdout(), r.stderr(), "sudo", "snap", "install", pkg); err != nil {
			return err
		}
	}
	return nil
}
