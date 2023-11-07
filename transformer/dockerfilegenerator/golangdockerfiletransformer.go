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
	"sort"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	"github.com/konveyor/move2kube-wasm/types"
	irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	"github.com/konveyor/move2kube-wasm/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/konveyor/move2kube-wasm/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

const (
	golangVersionMappingFilePath = "mappings/golangversions.yaml"
	// GolangModFilePathType points to the go.mod file path
	GolangModFilePathType transformertypes.PathType = "GoModFilePath"
	// GolangVersionsMappingKind defines kind of GolangVersionMappingKind
	GolangVersionsMappingKind types.Kind = "GolangVersionsMapping"
)

// GolangVersionsMapping stores the Go versions mapping
type GolangVersionsMapping struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             GolangVersionsMappingSpec `yaml:"spec,omitempty"`
}

// GolangVersionsMappingSpec stores the Go version spec
type GolangVersionsMappingSpec struct {
	DisableSort    bool                `yaml:"disableSort"`
	GolangVersions []map[string]string `yaml:"golangVersions"`
}

// GolangDockerfileGenerator implements the Transformer interface
type GolangDockerfileGenerator struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	GolangConfig *GolangDockerfileYamlConfig
	Spec         GolangVersionsMappingSpec
}

// GolangTemplateConfig implements Golang config interface
type GolangTemplateConfig struct {
	Ports          []int32
	AppName        string
	GolangImageTag string
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
	// load the version mapping file
	mappingFilePath := filepath.Join(t.Env.GetEnvironmentContext(), golangVersionMappingFilePath)
	spec, err := LoadGolangVersionMappingsFile(mappingFilePath)
	if err != nil {
		return fmt.Errorf("failed to load the Golang version mappings file at path %s . Error: %q", golangVersionMappingFilePath, err)
	}
	t.Spec = spec
	if t.GolangConfig.DefaultGoVersion == "" {
		if len(t.Spec.GolangVersions) != 0 {
			if _, ok := t.Spec.GolangVersions[0][versionKey]; ok {
				t.GolangConfig.DefaultGoVersion = t.Spec.GolangVersions[0][versionKey]
			}
		}
	}
	logrus.Debugf("Extracted Golang versions from Go version mappings file - %+v", t.Spec)
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
		if len(a.Paths[artifacts.ServiceDirPathType]) == 0 {
			continue
		}
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
		golangVersionToImageTagMapping := make(map[string]string)
		for _, golangVersion := range t.Spec.GolangVersions {
			if imageTag, ok := golangVersion[imageTagKey]; ok {
				golangVersionToImageTagMapping[golangVersion[versionKey]] = imageTag
			}
		}
		logrus.Debugf("Extracted Golang version-image tag mappings are %+v", golangVersionToImageTagMapping)
		var golangImageTag string
		imageTag, ok := golangVersionToImageTagMapping[modFile.Go.Version]
		if ok {
			golangImageTag = imageTag
			logrus.Debugf("Selected Golang image tag is - %s", golangImageTag)
		} else {
			golangImageTag = golangVersionToImageTagMapping[t.GolangConfig.DefaultGoVersion]
			logrus.Warnf("Could not find a matching Golang version in the mapping. Selecting image tag %s corresponding to the default Golang version %s", golangImageTag, t.GolangConfig.DefaultGoVersion)
		}
		detectedPorts := ir.GetAllServicePorts()
		if len(detectedPorts) == 0 {
			detectedPorts = append(detectedPorts, common.DefaultServicePort)
		}
		detectedPorts = commonqa.GetPortsForService(detectedPorts, `"`+a.Name+`"`)
		golangConfig := GolangTemplateConfig{
			AppName:        a.Name,
			Ports:          detectedPorts,
			GolangImageTag: golangImageTag,
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

// LoadGolangVersionMappingsFile loads the Golang version mappings file
func LoadGolangVersionMappingsFile(mappingFilePath string) (GolangVersionsMappingSpec, error) {
	mappingFile := GolangVersionsMapping{}
	if err := common.ReadMove2KubeYaml(mappingFilePath, &mappingFile); err != nil {
		return mappingFile.Spec, fmt.Errorf("failed to load the Golang versions mapping file at path %s . Error: %q", mappingFilePath, err)
	}
	// validate the file
	if len(mappingFile.Spec.GolangVersions) == 0 {
		return mappingFile.Spec, fmt.Errorf("the Golang version mappings file at path %s is invalid. Atleast one Go version should be specified", mappingFilePath)
	}
	for i, v := range mappingFile.Spec.GolangVersions {
		if _, ok := v[versionKey]; !ok {
			return mappingFile.Spec, fmt.Errorf("the Golang version is missing from the object %#v at the %dth index in the array", mappingFile.Spec.GolangVersions[i], i)
		}
	}
	// sort the list using semantic version comparison
	if !mappingFile.Spec.DisableSort {
		sort.SliceStable(mappingFile.Spec.GolangVersions, func(i, j int) bool {
			return semver.Compare(mappingFile.Spec.GolangVersions[i][versionKey], mappingFile.Spec.GolangVersions[j][versionKey]) == 1
		})
	}
	return mappingFile.Spec, nil
}
