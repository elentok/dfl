package packagemgr

func (r Runner) findMissingBrew(opts InstallOptions) ([]string, error) {
	args := []string{"list", "--full-name"}
	if opts.Cask {
		args = append(args, "--cask")
	}

	output, err := r.exec().Output("brew", args...)
	if err != nil {
		return nil, err
	}

	installed := map[string]bool{}
	for _, line := range splitLines(output) {
		installed[line] = true
	}

	var missing []string
	for _, pkg := range opts.Packages {
		fullName := pkg
		if opts.Tap != "" {
			fullName = opts.Tap + "/" + pkg
		}
		if installed[pkg] || installed[fullName] {
			continue
		}
		missing = append(missing, pkg)
	}
	return missing, nil
}

func (r Runner) installBrewPkgs(missing []string, opts InstallOptions) error {
	args := []string{"install"}
	if opts.Cask {
		args = append(args, "--cask")
	}
	args = append(args, missing...)
	return r.exec().Run(r.stdout(), r.stderr(), "brew", args...)
}

func (r Runner) ensureBrewTap(tap string) error {
	output, err := r.exec().Output("brew", "tap")
	if err != nil {
		return err
	}
	for _, line := range splitLines(output) {
		if line == tap {
			return nil
		}
	}
	return r.exec().Run(r.stdout(), r.stderr(), "brew", "tap", tap)
}
