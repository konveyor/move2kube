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

package dockerfilegenerators

import (
	"io/ioutil"
	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"

	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const (
	goVersion = "1.16"
	goExt     = ".go"
	golang    = "golang"
)

// GolangDockerfileGenerator implements the Transformer interface
type GolangDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *GolangDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *GolangDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *GolangDockerfileGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *GolangDockerfileGenerator) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	modFilePath := filepath.Join(dir, "go.mod")
	data, err := ioutil.ReadFile(modFilePath)
	if err != nil {
		logrus.Errorf("Error while reading the go.mod file : %s", err)
		return nil, nil, nil
	}
	modFile, err := modfile.Parse(modFilePath, data, nil)
	if err != nil {
		logrus.Errorf("Error while parsing the go.mod file : %s", err)
		return nil, nil, nil
	}
	if modFile.Go == nil {
		// logrus.Errorf("Didn't find the Go version in the go.mod file at path %s, selecting Go version %s", modFilePath, goVersion)
		modFile.Go.Version = goVersion
	}
	prefix, _, ok := module.SplitPathVersion(modFile.Module.Mod.Path)
	if !ok {
		logrus.Errorf("Invalid module path")
		return nil, nil, nil
	}
	serviceName := filepath.Base(prefix)
	var detectedPorts []int32
	detectedPorts = append(detectedPorts, 8080)
	detectedPorts = commonqa.GetPortsForService(detectedPorts)
	namedServices = map[string]transformertypes.ServicePlan{
		serviceName: []transformertypes.TransformerPlan{{
			Mode:              t.Config.Spec.Mode,
			ArtifactTypes:     []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {dir},
			},
			Configs: map[string]interface{}{
				artifacts.DockerfileTemplateConfigConfigType: map[string]interface{}{
					"ports":      detectedPorts, //TODO: Write parser to parse and identify port
					"app_name":   "app-bin",
					"go_version": modFile.Go.Version,
				},
			},
		}},
	}
	return namedServices, nil, nil
}

// Transform transforms the artifacts
func (t *GolangDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	return transform(t.Config, t.Env, newArtifacts)
}
