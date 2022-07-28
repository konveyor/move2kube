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
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// RustDockerfileGenerator implements the Transformer interface
type RustDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

const (
	cargoTomlFile  = "Cargo.toml"
	rocketTomlFile = "Rocket.toml"
)

// RustTemplateConfig implements Nodejs config interface
type RustTemplateConfig struct {
	Port          int32
	AppName       string
	RocketToml    string
	RocketAddress string
}

// CargoTomlConfig implements Cargo.toml config interface
type CargoTomlConfig struct {
	Package PackageInfo
}

//PackageInfo implements package properties
type PackageInfo struct {
	Name    string
	Version string
}

// RocketTomlConfig implements Rocket.toml config interface
type RocketTomlConfig struct {
	Global GlobalParameters
}

//GlobalParameters implements global properties in Rocket.toml
type GlobalParameters struct {
	Address string
	Port    int32
}

// Init Initializes the transformer
func (t *RustDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *RustDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *RustDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	if _, err := os.Stat(filepath.Join(dir, cargoTomlFile)); err == nil {
		var cargoTomlConfig CargoTomlConfig
		if _, err := toml.DecodeFile(filepath.Join(dir, cargoTomlFile), &cargoTomlConfig); err == nil {
			serviceName := cargoTomlConfig.Package.Name
			services = map[string][]transformertypes.Artifact{
				serviceName: {{
					Paths: map[transformertypes.PathType][]string{
						artifacts.ServiceDirPathType: {dir},
					},
				}},
			}
			return services, nil
		}
	}
	return nil, nil
}

// Transform transforms the artifacts
func (t *RustDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ServiceDirPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ServiceDirPathType][0], err)
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("Unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("Unable to load config for Transformer into %T : %s", sImageName, err)
		}
		ir := irtypes.IR{}
		irPresent := true
		err = a.GetConfig(irtypes.IRConfigType, &ir)
		if err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T : %s", ir, err)
		}
		ports := ir.GetAllServicePorts()
		var rustConfig RustTemplateConfig
		rustConfig.AppName = a.Name
		rocketTomlFilePath := filepath.Join(a.Paths[artifacts.ServiceDirPathType][0], rocketTomlFile)
		if _, err := os.Stat(rocketTomlFilePath); err == nil {
			rustConfig.RocketToml = rocketTomlFile
			var rocketTomlConfig RocketTomlConfig
			if _, err := toml.DecodeFile(rocketTomlFilePath, &rocketTomlConfig); err == nil {
				ports = append(ports, rocketTomlConfig.Global.Port)
				rustConfig.RocketAddress = rocketTomlConfig.Global.Address
			}
		}
		if len(ports) == 0 {
			ports = append(ports, common.DefaultServicePort)
		}
		rustConfig.Port = commonqa.GetPortForService(ports, `"`+a.Name+`"`)
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
			TemplateConfig: rustConfig,
		})
		paths := a.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, common.DefaultDockerfileName)}
		p := transformertypes.Artifact{
			Name:  sImageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: sImageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:  sConfig.ServiceName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: a.Paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: sImageName,
				artifacts.ServiceConfigType:   sConfig,
			},
		}
		if irPresent {
			dfs.Configs[irtypes.IRConfigType] = ir
		}
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}
