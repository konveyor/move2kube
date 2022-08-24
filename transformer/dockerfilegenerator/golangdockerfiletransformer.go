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

package dockerfilegenerator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const (
	defaultGoVersion = "1.18"
	// GolangModFilePathType points to the go.mod file path
	GolangModFilePathType transformertypes.PathType = "GoModFilePath"
)

// GolangDockerfileGenerator implements the Transformer interface
type GolangDockerfileGenerator struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	GolangConfig *GolangDockerfileYamlConfig
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
	t.GolangConfig = &GolangDockerfileYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.GolangConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.GolangConfig, err)
		return err
	}
	if t.GolangConfig.DefaultGoVersion == "" {
		t.GolangConfig.DefaultGoVersion = defaultGoVersion
	}
	return nil
}

// GetConfig returns the transformer config
func (t *GolangDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *GolangDockerfileGenerator) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	modFilePath := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(modFilePath)
	if err != nil {
		return nil, nil
	}
	modFile, err := modfile.Parse(modFilePath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the go.mod file at path %s . Error: %q", modFilePath, err)
	}
	prefix, _, ok := module.SplitPathVersion(modFile.Module.Mod.Path)
	if !ok {
		logrus.Errorf("Invalid module path")
		return nil, nil
	}
	serviceName := filepath.Base(prefix)
	normalizedServiceName := common.MakeStringK8sServiceNameCompliant(serviceName)
	services := map[string][]transformertypes.Artifact{
		normalizedServiceName: {{
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceDirPathType: {dir},
				GolangModFilePathType:        {modFilePath},
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.OriginalNameConfigType: artifacts.OriginalNameConfig{OriginalName: serviceName},
			},
		}},
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *GolangDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ServiceDirPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ServiceDirPathType][0], err)
			continue
		}
		serviceConfig := artifacts.ServiceConfig{}
		if err := a.GetConfig(artifacts.ServiceConfigType, &serviceConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", serviceConfig, err)
			continue
		}
		imageName := artifacts.ImageName{}
		if err := a.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", imageName, err)
		}
		if imageName.ImageName == "" {
			imageName.ImageName = common.MakeStringContainerImageNameCompliant(serviceConfig.ServiceName)
		}
		ir := irtypes.IR{}
		irPresent := true
		if err := a.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T : %s", ir, err)
		}
		data, err := os.ReadFile(a.Paths[GolangModFilePathType][0])
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
		detectedPorts := ir.GetAllServicePorts()
		if len(detectedPorts) == 0 {
			detectedPorts = append(detectedPorts, common.DefaultServicePort)
		}
		detectedPorts = commonqa.GetPortsForService(detectedPorts, `"`+a.Name+`"`)
		golangConfig := GolangTemplateConfig{
			AppName:   a.Name,
			Ports:     detectedPorts,
			GoVersion: modFile.Go.Version,
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
			Name:  imageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ServiceConfigType:   serviceConfig,
				artifacts.ImageNameConfigType: imageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:  serviceConfig.ServiceName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: a.Paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ServiceConfigType:   serviceConfig,
				artifacts.ImageNameConfigType: imageName,
			},
		}
		if irPresent {
			dfs.Configs[irtypes.IRConfigType] = ir
		}
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}
