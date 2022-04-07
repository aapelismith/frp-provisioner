package version

import (
	"runtime"
	"time"
)

var (
	// Semver holds the current version of controller-manager.
	Semver = "dev"
	// BuildDate holds the build date of controller-manager.
	BuildDate = "I don't remember exactly"
	// StartDate holds the start date of controller-manager.
	StartDate = time.Now()
	// GitCommit The commit ID of the current commit.
	GitCommit = "I don't remember exactly"
)

// Version describes compile time information.
type Version struct {
	// Semver is the current semver.
	Semver string `json:"version,omitempty"`
	// GitCommit is the git sha1.
	GitCommit string `json:"git_commit,omitempty"`
	// BuildDate holds the build date of this component.
	BuildDate string `json:"build_date,omitempty"`
	// StartDate holds the start date of this component.
	StartDate time.Time `json:"startDate,omitempty"`
	// GoVersion is the version of the Go compiler used.
	GoVersion string `json:"go_version,omitempty"`
}

// Get returns build info
func Get() *Version {
	return &Version{
		Semver:    Semver,
		GitCommit: GitCommit,
		StartDate: StartDate,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}
}
