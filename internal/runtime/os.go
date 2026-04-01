package runtime

type OSType string

const (
	OSUnknown OSType = "unknown"
	OSMac     OSType = "mac"
	OSLinux   OSType = "linux"
	OSWSL     OSType = "wsl"
)
