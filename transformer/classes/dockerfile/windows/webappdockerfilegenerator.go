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

package windows

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/konveyor/move2kube/types/source/dotnet"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// WebTemplateConfig implements .Net Web config interface
type WebTemplateConfig struct {
	Ports            []int32
	AppName          string
	BaseImageVersion string
}

// WinWebAppDockerfileGenerator implements the Transformer interface
type WinWebAppDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *WinWebAppDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *WinWebAppDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *WinWebAppDockerfileGenerator) DirectoryDetect(dir string) (namedServices map[string][]transformertypes.Artifact, err error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Error while trying to read directory: %s", err)
		return nil, err
	}
	appName := ""
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
				logrus.Debugf("Could not read the project file: %s", err)
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
			if isSLProj {
				continue
			}

			appName = strings.TrimSuffix(filepath.Base(de.Name()), filepath.Ext(de.Name()))
		}

		// Exit soon of after the solution file is found
		break
	}

	if appName == "" {
		return nil, nil
	}

	namedServices = map[string][]transformertypes.Artifact{
		appName: {{
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {dir},
			},
		}},
	}
	return namedServices, nil
}

// Transform transforms the artifacts
func (t *WinWebAppDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}

		var detectedPorts []int32
		detectedPorts = append(detectedPorts, 8080) //TODO: Write parser to parse and identify port
		detectedPorts = commonqa.GetPortsForService(detectedPorts, a.Name)
		var webConfig WebTemplateConfig
		webConfig.AppName = a.Name
		webConfig.Ports = detectedPorts
		webConfig.BaseImageVersion = dotnet.DefaultBaseImageVersion

		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(sConfig.ServiceName)
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: webConfig,
		})
		paths := a.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, "Dockerfile")}
		p := transformertypes.Artifact{
			Name:     sImageName.ImageName,
			Artifact: artifacts.DockerfileArtifactType,
			Paths:    paths,
			Configs: map[string]interface{}{
				artifacts.ImageNameConfigType: sImageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:     sConfig.ServiceName,
			Artifact: artifacts.DockerfileForServiceArtifactType,
			Paths:    a.Paths,
			Configs: map[string]interface{}{
				artifacts.ImageNameConfigType: sImageName,
				artifacts.ServiceConfigType:   sConfig,
			},
		}
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}
