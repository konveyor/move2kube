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

package dockerfile

import (
	"io/ioutil"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const (
	// GolangModFilePathType points to the go.mod file path
	GolangModFilePathType transformertypes.PathType = "GoModFilePath"
)

// GolangDockerfileGenerator implements the Transformer interface
type GolangDockerfileGenerator struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	GolangConfig GolangDockerfileYamlConfig
}

// GolangTemplateConfig implements Golang config interface
type GolangTemplateConfig struct {
	Ports     []int32
	AppName   string
	GoVersion string
}

// GolangDockerfileYamlConfig represents the configuration of the Golang dockerfile
type GolangDockerfileYamlConfig struct {
	DefaultGoVersion string `yaml:"defaultGoVersion"`
}

// Init Initializes the transformer
func (t *GolangDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.GolangConfig = GolangDockerfileYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, &t.GolangConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.GolangConfig, err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *GolangDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *GolangDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	modFilePath := filepath.Join(dir, "go.mod")
	data, err := ioutil.ReadFile(modFilePath)
	if err != nil {
		return nil, nil
	}
	modFile, err := modfile.Parse(modFilePath, data, nil)
	if err != nil {
		logrus.Errorf("Error while parsing the go.mod file : %s", err)
		return nil, nil
	}
	prefix, _, ok := module.SplitPathVersion(modFile.Module.Mod.Path)
	if !ok {
		logrus.Errorf("Invalid module path")
		return nil, nil
	}
	serviceName := filepath.Base(prefix)
	services = map[string][]transformertypes.Artifact{
		serviceName: {{
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {dir},
				GolangModFilePathType:         {modFilePath},
			},
		}},
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *GolangDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
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
		data, err := ioutil.ReadFile(a.Paths[GolangModFilePathType][0])
		if err != nil {
			logrus.Errorf("Error while reading the go.mod file : %s", err)
			return nil, nil, nil
		}
		modFile, err := modfile.Parse(a.Paths[GolangModFilePathType][0], data, nil)
		if err != nil {
			logrus.Errorf("Error while parsing the go.mod file : %s", err)
			return nil, nil, nil
		}
		if modFile.Go == nil {
			logrus.Debugf("Didn't find the Go version in the go.mod file at path %s, selecting Go version %s", a.Paths[GolangModFilePathType][0], t.GolangConfig.DefaultGoVersion)
			modFile.Go.Version = t.GolangConfig.DefaultGoVersion
		}
		var detectedPorts []int32
		detectedPorts = append(detectedPorts, 8080) //TODO: Write parser to parse and identify port
		detectedPorts = commonqa.GetPortsForService(detectedPorts, a.Name)
		var golangConfig GolangTemplateConfig
		golangConfig.AppName = a.Name
		golangConfig.Ports = detectedPorts
		golangConfig.GoVersion = modFile.Go.Version

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
			TemplateConfig: golangConfig,
		})
		paths := a.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, common.DefaultDockerfileName)}
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
