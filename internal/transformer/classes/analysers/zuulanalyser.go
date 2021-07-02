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

package analysers

import (
	//  "encoding/xml"
	//  "github.com/creekorful/mvnparser"
	//  "path/filepath"
	//  "io/ioutil"
	 "github.com/konveyor/move2kube/environment"
	 "github.com/konveyor/move2kube/internal/common"
	 //"github.com/konveyor/move2kube/internal/transformer/classes/analysers/compose"
	 //collecttypes "github.com/konveyor/move2kube/types/collection"
	 irtypes "github.com/konveyor/move2kube/types/ir"
	 plantypes "github.com/konveyor/move2kube/types/plan"
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
	 Env    environment.Environment
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
func (t *ZuulAnalyser) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	 t.Config = tc
	 t.Env = env
	 return nil
}
 
// GetConfig returns the transformer config
func (t *ZuulAnalyser) GetConfig() (transformertypes.Transformer, environment.Environment) {
	 return t.Config, t.Env
}
 
// BaseDirectoryDetect runs detect in base directory
func (t *ZuulAnalyser) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	namedServices = map[string]plantypes.Service{}
	 yamlpaths, err := common.GetFilesByExt(dir, []string{".yaml", ".yml"})
	 if err != nil {
		 logrus.Errorf("Unable to fetch yaml files at path %s Error: %q", dir, err)
		 return nil, nil, err
	 }
	 for _, path := range yamlpaths {
		 z := Zuul{}
		 if err := common.ReadYaml(path, &z); err != nil || z.ZuulSpec.RouteSpec == nil{
			 continue
		 }

		 for servicename, routepath := range z.ZuulSpec.RouteSpec {

			// TODO: routepath (ant style) to regex

			routepath = routepath[:len(routepath)-2]
			ct := plantypes.Transformer{
				Mode:                   transformertypes.ModeContainer,
				ArtifactTypes:          []transformertypes.ArtifactType{irtypes.IRArtifactType},
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
			namedServices[servicename] = append(namedServices[servicename], ct)
		 }
	 }
	 return namedServices, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *ZuulAnalyser) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {

	 
	 return nil, nil, nil
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
		var pConfig artifacts.PlanConfig
		err = a.GetConfig(artifacts.PlanConfigType, &pConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", pConfig, err)
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
		serviceConfig := irtypes.Service{Name: sConfig.ServiceName, ServiceRelPath: config.ServiceRelativePath, Annotations: map[string]string{common.ExposeSelector: common.AnnotationLabelValue} }
		ir.Services[sConfig.ServiceName] = serviceConfig
		artifactsCreated = append(artifactsCreated, transformertypes.Artifact{
		Name:     pConfig.PlanName,
		Artifact: irtypes.IRArtifactType,
		Configs: map[transformertypes.ConfigType]interface{}{
			irtypes.IRConfigType:     ir,
			artifacts.PlanConfigType: pConfig,
		},
		})
	}
	return nil, artifactsCreated, nil
}
