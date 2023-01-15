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

package java

import (
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

// ZuulAnalyser implements Transformer interface
type ZuulAnalyser struct {
	Config   transformertypes.Transformer
	Env      *environment.Environment
	services map[string]string
}

// ZuulSpec defines zuul specification
type ZuulSpec struct {
	RouteSpec map[string]string `yaml:"routes"`
}

// Zuul defines zuul spring boot properties file
type Zuul struct {
	ZuulSpec ZuulSpec `yaml:"zuul"`
}

// Init Initializes the transformer
func (t *ZuulAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	envSource := env.GetEnvironmentSource()
	var yamlpaths []string
	if envSource != "" {
		yamlpaths, err = common.GetFilesByExt(env.GetEnvironmentSource(), []string{".yaml", ".yml"})
		if err != nil {
			logrus.Errorf("Unable to fetch yaml files at path %s Error: %q", env.GetEnvironmentSource(), err)
			return err
		}
	}
	t.services = map[string]string{}
	for _, path := range yamlpaths {
		z := Zuul{}
		if err := common.ReadYaml(path, &z); err != nil || z.ZuulSpec.RouteSpec == nil {
			continue
		}
		for servicename, routepath := range z.ZuulSpec.RouteSpec {
			// TODO: routepath (ant style) to regex
			routepath = strings.TrimSuffix(routepath, "**")
			t.services[servicename] = routepath
		}
	}
	return nil
}

// GetConfig returns the transformer config
func (t *ZuulAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in base directory
func (t *ZuulAnalyser) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *ZuulAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		ir := irtypes.IR{}
		if err := a.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T Error: %q", ir, err)
			continue
		}
		for sn, s := range ir.Services {
			if r, ok := t.services[sn]; ok {
				if len(s.ServiceToPodPortForwardings) > 0 {
					s.ServiceToPodPortForwardings[0].ServiceRelPath = r
				}
				if s.Annotations == nil {
					s.Annotations = map[string]string{}
				}
			}
			ir.Services[sn] = s
		}
		a.Configs[irtypes.IRConfigType] = ir
		artifactsCreated = append(artifactsCreated, a)
	}
	return nil, artifactsCreated, nil
}
