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
	"strconv"
	"strings"

	gopache "github.com/Akash-Nayak/GopacheConfig"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
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
	ConfFile     string
	ConfFilePort int32
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

// parseConfFile parses the conf file to detect the port
func parseConfFile(confFilePath string) (int32, error) {
	var port int32
	confFile, err := os.Open(confFilePath)
	if err != nil {
		logrus.Errorf("Could not open the apache config file: %s", err)
		return port, err
	}
	defer confFile.Close()
	root, err := gopache.Parse(confFile)
	if err != nil {
		logrus.Errorf("Error while parsing apache config file : %s", err)
		return port, err
	}
	match, err := root.FindOne(virtualHost)
	if err != nil {
		logrus.Debugf("Could not find the VirtualHost in apache config file: %s", err)
		return port, err
	}
	tokens := strings.Split(match.Content, ":")
	if len(tokens) > 1 {
		detectedPort, err := strconv.ParseInt(tokens[1], 10, 32)
		if err != nil {
			logrus.Errorf("Error while converting the port from string to int : %s", err)
			return port, err
		}
		return int32(detectedPort), nil
	}
	return port, err
}

// detectConfFiles detects if conf files are present or not
func detectConfFiles(dir string) ([]string, error) {
	var confFilesPaths []string
	confFiles, err := common.GetFilesByExt(dir, []string{confExt})
	if err != nil {
		logrus.Debugf("Could not find conf files %s", err)
		return confFilesPaths, err
	}
	for _, confFilePath := range confFiles {
		confFile, err := os.Open(confFilePath)
		if err != nil {
			logrus.Debugf("Could not open the conf file: %s", err)
			confFile.Close()
			continue
		}
		defer confFile.Close()
		_, err = gopache.Parse(confFile)
		if err != nil {
			logrus.Debugf("Error while parsing conf file : %s", err)
			continue
		}
		confFileRelPath, err := filepath.Rel(dir, confFilePath)
		if err != nil {
			logrus.Errorf("Unable to resolve apache config file path %s as rel path : %s", confFilePath, err)
			continue
		}
		confFilesPaths = append(confFilesPaths, confFileRelPath)
	}
	return confFilesPaths, nil
}

// GetConfFileForService returns ports used by a service
func GetConfFileForService(confFiles []string, serviceName string) string {
	noAnswer := "none of the above"
	confFiles = append(confFiles, noAnswer)
	quesKey := common.JoinQASubKeys(common.ConfigServicesKey, `"`+serviceName+`"`, common.ConfigApacheConfFileForServiceKeySegment)
	desc := fmt.Sprintf("Choose the apache config file to be used for the service %s", serviceName)
	hints := []string{fmt.Sprintf("Selected apache config file will be used for identifying the port to be exposed for the service %s", serviceName)}
	selectedConfFile := qaengine.FetchSelectAnswer(quesKey, desc, hints, confFiles[0], confFiles, nil)
	if selectedConfFile == noAnswer {
		logrus.Debugf("No apache config file selected for the service %s", serviceName)
		return ""
	}
	return selectedConfFile
}

// DirectoryDetect runs detect in each sub directory
func (t *PHPDockerfileGenerator) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	phpFiles, err := common.GetFilesByExtInCurrDir(dir, []string{phpExt})
	if err != nil {
		return nil, fmt.Errorf("failed to look for .php files in the directory %s . Error: %q", dir, err)
	}
	if len(phpFiles) == 0 {
		return nil, nil
	}
	serviceName := filepath.Base(dir)
	normalizedServiceName := common.MakeStringK8sServiceNameCompliant(serviceName)
	services := map[string][]transformertypes.Artifact{
		normalizedServiceName: {{
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceDirPathType: {dir},
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.OriginalNameConfigType: artifacts.OriginalNameConfig{OriginalName: serviceName},
			},
		}},
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *PHPDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if len(a.Paths[artifacts.ServiceDirPathType]) == 0 {
			continue
		}
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
		detectedPorts := ir.GetAllServicePorts()
		var phpConfig PhpTemplateConfig
		confFiles, err := detectConfFiles(a.Paths[artifacts.ServiceDirPathType][0])
		if err != nil {
			logrus.Debugf("Could not detect any conf files %s", err)
		} else {
			if len(confFiles) == 1 {
				phpConfig.ConfFile = confFiles[0]
			} else if len(confFiles) > 1 {
				phpConfig.ConfFile = GetConfFileForService(confFiles, a.Name)
			}
			if phpConfig.ConfFile != "" {
				phpConfig.ConfFilePort, err = parseConfFile(filepath.Join(a.Paths[artifacts.ServiceDirPathType][0], phpConfig.ConfFile))
				if err != nil {
					logrus.Errorf("Error while parsing configuration file : %s", err)
				}
			}
			if phpConfig.ConfFilePort == 0 {
				phpConfig.ConfFilePort = commonqa.GetPortForService(detectedPorts, `"`+a.Name+`"`)
			}
		}
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
