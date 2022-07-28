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
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/types"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

const (
	packageJSONFile        = "package.json"
	versionMappingFilePath = "mappings/nodeversions.yaml"
	// NodeVersionsMappingKind defines kind of NodeVersionMappingKind
	NodeVersionsMappingKind types.Kind = "NodeVersionsMapping"
)

// NodeVersionsMapping stores the Node versions mapping
type NodeVersionsMapping struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             NodeVersionsMappingSpec `yaml:"spec,omitempty"`
}

// NodeVersionsMappingSpec stores the Node version spec
type NodeVersionsMappingSpec struct {
	NodeVersions []string `yaml:"nodeVersions"`
}

// NodejsDockerfileGenerator implements the Transformer interface
type NodejsDockerfileGenerator struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	NodejsConfig *NodejsDockerfileYamlConfig
	NodeVersions []string
}

// NodejsTemplateConfig implements Nodejs config interface
type NodejsTemplateConfig struct {
	Port        int32
	Build       bool
	NodeVersion string
}

// NodejsDockerfileYamlConfig represents the configuration of the Nodejs dockerfile
type NodejsDockerfileYamlConfig struct {
	DefaultNodejsVersion string `yaml:"defaultNodejsVersion"`
}

// PackageJSON represents NodeJS package.json
type PackageJSON struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Keywords     []string          `json:"keywords"`
	Homepage     string            `json:"homepage"`
	License      string            `json:"license"`
	Files        []string          `json:"files"`
	Main         string            `json:"main"`
	Scripts      map[string]string `json:"scripts"`
	Os           []string          `json:"os"`
	CPU          []string          `json:"cpu"`
	Private      bool              `json:"private"`
	Engines      map[string]string `json:"engines"`
	Dependencies map[string]string `json:"dependencies"`
}

// Init Initializes the transformer
func (t *NodejsDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.NodejsConfig = &NodejsDockerfileYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.NodejsConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.NodejsConfig, err)
		return err
	}
	mappingFile := NodeVersionsMapping{}
	mappingFilePath := filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath)
	if err := common.ReadMove2KubeYaml(mappingFilePath, &mappingFile); err != nil {
		return fmt.Errorf("failed to load the Node versions mapping file at path %s . Error: %q", mappingFilePath, err)
	}
	if len(mappingFile.Spec.NodeVersions) == 0 {
		return fmt.Errorf("the mapping file at path %s is invalid", mappingFilePath)
	}
	t.NodeVersions = mappingFile.Spec.NodeVersions
	if len(t.NodeVersions) == 0 {
		return fmt.Errorf("atleast one node version should be specified in the nodeversions mappings file- %s", mappingFilePath)
	}
	if t.NodejsConfig.DefaultNodejsVersion == "" {
		t.NodejsConfig.DefaultNodejsVersion = t.NodeVersions[0]
	}
	logrus.Debugf("Extracted node versions from nodeversion mappings file - %+v", t.NodeVersions)
	return nil
}

// GetConfig returns the transformer config
func (t *NodejsDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *NodejsDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	var packageJSON PackageJSON
	if err := common.ReadJSON(filepath.Join(dir, packageJSONFile), &packageJSON); err != nil {
		logrus.Debugf("unable to read the package.json file: %s", err)
		return nil, nil
	}
	if packageJSON.Name == "" {
		err = fmt.Errorf("unable to get name of nodejs service at %s. Ignoring", dir)
		return nil, err
	}
	services = map[string][]transformertypes.Artifact{
		packageJSON.Name: {{
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceDirPathType: {dir},
			},
		}},
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *NodejsDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
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
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		ir := irtypes.IR{}
		irPresent := true
		err = a.GetConfig(irtypes.IRConfigType, &ir)
		if err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T : %s", ir, err)
		}
		build := false
		var nodeVersion string
		var packageJSON PackageJSON
		if err := common.ReadJSON(filepath.Join(a.Paths[artifacts.ServiceDirPathType][0], packageJSONFile), &packageJSON); err != nil {
			logrus.Errorf("unable to read the package.json file: %s", err)
			continue
		}
		if _, ok := packageJSON.Scripts["build"]; ok {
			build = true
		}
		if nodeVersionConstraint, ok := packageJSON.Engines["node"]; ok {
			nodeVersion = getNodeVersion(nodeVersionConstraint, t.NodejsConfig.DefaultNodejsVersion, t.NodeVersions)
			logrus.Debugf("Selected nodeVersion is - %s", nodeVersion)
		}
		if nodeVersion == "" {
			logrus.Infof("No Node version specified in the package.json file. Selecting the default Node version- %s", t.NodejsConfig.DefaultNodejsVersion)
			nodeVersion = t.NodejsConfig.DefaultNodejsVersion
		}
		ports := ir.GetAllServicePorts()
		if len(ports) == 0 {
			envMap, err := godotenv.Read(filepath.Join(a.Paths[artifacts.ServiceDirPathType][0], ".env"))
			if err == nil {
				portString, ok := envMap["PORT"]
				if ok {
					port, err := cast.ToInt32E(portString)
					if err == nil {
						ports = []int32{port}
					}
				}
			} else {
				logrus.Debugf("Could not parse the .env file, %s", err)
			}
		}
		port := commonqa.GetPortForService(ports, `"`+a.Name+`"`)
		var nodejsConfig NodejsTemplateConfig
		nodejsConfig.Build = build
		nodejsConfig.Port = port
		nodejsConfig.NodeVersion = nodeVersion
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
			TemplateConfig: nodejsConfig,
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
