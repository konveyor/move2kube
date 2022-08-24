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
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	dotnetutils "github.com/konveyor/move2kube/transformer/dockerfilegenerator/dotnet"
	irtypes "github.com/konveyor/move2kube/types/ir"
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

// parseAppConfig parses the application config
func (t *WinConsoleAppDockerfileGenerator) parseAppConfigForPort(AppCfgFilePath string) ([]int32, error) {
	appConfigFile, err := os.Open(AppCfgFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not open the App.config file: %s", err)
	}

	defer appConfigFile.Close()

	byteValue, _ := io.ReadAll(appConfigFile)
	appCfg := dotnet.AppConfig{}
	xml.Unmarshal(byteValue, &appCfg)
	if err != nil {
		return nil, fmt.Errorf("could not parse the App.config file: %s", err)
	}

	ports := []int32{}
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
func (t *WinConsoleAppDockerfileGenerator) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	slnPaths, err := common.GetFilesByExtInCurrDir(dir, []string{dotnet.VISUAL_STUDIO_SOLUTION_FILE_EXT})
	if err != nil {
		return nil, fmt.Errorf("failed to list the dot net visual studio solution files in the directory %s . Error: %q", dir, err)
	}
	if len(slnPaths) == 0 {
		return nil, nil
	}
	if len(slnPaths) > 1 {
		logrus.Debugf("more than one visual studio solution file detected. Number of .sln files %d", len(slnPaths))
	}
	slnPath := slnPaths[0]
	appName := dotnetutils.GetParentProjectName(slnPath)
	normalizedAppName := common.MakeStringK8sServiceNameCompliant(appName)

	relCSProjPaths, err := dotnetutils.GetCSProjPathsFromSlnFile(slnPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get the .csproj paths from .sln file at path %s . Error: %q", slnPath, err)
	}

	if len(relCSProjPaths) == 0 {
		return nil, fmt.Errorf("no c sharp projects available found in the .sln at path %s", slnPath)
	}

	appCfgFilePaths := []string{}

	found := false
	for _, relCSProjPath := range relCSProjPaths {
		csProjPath := filepath.Join(dir, strings.TrimSpace(relCSProjPath))
		csProjBytes, err := os.ReadFile(csProjPath)
		if err != nil {
			logrus.Errorf("failed to read the c sharp project file at path %s . Error: %q", csProjPath, err)
			continue
		}
		configuration := dotnet.CSProj{}
		if err := xml.Unmarshal(csProjBytes, &configuration); err != nil {
			logrus.Errorf("failed to parse the xml c sharp project file at path %s . Error: %q", csProjPath, err)
			continue
		}
		idx := common.FindIndex(
			configuration.PropertyGroups,
			func(x dotnet.PropertyGroup) bool { return x.TargetFrameworkVersion != "" },
		)
		if idx == -1 {
			logrus.Debugf("failed to find the target framework in any of the property groups inside the c sharp project file at path %s", csProjPath)
			continue
		}
		targetFrameworkVersion := configuration.PropertyGroups[idx].TargetFrameworkVersion
		if !dotnet.Version4.MatchString(targetFrameworkVersion) {
			logrus.Errorf("console dot net tranformer: the c sharp project file at path %s does not have a supported framework version. Actual version: %s", csProjPath, targetFrameworkVersion)
			continue
		}
		isWebProj, err := isWeb(configuration)
		if err != nil {
			logrus.Errorf("failed to detect if it's a web/asp net project. Error: %q", err)
			continue
		}
		if isWebProj {
			continue
		}
		found = true
		appCfgFilePath := filepath.Join(dir, filepath.Dir(relCSProjPath), AppCfgFile)
		if _, err := os.Stat(appCfgFilePath); err != nil {
			continue
		}
		appCfgFilePaths = append(appCfgFilePaths, appCfgFilePath)
	}

	if !found {
		return nil, nil
	}

	services := map[string][]transformertypes.Artifact{
		normalizedAppName: {{
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceDirPathType: {dir},
				AppConfigFilePathListType:    appCfgFilePaths,
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.OriginalNameConfigType: artifacts.OriginalNameConfig{OriginalName: appName},
			},
		}},
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *WinConsoleAppDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 {
			continue
		}
		serviceDir := newArtifact.Paths[artifacts.ServiceDirPathType][0]
		relServiceDir, err := filepath.Rel(t.Env.GetEnvironmentSource(), serviceDir)
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative. Error: %q", serviceDir, err)
			continue
		}
		serviceConfig := artifacts.ServiceConfig{}
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &serviceConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", serviceConfig, err)
			continue
		}
		imageName := artifacts.ImageName{}
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", imageName, err)
		}
		ir := irtypes.IR{}
		irPresent := true
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T : %s", ir, err)
		}
		detectedPorts := []int32{}
		for _, appConfigFilePath := range newArtifact.Paths[AppConfigFilePathListType] {
			portList, err := t.parseAppConfigForPort(appConfigFilePath)
			if err != nil {
				logrus.Errorf("%s", err)
				continue
			}
			if portList != nil {
				detectedPorts = append(detectedPorts, portList...)
			}
		}
		if len(detectedPorts) == 0 {
			detectedPorts = ir.GetAllServicePorts()
		}
		detectedPorts = commonqa.GetPortsForService(detectedPorts, `"`+newArtifact.Name+`"`)
		var consoleConfig ConsoleTemplateConfig
		consoleConfig.AppName = newArtifact.Name
		consoleConfig.Ports = detectedPorts
		consoleConfig.BaseImageVersion = dotnet.DefaultBaseImageVersion

		if imageName.ImageName == "" {
			imageName.ImageName = common.MakeStringContainerImageNameCompliant(serviceConfig.ServiceName)
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
			DestPath:       filepath.Join(common.DefaultSourceDir, relServiceDir),
			TemplateConfig: consoleConfig,
		})
		paths := newArtifact.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relServiceDir, common.DefaultDockerfileName)}
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
