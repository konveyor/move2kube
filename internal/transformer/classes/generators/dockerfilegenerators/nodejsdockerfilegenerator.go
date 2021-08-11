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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudrecipes/packagejson"
	"github.com/joho/godotenv"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"golang.org/x/mod/semver"
)

const (
	nodeVersion = "12"
	packageJson = "package.json"
)

// NodejsDockerfileGenerator implements the Transformer interface
type NodejsDockerfileGenerator struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	NodejsConfig NodejsDockerfileYamlConfig
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
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Keywords    []string          `json:"keywords"`
	Homepage    string            `json:"homepage"`
	License     string            `json:"license"`
	Files       []string          `json:"files"`
	Main        string            `json:"main"`
	Scripts     map[string]string `json:"scripts"`
	Os          []string          `json:"os"`
	Cpu         []string          `json:"cpu"`
	Private     bool              `json:"private"`
	Engines     map[string]string `json:"engines"`
}

// parseJson parses package.json payload and returns structure.
func parseJson(payload []byte) (*PackageJSON, error) {
	var packagejson *PackageJSON
	err := json.Unmarshal(payload, &packagejson)
	return packagejson, err
}

// Init Initializes the transformer
func (t *NodejsDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.NodejsConfig = NodejsDockerfileYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, &t.NodejsConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.NodejsConfig, err)
		return err
	}
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
	parsedPackageJson, err := packagejson.Parse(packagejsondata)
	if err != nil {
		logrus.Debugf("Found package.json, but unable to parse it to get project name. Ignoring : %s", err)
		return nil, nil, nil
	}
	if parsedPackageJson.Name == "" {
		err = fmt.Errorf("unable to get project name of nodejs project at %s. Ignoring", dir)
		return nil, nil, err
	}
	namedServices = map[string]transformertypes.ServicePlan{
		parsedPackageJson.Name: []transformertypes.TransformerPlan{{
			Mode:              t.Config.Spec.Mode,
			ArtifactTypes:     []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {dir},
			},
		}},
	}
	return namedServices, nil, nil
}

// Transform transforms the artifacts
func (t *NodejsDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}
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
		build := false
		nodeVersion := t.NodejsConfig.DefaultNodejsVersion
		packagejsondata, err := os.ReadFile(filepath.Join(a.Paths[artifacts.ProjectPathPathType][0], packageJson))
		if err != nil {
			logrus.Debugf("unable to read the package.json file: %s", err)
		} else {
			parsedPackageJson, err := parseJson(packagejsondata)
			if err != nil {
				logrus.Debugf("Found package.json, but unable to parse it. Ignoring : %s", err)
			}
			if _, ok := parsedPackageJson.Scripts["build"]; ok {
				build = true
			}
			if node, ok := parsedPackageJson.Engines["node"]; ok {
				if !strings.HasPrefix(node, "v") {
					node = "v" + node
				}
				nodeVersion = strings.TrimPrefix(semver.Major(node), "v")
				logrus.Infof("============= nodeVersion = %s ===========\n", nodeVersion)
			}
			// var jsonData map[string]interface{}
			// if err := json.Unmarshal(packagejsondata, &jsonData); err != nil {
			// 	logrus.Errorf("Error while unmarshalling package.json : %s", err)
			// }
			// enginesI, ok := jsonData["engines"]
			// if ok {
			// 	engines, ok := enginesI.(map[string]interface{})
			// 	if ok {
			// 		nodeVersionI, ok := engines["node"]
			// 		if ok {
			// 			node, ok := nodeVersionI.(string)
			// 			if ok {
			// 				if !strings.HasPrefix(node, "v") {
			// 					node = "v" + node
			// 				}
			// 				nodeVersion = strings.TrimPrefix(semver.Major(node), "v")
			// 			}
			// 		}
			// 	}
			// }
		}
		var ports []int32
		envMap, err := godotenv.Read(filepath.Join(a.Paths[artifacts.ProjectPathPathType][0], ".env"))
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
		port := commonqa.GetPortForService(ports, a.Name)
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
