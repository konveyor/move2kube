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
	irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	"github.com/konveyor/move2kube-wasm/types/qaengine/commonqa"
	"github.com/spf13/cast"
	"os"

	//"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	"github.com/konveyor/move2kube-wasm/types"
	//irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	//"github.com/konveyor/move2kube-wasm/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/konveyor/move2kube-wasm/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	//"github.com/spf13/cast"
	"golang.org/x/mod/semver"
)

// -----------------------------------------------------------------------------------
// Struct to load the package.json file
// -----------------------------------------------------------------------------------

// PackageJSON represents NodeJS package.json
type PackageJSON struct {
	Private        bool              `json:"private"`
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	Description    string            `json:"description"`
	Homepage       string            `json:"homepage"`
	License        string            `json:"license"`
	Main           string            `json:"main"`
	PackageManager string            `json:"packageManager,omitempty"`
	Keywords       []string          `json:"keywords,omitempty"`
	Files          []string          `json:"files,omitempty"`
	Os             []string          `json:"os,omitempty"`
	CPU            []string          `json:"cpu,omitempty"`
	Scripts        map[string]string `json:"scripts,omitempty"`
	Engines        map[string]string `json:"engines,omitempty"`
	Dependencies   map[string]string `json:"dependencies,omitempty"`
}

// NodejsTemplateConfig implements Nodejs config interface
type NodejsTemplateConfig struct {
	Port                  int32
	Build                 bool
	NodeVersion           string
	NodeImageTag          string
	NodeMajorVersion      string
	NodeVersionProperties map[string]string
	PackageManager        string
}

// -----------------------------------------------------------------------------------
// Mappings file
// -----------------------------------------------------------------------------------

// NodeVersionsMapping stores the Node versions mapping
type NodeVersionsMapping struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             NodeVersionsMappingSpec `yaml:"spec,omitempty"`
}

// NodeVersionsMappingSpec stores the Node version spec
type NodeVersionsMappingSpec struct {
	DisableSort  bool                `yaml:"disableSort"`
	NodeVersions []map[string]string `yaml:"nodeVersions"`
}

// -----------------------------------------------------------------------------------
// Transformer
// -----------------------------------------------------------------------------------

// NodejsDockerfileGenerator implements the Transformer interface
type NodejsDockerfileGenerator struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	NodejsConfig *NodejsDockerfileYamlConfig
	Spec         NodeVersionsMappingSpec
}

// NodejsDockerfileYamlConfig represents the configuration of the Nodejs dockerfile
type NodejsDockerfileYamlConfig struct {
	DefaultNodejsVersion  string `yaml:"defaultNodejsVersion"`
	DefaultPackageManager string `yaml:"defaultPackageManager"`
}

const (
	packageJSONFile        = "package.json"
	versionMappingFilePath = "mappings/nodeversions.yaml"
	defaultPackageManager  = "npm"
	imageTagKey            = "imageTag"
	versionKey             = "version"
	// NodeVersionsMappingKind defines kind of NodeVersionMappingKind
	NodeVersionsMappingKind types.Kind = "NodeVersionsMapping"
)

// Init Initializes the transformer
func (t *NodejsDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.Env = env

	// load the config
	t.NodejsConfig = &NodejsDockerfileYamlConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.NodejsConfig); err != nil {
		return fmt.Errorf("unable to load config for Transformer %+v into %T . Error: %q", t.Config.Spec.Config, t.NodejsConfig, err)
	}
	if t.NodejsConfig.DefaultPackageManager == "" {
		t.NodejsConfig.DefaultPackageManager = defaultPackageManager
	}

	// load the version mapping file
	mappingFilePath := filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath)
	spec, err := LoadNodeVersionMappingsFile(mappingFilePath)
	if err != nil {
		return fmt.Errorf("failed to load the node version mappings file at path %s . Error: %q", versionMappingFilePath, err)
	}
	t.Spec = spec
	if t.NodejsConfig.DefaultNodejsVersion == "" {
		if len(t.Spec.NodeVersions) != 0 {
			if _, ok := t.Spec.NodeVersions[0][versionKey]; ok {
				t.NodejsConfig.DefaultNodejsVersion = t.Spec.NodeVersions[0][versionKey]
			}
		}
	}
	logrus.Debugf("Extracted node versions from nodeversion mappings file - %+v", t.Spec)
	return nil
}

