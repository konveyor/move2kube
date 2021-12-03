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
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/konveyor/move2kube/types/source/dotnet"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// DotNet5DockerfileGenerator implements the Transformer interface
type DotNet5DockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

const (
	csproj             = ".csproj"
	launchSettingsJSON = "launchSettings.json"
	// CsprojFilePathType points to the .csproj file path
	CsprojFilePathType transformertypes.PathType = "CsprojFilePath"
)

var (
	portRegex          = regexp.MustCompile(`[0-9]+`)
	dotNetCoreVersions = []string{"net5.0", "netcoreapp2.1", "netstandard2.0"}
)

// DotNet5TemplateConfig implements Nodejs config interface
type DotNet5TemplateConfig struct {
	Port            int32
	HTTPPort        int32
	HTTPSPort       int32
	AppName         string
	CsprojFilePath  string
	IsNodeJSProject bool
}

//LaunchSettings defines launchSettings.json properties
type LaunchSettings struct {
	Profiles map[string]LaunchProfile `json:"profiles"`
}

//LaunchProfile implements launch profile properties
type LaunchProfile struct {
	CommandName          string            `json:"CommandName"`
	DotnetRunMessages    string            `json:"dotnetRunMessages"`
	LaunchBrowser        bool              `json:"launchBrowser"`
	ApplicationURL       string            `json:"applicationUrl"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
}

// Init Initializes the transformer
func (t *DotNet5DockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *DotNet5DockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *DotNet5DockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Error while trying to read directory : %s", err)
		return nil, err
	}
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		ext := filepath.Ext(de.Name())
		if ext != csproj {
			continue
		}
		csprojFile := filepath.Join(dir, de.Name())
		xmlFile, err := os.Open(csprojFile)
		if err != nil {
			logrus.Errorf("Could not open the csproj file: %s", err)
		}
		defer xmlFile.Close()
		byteValue, err := io.ReadAll(xmlFile)
		if err != nil {
			logrus.Errorf("Could not read the csproj file: %s", err)
		}
		configuration := dotnet.CSProj{}
		err = xml.Unmarshal(byteValue, &configuration)
		if err != nil {
			logrus.Errorf("Could not parse the project file %s", err)
		}
		serviceName := strings.TrimSuffix(filepath.Base(csprojFile), filepath.Ext(csprojFile))
		if common.IsStringPresent(dotNetCoreVersions, configuration.PropertyGroup.TargetFramework) {
			services = map[string][]transformertypes.Artifact{
				serviceName: {{
					Paths: map[string][]string{
						artifacts.ProjectPathPathType: {dir},
						CsprojFilePathType:            {csprojFile},
					},
				}},
			}
			return services, nil
		}
	}
	return nil, nil
}

// Transform transforms the artifacts
func (t *DotNet5DockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
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
		var dotnet5Config DotNet5TemplateConfig
		dotnet5Config.AppName = a.Name
		dotnet5Config.CsprojFilePath, err = filepath.Rel(a.Paths[artifacts.ProjectPathPathType][0], a.Paths[CsprojFilePathType][0])
		if err != nil {
			logrus.Errorf("Error while getting the relative path of csproj file%s", err)
		}
		jsonFiles, err := common.GetFilesByExt(a.Paths[artifacts.ProjectPathPathType][0], []string{".json"})
		if err != nil {
			logrus.Errorf("Error while finding json files %s", err)
		}
		for _, jsonFile := range jsonFiles {
			if filepath.Base(jsonFile) == launchSettingsJSON {
				launchSettings := LaunchSettings{}
				if err := common.ReadJSON(jsonFile, &launchSettings); err != nil {
					logrus.Errorf("Unable to read the launchSettings.json file: %s", err)
					continue
				}
				if launchSettings.Profiles[dotnet5Config.AppName].ApplicationURL != "" {
					Urls := strings.Split(launchSettings.Profiles[dotnet5Config.AppName].ApplicationURL, ";")
					for _, url := range Urls {
						portStr := portRegex.FindAllString(url, 1)[0]
						port, err := strconv.ParseInt(portStr, 10, 32)
						if err != nil {
							logrus.Errorf("Error while converting the port from string to int : %s", err)
						} else if strings.HasPrefix(url, "https://") {
							dotnet5Config.HTTPSPort = int32(port)
						} else if strings.HasPrefix(url, "http://") {
							dotnet5Config.HTTPPort = int32(port)
						}
					}
					ports = append(ports, dotnet5Config.HTTPPort)
				}
			}
			if filepath.Base(jsonFile) == packageJSONFile {
				var packageJSON PackageJSON
				if err := common.ReadJSON(jsonFile, &packageJSON); err != nil {
					logrus.Debugf("Unable to read the package.json file: %s", err)
				} else {
					dotnet5Config.IsNodeJSProject = true
				}
			}
		}
		dotnet5Config.Port = commonqa.GetPortForService(ports, a.Name)
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
			TemplateConfig: dotnet5Config,
		})
		paths := a.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, common.DefaultDockerfileName)}
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
		if irPresent {
			dfs.Configs[irtypes.IRConfigType] = ir
		}
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}
