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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/konveyor/move2kube-wasm/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// WarAnalyser implements Transformer interface
type WarAnalyser struct {
	Config    transformertypes.Transformer
	Env       *environment.Environment
	WarConfig *WarYamlConfig
}

// WarYamlConfig stores the war related information
type WarYamlConfig struct {
	JavaVersion string `yaml:"defaultJavaVersion"`
}

// WarDockerfileTemplate stores parameters for the dockerfile template
type WarDockerfileTemplate struct {
	DeploymentFile                    string
	BuildContainerName                string
	DeploymentFileDirInBuildContainer string
	EnvVariables                      map[string]string
}

// Init Initializes the transformer
func (t *WarAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.WarConfig = &WarYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.WarConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.WarConfig, err)
		return err
	}
	if t.WarConfig.JavaVersion == "" {
		t.WarConfig.JavaVersion = defaultJavaVersion
	}
	return nil
}

// GetConfig returns the transformer config
func (t *WarAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *WarAnalyser) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	services := map[string][]transformertypes.Artifact{}
	paths, err := common.GetFilesInCurrentDirectory(dir, nil, []string{`\.war$`})
	if err != nil {
		return services, fmt.Errorf("failed to look for .war archives in the directory %s . Error: %q", dir, err)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	for _, path := range paths {
		serviceName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		normalizedServiceName := common.MakeStringK8sServiceNameCompliant(serviceName)
		imageName := common.MakeStringContainerImageNameCompliant(serviceName)
		newArtifact := transformertypes.Artifact{
			Paths: map[transformertypes.PathType][]string{
				artifacts.WarPathType:        {path},
				artifacts.ServiceDirPathType: {dir},
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.WarConfigType: artifacts.WarArtifactConfig{
					DeploymentFilePath: filepath.Base(path),
					JavaVersion:        t.WarConfig.JavaVersion,
				},
				artifacts.ImageNameConfigType: artifacts.ImageName{ImageName: imageName},
			},
		}
		services[normalizedServiceName] = append(services[normalizedServiceName], newArtifact)
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *WarAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		a.Type = artifacts.WarArtifactType
		createdArtifacts = append(createdArtifacts, a)
	}
	return pathMappings, createdArtifacts, nil
}
