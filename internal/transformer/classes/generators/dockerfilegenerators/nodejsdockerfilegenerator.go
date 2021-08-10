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
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudrecipes/packagejson"
	"github.com/konveyor/move2kube/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// NodejsDockerfileGenerator implements the Transformer interface
type NodejsDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *NodejsDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *NodejsDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *NodejsDockerfileGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *NodejsDockerfileGenerator) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	packagejsondata, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, nil, nil
	}
	p, err := packagejson.Parse(packagejsondata)
	if err != nil {
		logrus.Debugf("Found package.json, but unable to parse it to get project name. Ignoring : %s", err)
		return nil, nil, nil
	}
	if p.Name == "" {
		err = fmt.Errorf("unable to get project name of nodejs project at %s. Ignoring", dir)
		return nil, nil, err
	}
	namedServices = map[string]transformertypes.ServicePlan{
		p.Name: []transformertypes.TransformerPlan{{
			Mode:              t.Config.Spec.Mode,
			ArtifactTypes:     []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {dir},
			},
			Configs: map[string]interface{}{
				artifacts.DockerfileTemplateConfigConfigType: map[string]interface{}{
					"Ports": []string{"8080"}, //TODO: Write parser to parse and identify port
				},
			},
		}},
	}
	return namedServices, nil, nil
}

// Transform transforms the artifacts
func (t *NodejsDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	return Transform(t.Config, t.Env, newArtifacts)
}
