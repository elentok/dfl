package manifest

func MatchesWhen(when When, ctx MachineContext) bool {
	return matchOS(when.OS, ctx) && true
}

func PackageMatches(pkg PackageSpec, ctx MachineContext) bool {
	return matchOS(pkg.WhenOS, ctx) &&
		matchLinuxDistro(pkg.WhenLinuxDistro, ctx) &&
		matchFeatures(pkg.WhenFeatures, ctx)
}

func StepMatches(step StepSpec, ctx MachineContext) bool {
	return matchOS(step.OS, ctx)
}

func ComponentMatches(component ComponentRef, ctx MachineContext) bool {
	return matchOS(component.WhenOS, ctx) &&
		matchLinuxDistro(component.WhenLinuxDistro, ctx) &&
		matchFeatures(component.WhenFeatures, ctx)
}

func RepoMatches(repo RepoSpec, ctx MachineContext) bool {
	return matchOS(repo.WhenOS, ctx) &&
		matchLinuxDistro(repo.WhenLinuxDistro, ctx) &&
		matchFeatures(repo.WhenFeatures, ctx)
}

func matchOS(values []string, ctx MachineContext) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == ctx.OS {
			return true
		}
	}
	return false
}

func matchLinuxDistro(values []string, ctx MachineContext) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == ctx.LinuxDistro {
			return true
		}
	}
	return false
}

func matchFeatures(values []string, ctx MachineContext) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if !ctx.FeatureFlags[value] {
			return false
		}
	}
	return true
}
