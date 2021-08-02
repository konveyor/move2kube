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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	gopache "github.com/Akash-Nayak/GopacheConfig"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	phpExt      = ".php"
	virtualHost = "VirtualHost"
	confExt     = ".conf"
)

// PHPDockerfileGenerator implements the Transformer interface
type PHPDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// PhpTemplateConfig implements Php config interface
type PhpTemplateConfig struct {
	Ports    []int32
	ConfFile string
}

// Init Initializes the transformer
func (t *PHPDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *PHPDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *PHPDockerfileGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// ParseConfFile parses the conf file to detect the port
func ParseConfFile(confFilePath string) int32 {
	var port int32
	confFile, err := os.Open(confFilePath)
	if err != nil {
		logrus.Errorf("Could not open the conf file: %s", err)
		return port
	}
	root, err := gopache.Parse(confFile)
	if err != nil {
		logrus.Errorf("Error while parsing configuration file : %s", err)
		return port
	}
	match, err := root.FindOne(virtualHost)
	if err != nil {
		logrus.Debugf("Could not find the VirtualHost in conf file: %s", err)
		return port
	}
	tokens := strings.Split(match.Content, ":")
	if len(tokens) > 1 {
		detectedPort, err := strconv.ParseInt(tokens[1], 10, 32)
		if err != nil {
			logrus.Errorf("Error while converting the port from string to int : %s", err)
			return port
		}
		return int32(detectedPort)
	}
	defer confFile.Close()
	return port
}

// DetectConfFiles detects if conf files are present or not
func DetectConfFiles(dir string) ([]string, error) {
	var confFilesPaths []string
	confFiles, err := common.GetFilesByExt(dir, []string{confExt})
	if err != nil {
		logrus.Debugf("Could not find conf files %s", err)
		return confFilesPaths, err
	}
	for _, confFilePath := range confFiles {
		confFileRelPath, err := filepath.Rel(dir, confFilePath)
		if err != nil {
			logrus.Errorf("Unable to resolve conf file path %s as rel path : %s", confFilePath, err)
			continue
		}
		confFilesPaths = append(confFilesPaths, confFileRelPath)
	}
	return confFilesPaths, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *PHPDockerfileGenerator) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Error while trying to read directory : %s", err)
		return nil, nil, err
	}
	for _, de := range dirEntries {
		if filepath.Ext(de.Name()) != phpExt {
			continue
		}
		unnamedServices = []transformertypes.TransformerPlan{{
			Mode:              t.Config.Spec.Mode,
			ArtifactTypes:     []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {dir},
			},
		}}
		return nil, unnamedServices, nil
	}
	return nil, nil, nil
}

// Transform transforms the artifacts
func (t *PHPDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
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
		var detectedPorts []int32
		var phpConfig PhpTemplateConfig
		var confFilePath string
		confFiles, err := DetectConfFiles(a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Debugf("Could not detect any conf files %s", err)
		} else {
			if len(confFiles) == 1 {
				confFilePath = confFiles[0]
			} else if len(confFiles) > 1 {
				confFilePath = commonqa.GetConfFileForService(confFiles, a.Name)
			}
			if confFilePath != "" {
				port := ParseConfFile(filepath.Join(a.Paths[artifacts.ProjectPathPathType][0], confFilePath))
				if port != 0 {
					detectedPorts = append(detectedPorts, port)
				}
				phpConfig.ConfFile = confFilePath
			}
		}
		detectedPorts = commonqa.GetPortsForService(detectedPorts, a.Name)
		phpConfig.Ports = detectedPorts
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
			TemplateConfig: phpConfig,
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
