package packagemgr

import "encoding/json"

func (r Runner) findMissingNPM(packages []string) ([]string, error) {
	output, err := r.exec().Output("npm", "list", "-g", "--depth=0", "--json")
	if err != nil {
		return nil, err
	}

	var payload struct {
		Dependencies map[string]json.RawMessage `json:"dependencies"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, err
	}

	var missing []string
	for _, pkg := range packages {
		if _, ok := payload.Dependencies[pkg]; !ok {
			missing = append(missing, pkg)
		}
	}
	return missing, nil
}

func (r Runner) installNPMPkgs(missing []string) error {
	args := append([]string{"install", "-g"}, missing...)
	return r.exec().Run(r.stdout(), r.stderr(), "npm", args...)
}
