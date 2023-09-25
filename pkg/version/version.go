package version

import (
	"encoding/json"
	"runtime"
	"time"
)

var (
	// Semver holds the current semver.
	Semver = "dev"
	// BuildDate holds the build date of controller-manager.
	BuildDate = "I don't remember exactly"
	// StartDate holds the start date of controller-manager.
	StartDate = time.Now()
	// GitCommit holds the git sha1.
	GitCommit = "I don't remember exactly"
)

// Version holds the version information of controller-manager.
type Version struct {
	// Semver is the semantic version of this component.
	Semver string `json:"version,omitempty"`
	// GitCommit holds the git sha1 of this component.
	GitCommit string `json:"gitCommit,omitempty"`
	// BuildDate holds the build date of this component.
	BuildDate string `json:"buildDate,omitempty"`
	// StartDate holds the start date of this component.
	StartDate time.Time `json:"startDate,omitempty"`
	// GoVersion holds the go version of this component.
	GoVersion string `json:"goVersion,omitempty"`
}

// String returns version information as a string.
func (v *Version) String() string {
	value, _ := json.Marshal(v)
	return string(value)
}

// Get returns the version information.
func Get() *Version {
	return &Version{
		Semver:    Semver,
		GitCommit: GitCommit,
		StartDate: StartDate,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}
}