// GetConfig returns the transformer config
func (t *NodejsDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *NodejsDockerfileGenerator) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	packageJsonPath := filepath.Join(dir, packageJSONFile)
	packageJson := PackageJSON{}
	if err := common.ReadJSON(packageJsonPath, &packageJson); err != nil {
		logrus.Debugf("failed to read the package.json file at the path %s . Error: %q", packageJsonPath, err)
		return nil, nil
	}
	if packageJson.Name == "" {
		return nil, fmt.Errorf("unable to get name of nodejs service at %s. Ignoring", dir)
	}
	serviceName := packageJson.Name
	normalizedServiceName := common.MakeStringK8sServiceNameCompliant(serviceName)
	services := map[string][]transformertypes.Artifact{
		normalizedServiceName: {{
			Paths: map[transformertypes.PathType][]string{artifacts.ServiceDirPathType: {dir}},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.OriginalNameConfigType: artifacts.OriginalNameConfig{OriginalName: serviceName},
			},
		}},
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *NodejsDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 {
			continue
		}
		serviceDir := newArtifact.Paths[artifacts.ServiceDirPathType][0]
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), serviceDir)
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative. Error: %q", serviceDir, err)
		}
		serviceConfig := artifacts.ServiceConfig{}
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &serviceConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T . Error: %q", serviceConfig, err)
			continue
		}
		imageName := artifacts.ImageName{}
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Debugf("unable to load config for Transformer into %T . Error: %q", imageName, err)
		}
		if imageName.ImageName == "" {
			imageName.ImageName = common.MakeStringContainerImageNameCompliant(serviceConfig.ServiceName)
		}
		ir := irtypes.IR{}
		irPresent := true
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T . Error: %q", ir, err)
		}
		build := false
		packageJSON := PackageJSON{}
		packageJsonPath := filepath.Join(serviceDir, packageJSONFile)
		if err := common.ReadJSON(packageJsonPath, &packageJSON); err != nil {
			logrus.Errorf("failed to parse the package.json file at path %s . Error: %q", packageJsonPath, err)
			continue
		}
		if _, ok := packageJSON.Scripts["build"]; ok {
			build = true
		}
		nodeVersion := t.NodejsConfig.DefaultNodejsVersion
		if nodeVersionConstraint, ok := packageJSON.Engines["node"]; ok {
			nodeVersion = getNodeVersion(
				nodeVersionConstraint,
				t.NodejsConfig.DefaultNodejsVersion,
				common.Map(t.Spec.NodeVersions, func(x map[string]string) string { return x[versionKey] }),
			)
			logrus.Debugf("Selected nodeVersion is - %s", nodeVersion)
		}
		packageManager := t.NodejsConfig.DefaultPackageManager
		if packageJSON.PackageManager != "" {
			parts := strings.Split(packageJSON.PackageManager, "@")
			if len(parts) > 0 {
				packageManager = parts[0]
			}
		}
		ports := ir.GetAllServicePorts()
		if len(ports) == 0 {
			envPath := filepath.Join(serviceDir, ".env")
			envMap, err := godotenv.Read(envPath)
			if err != nil {
				if !os.IsNotExist(err) {
					logrus.Warnf("failed to parse the .env file at the path %s . Error: %q", envPath, err)
				}
			} else if portString, ok := envMap["PORT"]; ok {
				port, err := cast.ToInt32E(portString)
				if err != nil {
					logrus.Errorf("failed to parse the port string '%s' as an integer. Error: %q", portString, err)
				} else {
					ports = []int32{port}
				}
			}
		}
		port := commonqa.GetPortForService(ports, `"`+newArtifact.Name+`"`)
		var props map[string]string
		if idx := common.FindIndex(t.Spec.NodeVersions, func(x map[string]string) bool { return x[versionKey] == nodeVersion }); idx != -1 {
			props = t.Spec.NodeVersions[idx]
		}
		nodejsConfig := NodejsTemplateConfig{
			Build:       build,
			Port:        port,
			NodeVersion: nodeVersion,
			// NodeImageTag:          getNodeImageTag(t.Spec.NodeVersions, nodeVersion), // To use this, change the base image in the Dockerfile template to- FROM node:{{ .NodeImageTag }}
			NodeMajorVersion:      strings.TrimPrefix(semver.Major(nodeVersion), "v"),
			NodeVersionProperties: props,
			PackageManager:        packageManager,
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
		paths := newArtifact.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, common.DefaultDockerfileName)}
		p := transformertypes.Artifact{
			Name:  imageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: imageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:  serviceConfig.ServiceName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: newArtifact.Paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: imageName,
				artifacts.ServiceConfigType:   serviceConfig,
			},
		}
		if irPresent {
			dfs.Configs[irtypes.IRConfigType] = ir
		}
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}

// LoadNodeVersionMappingsFile loads the node version mappings file
func LoadNodeVersionMappingsFile(mappingFilePath string) (NodeVersionsMappingSpec, error) {
	mappingFile := NodeVersionsMapping{}
	if err := common.ReadMove2KubeYaml(mappingFilePath, &mappingFile); err != nil {
		return mappingFile.Spec, fmt.Errorf("failed to load the Node versions mapping file at path %s . Error: %q", mappingFilePath, err)
	}
	// validate the file
	if len(mappingFile.Spec.NodeVersions) == 0 {
		return mappingFile.Spec, fmt.Errorf("the node version mappings file at path %s is invalid. Atleast one node version should be specified", mappingFilePath)
	}
	for i, v := range mappingFile.Spec.NodeVersions {
		if _, ok := v[versionKey]; !ok {
			return mappingFile.Spec, fmt.Errorf("the version is missing from the object %#v at the %dth index in the array", mappingFile.Spec.NodeVersions[i], i)
		}
	}
	// sort the list using semantic version comparison
	if !mappingFile.Spec.DisableSort {
		sort.SliceStable(mappingFile.Spec.NodeVersions, func(i, j int) bool {
			return semver.Compare(mappingFile.Spec.NodeVersions[i][versionKey], mappingFile.Spec.NodeVersions[j][versionKey]) == 1
		})
	}
	return mappingFile.Spec, nil
}
