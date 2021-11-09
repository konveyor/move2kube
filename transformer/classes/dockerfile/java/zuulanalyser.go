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
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	// ZuulServiceConfigType defines config type
	ZuulServiceConfigType transformertypes.ConfigType = "ZuulService"
)

const (
	// ZuulSpringBootFile defines path type
	ZuulSpringBootFile transformertypes.PathType = "SpringBootFile"
)

// ZuulAnalyser implements Transformer interface
type ZuulAnalyser struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// ZuulConfig defines service expose url path
type ZuulConfig struct {
	ServiceRelativePath string `yaml:"serviceRelativePath,omitempty"`
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
	return nil
}

// GetConfig returns the transformer config
func (t *ZuulAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in base directory
func (t *ZuulAnalyser) DirectoryDetect(dir string) (services map[string][]transformertypes.TransformerPlan, err error) {
	services = map[string][]transformertypes.TransformerPlan{}
	yamlpaths, err := common.GetFilesByExt(dir, []string{".yaml", ".yml"})
	if err != nil {
		logrus.Errorf("Unable to fetch yaml files at path %s Error: %q", dir, err)
		return nil, err
	}
	for _, path := range yamlpaths {
		z := Zuul{}
		if err := common.ReadYaml(path, &z); err != nil || z.ZuulSpec.RouteSpec == nil {
			continue
		}

		for servicename, routepath := range z.ZuulSpec.RouteSpec {

			// TODO: routepath (ant style) to regex

			routepath = routepath[:len(routepath)-2]
			ct := transformertypes.TransformerPlan{
				Mode:          transformertypes.ModeContainer,
				ArtifactTypes: []transformertypes.ArtifactType{irtypes.IRArtifactType},
				Configs: map[transformertypes.ConfigType]interface{}{
					ZuulServiceConfigType: ZuulConfig{
						ServiceRelativePath: routepath,
					}},
				Paths: map[transformertypes.PathType][]string{
					ZuulSpringBootFile: {
						path,
					},
				},
			}
			services[servicename] = append(services[servicename], ct)
		}
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *ZuulAnalyser) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}
		var config ZuulConfig
		err := a.GetConfig(ZuulServiceConfigType, &config)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", config, err)
			continue
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}

		ir := irtypes.NewIR()
		logrus.Debugf("Transforming %s", sConfig.ServiceName)
		//TOFIX
		serviceConfig := irtypes.Service{Name: sConfig.ServiceName, Annotations: map[string]string{common.ExposeSelector: common.AnnotationLabelValue}}
		ir.Services[sConfig.ServiceName] = serviceConfig
		artifactsCreated = append(artifactsCreated, transformertypes.Artifact{
			Name:     t.Env.GetProjectName(),
			Artifact: irtypes.IRArtifactType,
			Configs: map[transformertypes.ConfigType]interface{}{
				irtypes.IRConfigType: ir,
			},
		})
	}
	return nil, artifactsCreated, nil
}
