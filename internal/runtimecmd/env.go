package runtimecmd

import (
	"os"
	"path/filepath"
	"strings"
)

func WithExecutableOnPath(env []string) []string {
	exe, err := os.Executable()
	if err != nil {
		return env
	}

	exeDir := filepath.Dir(exe)
	for i, entry := range env {
		if !strings.HasPrefix(entry, "PATH=") {
			continue
		}
		env[i] = "PATH=" + exeDir + string(os.PathListSeparator) + strings.TrimPrefix(entry, "PATH=")
		return env
	}

	return append(env, "PATH="+exeDir)
}
