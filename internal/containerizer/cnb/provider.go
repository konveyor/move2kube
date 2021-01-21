/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cnb

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

const (
	orderLabel string = "io.buildpacks.buildpack.order"
)

var (
	cnbwarnnotsupported = false
	cnbwarnlongwait     = true
	cnbproviders        = []provider{&dockerAPIProvider{}, &containerRuntimeProvider{}, &packProvider{}, &runcProvider{}}
	cnbprovider         provider
	providerschecked    = false
)

type order []orderEntry

type orderEntry struct {
	Group []buildpackRef `toml:"group" json:"group"`
}

type buildpackRef struct {
	buildpackInfo
	Optional bool `toml:"optional,omitempty" json:"optional,omitempty"`
}

type buildpackInfo struct {
	ID       string `toml:"id" json:"id,omitempty"`
	Version  string `toml:"version" json:"version,omitempty"`
	Homepage string `toml:"homepage,omitempty" json:"homepage,omitempty"`
}

type provider interface {
	isBuilderSupported(path string, builder string) (bool, error)
	getAllBuildpacks(builders []string) (map[string][]string, error)
}

// GetAllBuildpacks returns all buildpacks supported
func GetAllBuildpacks(builders []string) (buildpacks map[string][]string) {
	for _, cp := range cnbproviders {
		buildpacks, err := cp.getAllBuildpacks(builders)
		if err == nil && len(buildpacks) > 0 {
			return buildpacks
		}
	}
	logCNBNotSupported()
	return buildpacks
}

// IsBuilderSupported returns if a builder supports a path
func IsBuilderSupported(path string, builder string) (valid bool) {
	logCNBLongWait()
	if !providerschecked {
		for _, cp := range cnbproviders {
			valid, err := cp.isBuilderSupported(path, builder)
			if err == nil {
				cnbprovider = cp
				providerschecked = true
				return valid // if one of the builders reports no support the other builders will as well.
			}
		}
		providerschecked = true
	} else if cnbprovider != nil {
		valid, err := cnbprovider.isBuilderSupported(path, builder)
		if err == nil {
			return valid
		}
	}
	logCNBNotSupported()
	return valid
}

func logCNBLongWait() {
	if !cnbwarnlongwait {
		log.Warn("This could take a few minutes to complete.")
		cnbwarnlongwait = true
	}
}

func logCNBNotSupported() {
	if !cnbwarnnotsupported {
		log.Warn("No CNB containerizer method accessible")
		cnbwarnnotsupported = true
	}
}

func getBuildersFromLabel(label string) (buildpacks []string) {
	buildpacks = []string{}
	ogs := order{}
	err := json.Unmarshal([]byte(label), &ogs)
	if err != nil {
		log.Warnf("Unable to read order : %s", err)
		return
	}
	log.Debugf("Builder data :%s", label)
	for _, og := range ogs {
		for _, buildpackref := range og.Group {
			buildpacks = append(buildpacks, buildpackref.ID)
		}
	}
	return
}
