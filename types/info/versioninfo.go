/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package info

import (
	"runtime"

	semver "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
)

var (
	// Update this whenever making a new release.
	// The version is of the format Major.Minor.Patch[-Prerelease][+BuildMetadata]
	// Given a version number MAJOR.MINOR.PATCH, increment the:
	// MAJOR version when you make incompatible API changes,
	// MINOR version when you add functionality in a backwards compatible manner, and
	// PATCH version when you make backwards compatible bug fixes.
	// Additional labels for pre-release and build metadata are available as extensions to the MAJOR.MINOR.PATCH format.
	// For more details about semver 2 see https://semver.org/
	version = "v0.3.0"

	// metadata is extra build time data
	buildmetadata = ""
	// gitCommit is the git sha1
	gitCommit = ""
	// gitTreeState is the state of the git tree
	gitTreeState = ""
)

// GetVersion returns the semver string of the version
func GetVersion() string {
	if buildmetadata == "" {
		return version
	}
	return version + "+" + buildmetadata
}

// GetVersionInfo returns version info
func GetVersionInfo() VersionInfo {
	v := VersionInfo{
		Version:      GetVersion(),
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		GoVersion:    runtime.Version(),
		Platform:     runtime.GOOS + "/" + runtime.GOARCH,
	}
	return v
}

// VersionInfo describes the compile time information.
type VersionInfo struct {
	// Version is the current semver.
	Version string `yaml:"version,omitempty"`
	// GitCommit is the git sha1.
	GitCommit string `yaml:"gitCommit,omitempty"`
	// GitTreeState is the state of the git tree.
	GitTreeState string `yaml:"gitTreeState,omitempty"`
	// GoVersion is the version of the Go compiler used.
	GoVersion string `yaml:"goVersion,omitempty"`
	// Platform gives the OS and ISA the app is running on
	Platform string `yaml:"platform,omitempty"`
}

// IsSameVersion checks if two versions are same and logs a message if the version is newer or older
func (v *VersionInfo) IsSameVersion() bool {
	binaryversion, err := semver.NewVersion(GetVersion())
	if err != nil {
		logrus.Warnf("Unable to load current version of binary : %s", err)
		return false
	}
	objversion, err := semver.NewVersion(v.Version)
	if err != nil {
		logrus.Warnf("Unable to load current version : %s", err)
		return false
	}

	compare := binaryversion.Compare(objversion)

	if compare == 0 {
		return true
	}
	if compare < 0 {
		logrus.Warnf("The file version (%s) is newer than the binary version (%s).", objversion, binaryversion)
	} else {
		logrus.Warnf("The file version (%s) is older than the binary version (%s).", objversion, binaryversion)
	}
	return false
}
