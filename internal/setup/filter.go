package setup

import (
	"fmt"

	"dfl/internal/manifest"
)

func filterComponents(components []manifest.ComponentRef, machine manifest.MachineContext, requested []string) ([]string, error) {
	requestedSet := map[string]bool{}
	for _, name := range requested {
		requestedSet[name] = true
	}

	declaredSet := map[string]bool{}
	var selected []string
	for _, component := range components {
		for _, name := range component.Names {
			declaredSet[name] = true
			if len(requestedSet) > 0 && !requestedSet[name] {
				continue
			}
			if manifest.ComponentMatches(component, machine) {
				selected = append(selected, name)
			}
		}
	}

	for name := range requestedSet {
		if !declaredSet[name] {
			return nil, fmt.Errorf("component %q is not declared in setup/default.yaml", name)
		}
	}

	return selected, nil
}
