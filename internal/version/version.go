// Package version exposes sanitized build metadata.
package version

import "runtime"

var (
	// Version is the semantic version injected at build time.
	Version = "dev"
	// Commit is the source commit injected at build time.
	Commit = "unknown"
	// BuildDate is the RFC 3339 build timestamp injected at build time.
	BuildDate = "unknown"
)

// Info contains public, non-sensitive build metadata.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// Current returns the current build metadata.
func Current() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}
}
