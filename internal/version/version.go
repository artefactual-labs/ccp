package version

import (
	"runtime/debug"
)

var (
	version   = "dev"
	gitCommit = "unknown"
)

func init() {
	if version != "dev" && gitCommit != "unknown" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			gitCommit = s.Value
		}
	}
}

func Version() string {
	return version
}

func GitCommit() string {
	return gitCommit
}
