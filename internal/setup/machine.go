package setup

import (
	"os"
	"strings"

	"dfl/internal/manifest"
	runtimectx "dfl/internal/runtime"
)

func detectMachineContext(ctx runtimectx.Context) manifest.MachineContext {
	machine := manifest.MachineContext{
		OS:           string(ctx.OS),
		FeatureFlags: map[string]bool{},
	}
	if ctx.OS == runtimectx.OSLinux || ctx.OS == runtimectx.OSWSL {
		machine.LinuxDistro = detectLinuxDistro()
	}
	return machine
}

func detectLinuxDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
	}
	return ""
}
