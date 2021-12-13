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
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// EarAnalyser implements Transformer interface
type EarAnalyser struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// EarDockerfileTemplate stores parameters for the dockerfile template
type EarDockerfileTemplate struct {
	DeploymentFile                    string
	BuildContainerName                string
	DeploymentFileDirInBuildContainer string
	EnvVariables                      map[string]string
}

// Init Initializes the transformer
func (t *EarAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *EarAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *EarAnalyser) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	services = map[string][]transformertypes.Artifact{}
	earFilePaths, err := common.GetFilesInCurrentDirectory(dir, nil, []string{".*[.]ear"})
	if err != nil {
		logrus.Errorf("Error while parsing directory %s for jar file : %s", dir, err)
		return nil, err
	}
	if len(earFilePaths) == 0 {
		return nil, nil
	}
	for _, path := range earFilePaths {
		newArtifact := transformertypes.Artifact{
			Paths: map[transformertypes.PathType][]string{
				artifacts.EarPathType:         {path},
				artifacts.ProjectPathPathType: {filepath.Dir(path)},
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.EarConfigType: artifacts.EarArtifactConfig{
					DeploymentFile: filepath.Base(path),
				},
			},
		}
		services[""] = append(services[""], newArtifact)
	}
	return
}

// Transform transforms the artifacts
func (t *EarAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		a.Type = artifacts.EarArtifactType
		createdArtifacts = append(createdArtifacts, a)
	}
	return pathMappings, createdArtifacts, nil
}
