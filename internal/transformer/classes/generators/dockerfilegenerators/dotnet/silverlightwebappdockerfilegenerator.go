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

package dotnet

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/transformer/classes/generators/dockerfilegenerators"
	"github.com/konveyor/move2kube/types/source/dotnet"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// WinSilverLightWebAppDockerfileGenerator implements the Transformer interface
type WinSilverLightWebAppDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *WinSilverLightWebAppDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *WinSilverLightWebAppDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *WinSilverLightWebAppDockerfileGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *WinSilverLightWebAppDockerfileGenerator) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Error while trying to read directory: %s", err)
		return nil, nil, err
	}
	appName := ""
	ports := make([]string, 0)
	for _, de := range dirEntries {
		if filepath.Ext(de.Name()) != dotnet.CsSln {
			continue
		}
		csProjPaths, err := parseSolutionFile(filepath.Join(dir, de.Name()))
		if err != nil {
			logrus.Errorf("%s", err)
			continue
		}

		if len(csProjPaths) == 0 {
			logrus.Errorf("No projects available for the solution: %s", de.Name())
			continue
		}

		for _, csPath := range csProjPaths {
			projPath := filepath.Join(strings.TrimSpace(dir), strings.TrimSpace(csPath))
			byteValue, err := ioutil.ReadFile(projPath)
			if err != nil {
				logrus.Errorf("Could not read the project file: %s", err)
				continue
			}

			configuration := dotnet.CSProj{}
			err = xml.Unmarshal(byteValue, &configuration)
			if err != nil {
				logrus.Errorf("Could not parse the project file: %s", err)
				continue
			}

			if configuration.PropertyGroup == nil ||
				configuration.PropertyGroup.TargetFrameworkVersion == "" ||
				!dotnet.FourXPattern.MatchString(configuration.PropertyGroup.TargetFrameworkVersion) {
				logrus.Errorf("Not a supported dotnet framework [%s]", configuration.PropertyGroup.TargetFrameworkVersion)
				continue
			}

			isWebProj, err := isWeb(configuration)
			if err != nil {
				logrus.Errorf("%s", err)
				continue
			}
			if !isWebProj {
				continue
			}

			isSLProj, err := isSilverlight(configuration)
			if err != nil {
				logrus.Errorf("%s", err)
				continue
			}
			if !isSLProj {
				continue
			}

			appName = strings.TrimSuffix(filepath.Base(de.Name()), filepath.Ext(de.Name()))
		}

		// Exit soon of after the solution file is found
		break
	}

	if appName == "" {
		return nil, nil, nil
	}

	namedServices = map[string]transformertypes.ServicePlan{
		appName: []transformertypes.TransformerPlan{{
			Mode:              t.Config.Spec.Mode,
			ArtifactTypes:     []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {dir},
			},
			Configs: map[string]interface{}{
				artifacts.DockerfileTemplateConfigConfigType: map[string]interface{}{
					"Ports":   ports,
					"appName": appName,
				},
			},
		}},
	}
	return namedServices, nil, nil
}

// Transform transforms the artifacts
func (t *WinSilverLightWebAppDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	return dockerfilegenerators.Transform(t.Config, t.Env, newArtifacts)
}