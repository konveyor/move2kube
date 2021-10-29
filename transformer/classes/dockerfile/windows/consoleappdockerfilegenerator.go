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
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/konveyor/move2kube/types/source/dotnet"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	// AppConfigFilePathListType points to the go.mod file path
	AppConfigFilePathListType transformertypes.PathType = "AppConfigFilePathList"

	// AppCfgFile is file name of App.Config file in dotnet projects
	AppCfgFile = "App.config"
)

// ConsoleTemplateConfig implements .Net Console config interface
type ConsoleTemplateConfig struct {
	Ports            []int32
	AppName          string
	BaseImageVersion string
}

// WinConsoleAppDockerfileGenerator implements the Transformer interface
type WinConsoleAppDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *WinConsoleAppDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *WinConsoleAppDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *WinConsoleAppDockerfileGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// parseAppConfig parses the application config
func (t *WinConsoleAppDockerfileGenerator) parseAppConfigForPort(AppCfgFilePath string) ([]int32, error) {
	appConfigFile, err := os.Open(AppCfgFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not open the App.config file: %s", err)
	}

	defer appConfigFile.Close()

	byteValue, _ := ioutil.ReadAll(appConfigFile)
	appCfg := dotnet.AppConfig{}
	xml.Unmarshal(byteValue, &appCfg)
	if err != nil {
		return nil, fmt.Errorf("could not parse the App.config file: %s", err)
	}

	ports := make([]int32, 0)
	for _, addKey := range appCfg.AppCfgSettings.AddList {
		parsedURL, err := url.ParseRequestURI(addKey.Value)
		if err != nil {
			logrus.Errorf("Could not parse URI: %s", err)
			continue
		}

		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			logrus.Warnf("Scheme or host is empty in URI")
			continue
		}

		_, port, err := net.SplitHostPort(parsedURL.Host)
		if err != nil {
			logrus.Errorf("Could not extract port from URI: %s", err)
			continue
		}

		portAsInt, err := strconv.ParseInt(port, 10, 32)
		if err != nil {
			logrus.Errorf("Could not process port from URI: %s", err)
			continue
		}

		ports = append(ports, int32(portAsInt))
	}

	if len(appCfg.Model.Services.ServiceList) == 0 {
		return ports, nil
	}

	for _, svc := range appCfg.Model.Services.ServiceList {
		for _, addKey := range svc.Host.BaseAddresses.AddList {
			parsedURL, err := url.ParseRequestURI(addKey.BaseAddress)
			if err != nil {
				logrus.Errorf("Could not parse URI: %s", err)
				continue
			}

			if parsedURL.Scheme == "" || parsedURL.Host == "" {
				logrus.Warnf("Scheme or host is empty in URI")
				continue
			}

			_, port, err := net.SplitHostPort(parsedURL.Host)
			if err != nil {
				logrus.Errorf("Could not extract port from URI: %s", err)
				continue
			}

			portAsInt, err := strconv.ParseInt(port, 10, 32)
			if err != nil {
				logrus.Errorf("Could not process port from URI: %s", err)
				continue
			}

			ports = append(ports, int32(portAsInt))
		}
	}

	return ports, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *WinConsoleAppDockerfileGenerator) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Error while trying to read directory: %s", err)
		return nil, nil, err
	}
	appName := ""
	appConfigList := make([]string, 0)
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
				logrus.Debugf("Not a supported dotnet framework [%s]", configuration.PropertyGroup.TargetFrameworkVersion)
				continue
			}

			isWebProj, err := isWeb(configuration)
			if err != nil {
				logrus.Errorf("%s", err)
				continue
			}
			if isWebProj {
				continue
			}

			appCfgFilePath := filepath.Join(dir, filepath.Dir(csPath), AppCfgFile)
			if _, err = os.Stat(appCfgFilePath); os.IsNotExist(err) {
				continue
			}
			appConfigList = append(appConfigList, appCfgFilePath)

			appName = strings.TrimSuffix(filepath.Base(de.Name()), filepath.Ext(de.Name()))
		}

		// Exit soon of after the solution file is found
		break
	}

	if appName == "" {
		return nil, nil, nil
	}

	namedServices = map[string]transformertypes.ServicePlan{
		appName: []transformertypes.TransformerPlan{{
			Mode:              t.Config.Spec.Mode,
			ArtifactTypes:     []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {dir},
				AppConfigFilePathListType:     appConfigList,
			},
		}},
	}
	return namedServices, nil, nil
}

// Transform transforms the artifacts
func (t *WinConsoleAppDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
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
		for _, appConfigFilePath := range a.Paths[AppConfigFilePathListType] {
			portList, err := t.parseAppConfigForPort(appConfigFilePath)
			if err != nil {
				logrus.Errorf("%s", err)
				continue
			}
			if portList != nil {
				detectedPorts = append(detectedPorts, portList...)
			}
		}

		detectedPorts = commonqa.GetPortsForService(detectedPorts, a.Name)
		var consoleConfig ConsoleTemplateConfig
		consoleConfig.AppName = a.Name
		consoleConfig.Ports = detectedPorts
		consoleConfig.BaseImageVersion = dotnet.DefaultBaseImageVersion

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
			TemplateConfig: consoleConfig,
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
