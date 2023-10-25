/*
 * Copyright 2021 Aapeli.Smith<aapeli.nian@gmail.com>.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package version

import (
	"encoding/json"
	"github.com/fatedier/frp/pkg/util/version"
	"runtime"
	"time"
)

var (
	// Semver holds the current semver.
	Semver = "dev"
	// BuildDate holds the build date of controller.
	BuildDate = "I don't remember exactly"
	// StartDate holds the start date of controller.
	StartDate = time.Now()
	// GitCommit holds the git sha1.
	GitCommit = "I don't remember exactly"
)

// Version holds the version information of controller.
type Version struct {
	FrpVersion string `json:"frpVersion,omitempty"`
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
		FrpVersion: version.Full(),
		Semver:     Semver,
		GitCommit:  GitCommit,
		StartDate:  StartDate,
		BuildDate:  BuildDate,
		GoVersion:  runtime.Version(),
	}
}
