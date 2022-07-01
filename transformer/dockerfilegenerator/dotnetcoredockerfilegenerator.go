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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	dotnetutils "github.com/konveyor/move2kube/transformer/dockerfilegenerator/dotnet"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/konveyor/move2kube/types/source/dotnet"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	defaultDotNetCoreVersion = "5.0"
)

var (
	portRegex       = regexp.MustCompile(`:(\d+)`)
	dotnetcoreRegex = regexp.MustCompile(`net(?:(?:coreapp)|(?:standard))?(\d+\.\d+)`)
)

// DotNetCoreTemplateConfig implements DotNetCore config interface
type DotNetCoreTemplateConfig struct {
	Ports                  []int32
	HTTPPort               int32
	HTTPSPort              int32
	CsprojFileName         string
	CsprojFilePath         string
	IsNodeJSProject        bool
	DotNetVersion          string
	PublishProfileFilePath string
	PublishUrl             string
}

// DotNetCoreDockerfileYamlConfig represents the configuration of the DotNetCore dockerfile
type DotNetCoreDockerfileYamlConfig struct {
	DefaultDotNetCoreVersion string `yaml:"defaultDotNetCoreVersion"`
}

// DotNetCoreDockerfileGenerator implements the Transformer interface
type DotNetCoreDockerfileGenerator struct {
	Config           transformertypes.Transformer
	Env              *environment.Environment
	DotNetCoreConfig *DotNetCoreDockerfileYamlConfig
}

// PublishProfile defines the publish profile
type PublishProfile struct {
	XMLName       xml.Name       `xml:"Project"`
	PropertyGroup *PropertyGroup `xml:"PropertyGroup"`
}

// PropertyGroup has publish properties of the project
type PropertyGroup struct {
	XMLName     xml.Name `xml:"PropertyGroup"`
	PublishUrl  string   `xml:"PublishUrl"`
	PublishUrlS string   `xml:"publishUrl"`
}

//LaunchSettings defines launchSettings.json properties
type LaunchSettings struct {
	Profiles map[string]LaunchProfile `json:"profiles"`
}

//LaunchProfile implements launch profile properties
type LaunchProfile struct {
	CommandName          string            `json:"CommandName"`
	LaunchBrowser        bool              `json:"launchBrowser"`
	ApplicationURL       string            `json:"applicationUrl"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
}

// Init Initializes the transformer
func (t *DotNetCoreDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.DotNetCoreConfig = &DotNetCoreDockerfileYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.DotNetCoreConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.DotNetCoreConfig, err)
		return err
	}
	if t.DotNetCoreConfig.DefaultDotNetCoreVersion == "" {
		t.DotNetCoreConfig.DefaultDotNetCoreVersion = defaultDotNetCoreVersion
	}
	return nil
}

// GetConfig returns the transformer config
func (t *DotNetCoreDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// getPublishProfile returns the publish profile for the service
func getPublishProfile(publishProfileFilesPath []string, serviceName, baseDir string) (string, string) {
	var publishProfileFileRelPath, publishUrl string
	var publishProfileFilesRelPath []string
	var err error
	for _, publishProfileFilePath := range publishProfileFilesPath {
		if publishProfileFileRelPath, err := filepath.Rel(baseDir, publishProfileFilePath); err == nil {
			publishProfileFilesRelPath = append(publishProfileFilesRelPath, publishProfileFileRelPath)
		}
	}
	if len(publishProfileFilesRelPath) == 1 {
		publishProfileFileRelPath = publishProfileFilesRelPath[0]
	} else if len(publishProfileFilesRelPath) > 1 {
		quesKey := common.JoinQASubKeys(common.ConfigServicesKey, serviceName, common.ConfigPublishProfileForServiceKeySegment)
		desc := fmt.Sprintf("Select the publish profile to be used for the service %s :", serviceName)
		hints := []string{fmt.Sprintf("Selected publish profile will be used for publishing the service %s", serviceName)}
		publishProfileFileRelPath = qaengine.FetchSelectAnswer(quesKey, desc, hints, publishProfileFilesRelPath[0], publishProfileFilesRelPath)
	}
	if publishProfileFileRelPath != "" {
		publishUrl, err = parsePublishProfileFile(filepath.Join(baseDir, publishProfileFileRelPath))
		if err != nil {
			logrus.Errorf("Error while parsing the publish profile %s", err)
			return publishProfileFileRelPath, ""
		}
	}
	return common.GetUnixPath(publishProfileFileRelPath), publishUrl
}

// findPublishProfiles returns the publish profiles of the cs project
func findPublishProfiles(csprojFilePath string) []string {
	publishProfileFiles, err := common.GetFilesByExt(filepath.Dir(csprojFilePath), []string{".pubxml"})
	if err != nil {
		logrus.Debugf("Error while finding publish profile (.pubxml) files %s", err)
		return []string{}
	}
	return publishProfileFiles
}

// parsePublishProfileFile parses the publish profile to get the PublishUrl
func parsePublishProfileFile(publishProfileFilePath string) (string, error) {
	var publishUrl string
	publishProfile := &PublishProfile{}
	err := common.ReadXML(publishProfileFilePath, publishProfile)
	if err != nil {
		logrus.Errorf("Unable to read publish profile file (%s) : %s", publishProfileFilePath, err)
		return "", err
	}
	if publishProfile.PropertyGroup.PublishUrl != "" {
		publishUrl = common.GetUnixPath(publishProfile.PropertyGroup.PublishUrl)
	} else if publishProfile.PropertyGroup.PublishUrlS != "" {
		publishUrl = common.GetUnixPath(publishProfile.PropertyGroup.PublishUrlS)
	}
	return publishUrl, nil
}

// getCsprojFileForService returns the start-up csproj file used by a service
func getCsprojFileForService(csprojFilesPath []string, baseDir string, serviceName string) string {
	var defaultCsprojFileIndex int
	for i, csprojFilePath := range csprojFilesPath {
		publishProfileFiles := findPublishProfiles(csprojFilePath)
		if len(publishProfileFiles) != 0 {
			defaultCsprojFileIndex = i
			break
		}
	}
	var csprojFilesRelPath []string
	for _, csprojFilePath := range csprojFilesPath {
		if csprojFileRelPath, err := filepath.Rel(baseDir, csprojFilePath); err == nil {
			csprojFileRelPath = common.GetUnixPath(csprojFileRelPath)
			csprojFilesRelPath = append(csprojFilesRelPath, csprojFileRelPath)
		}
	}
	quesKey := common.JoinQASubKeys(common.ConfigServicesKey, serviceName, common.ConfigCsprojFileForServiceKeySegment)
	desc := fmt.Sprintf("Select the csproj file to be used for the service %s :", serviceName)
	hints := []string{fmt.Sprintf("Selected csproj file will be used for starting the service %s", serviceName)}
	return qaengine.FetchSelectAnswer(quesKey, desc, hints, csprojFilesRelPath[defaultCsprojFileIndex], csprojFilesRelPath)
}

// getCsprojPathsFromSolutionFile parses the solution file for cs project file paths
func getCsprojPathsFromSolutionFile(inputPath string) ([]string, error) {
	solFileTxt, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("could not open the solution file: %s", err)
	}
	serviceDirPaths := []string{}
	matches := dotnet.ProjBlockRegex.FindAllStringSubmatch(string(solFileTxt), -1)
	for _, match := range matches {
		serviceDirPath := match[1]
		serviceDirPath = strings.TrimSpace(serviceDirPath)
		serviceDirPath = common.GetUnixPath(serviceDirPath)
		if filepath.Ext(serviceDirPath) != dotnetutils.CSPROJ_FILE_EXT {
			continue
		}
		serviceDirPaths = append(serviceDirPaths, serviceDirPath)
	}
	return serviceDirPaths, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *DotNetCoreDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	var solutionFilePath string
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	appName := ""
	dotNetCoreCsprojPaths := []string{}
	for _, de := range dirEntries {
		if filepath.Ext(de.Name()) != dotnet.VISUAL_STUDIO_SOLUTION_FILE_EXT {
			continue
		}
		csProjPaths, err := getCsprojPathsFromSolutionFile(filepath.Join(dir, de.Name()))
		if err != nil {
			logrus.Errorf("Error while parsing the project solution file (%s) - %s", filepath.Join(dir, de.Name()), err)
			continue
		}
		if len(csProjPaths) == 0 {
			logrus.Errorf("No projects available for the solution: %s", de.Name())
			continue
		}
		for _, csPath := range csProjPaths {
			projPath := filepath.Join(dir, csPath)
			csprojConfiguration := &dotnet.CSProj{}
			if err := common.ReadXML(projPath, csprojConfiguration); err != nil {
				logrus.Debugf("Error while reading the csproj file (%s) : %s", projPath, err)
				continue
			}
			if idx := common.FindIndex(csprojConfiguration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.TargetFramework != "" }); idx != -1 &&
				dotnetcoreRegex.MatchString(csprojConfiguration.PropertyGroups[idx].TargetFramework) {
				dotNetCoreCsprojPaths = append(dotNetCoreCsprojPaths, projPath)
			} else {
				logrus.Debugf("unable to find compatible ASP.NET Core target framework for the csproj file at path %s hence skipping. Actual: %#v", projPath, csprojConfiguration.PropertyGroups)
			}
		}
		if len(dotNetCoreCsprojPaths) == 0 {
			return nil, nil
		}
		appName = strings.TrimSuffix(filepath.Base(de.Name()), filepath.Ext(de.Name()))
		solutionFilePath = filepath.Join(dir, de.Name())

		// Exit soon of after the solution file is found
		break
	}

	if appName == "" {
		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, de := range dirEntries {
			if de.IsDir() {
				continue
			}
			if filepath.Ext(de.Name()) != dotnetutils.CSPROJ_FILE_EXT {
				continue
			}
			csprojFilePath := filepath.Join(dir, de.Name())
			csprojConfiguration := &dotnet.CSProj{}
			if err := common.ReadXML(csprojFilePath, csprojConfiguration); err != nil {
				logrus.Errorf("unable to read the csproj file at path %s . Error: %q", csprojFilePath, err)
				continue
			}
			if idx := common.FindIndex(csprojConfiguration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.TargetFramework != "" }); idx != -1 &&
				dotnetcoreRegex.MatchString(csprojConfiguration.PropertyGroups[idx].TargetFramework) {
				dotNetCoreCsprojPaths = append(dotNetCoreCsprojPaths, csprojFilePath)
				appName = strings.TrimSuffix(filepath.Base(csprojFilePath), filepath.Ext(csprojFilePath))
				break // Exit soon after the valid csproj file is found
			}
			logrus.Debugf("unable to find compatible ASP.NET Core target framework for the csproj file at path %s hence skipping. Actual: %#v", csprojFilePath, csprojConfiguration.PropertyGroups)
		}
	}

	if appName == "" {
		return nil, nil
	}
	appName = common.NormalizeForMetadataName(appName)

	dotnetcoreService := transformertypes.Artifact{
		Paths: map[transformertypes.PathType][]string{
			artifacts.ServiceDirPathType:              {dir},
			dotnetutils.DotNetCoreCsprojFilesPathType: dotNetCoreCsprojPaths,
		},
	}
	if solutionFilePath != "" {
		dotnetcoreService.Paths[dotnetutils.DotNetCoreSolutionFilePathType] = []string{solutionFilePath}
	}
	services = map[string][]transformertypes.Artifact{
		appName: {dotnetcoreService},
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *DotNetCoreDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
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
		ports := ir.GetAllServicePorts()
		var dotNetCoreTemplateConfig DotNetCoreTemplateConfig
		var csprojRelFilePath string
		if len(a.Paths[dotnetutils.DotNetCoreCsprojFilesPathType]) == 1 {
			csprojRelFilePath, err = filepath.Rel(a.Paths[artifacts.ServiceDirPathType][0], a.Paths[dotnetutils.DotNetCoreCsprojFilesPathType][0])
			if err != nil {
				logrus.Errorf("Unable to convert csproj file path %s to be relative : %s", a.Paths[dotnetutils.DotNetCoreCsprojFilesPathType][0], err)
				continue
			}
		} else {
			csprojRelFilePath = getCsprojFileForService(a.Paths[dotnetutils.DotNetCoreCsprojFilesPathType], a.Paths[artifacts.ServiceDirPathType][0], a.Name)
		}
		publishProfileFilesPath := findPublishProfiles(filepath.Join(a.Paths[artifacts.ServiceDirPathType][0], csprojRelFilePath))
		dotNetCoreTemplateConfig.PublishProfileFilePath, dotNetCoreTemplateConfig.PublishUrl = getPublishProfile(publishProfileFilesPath, a.Name, a.Paths[artifacts.ServiceDirPathType][0])
		dotNetCoreTemplateConfig.CsprojFilePath = csprojRelFilePath
		dotnetProjectName := strings.TrimSuffix(filepath.Base(dotNetCoreTemplateConfig.CsprojFilePath), filepath.Ext(dotNetCoreTemplateConfig.CsprojFilePath))
		dotNetCoreTemplateConfig.CsprojFileName = dotnetProjectName
		jsonFiles, err := common.GetFilesByName(a.Paths[artifacts.ServiceDirPathType][0], []string{dotnetutils.LaunchSettingsJSON, packageJSONFile}, nil)
		if err != nil {
			logrus.Debugf("Error while finding json files %s", err)
		}
		for _, jsonFile := range jsonFiles {
			if filepath.Base(jsonFile) == dotnetutils.LaunchSettingsJSON {
				launchSettings := LaunchSettings{}
				if err := common.ReadJSON(jsonFile, &launchSettings); err != nil {
					logrus.Errorf("Unable to read the launchSettings.json file: %s", err)
					continue
				}
				if _, ok := launchSettings.Profiles[dotnetProjectName]; ok {
					if launchSettings.Profiles[dotnetProjectName].ApplicationURL != "" {
						Urls := strings.Split(launchSettings.Profiles[dotnetProjectName].ApplicationURL, ";")
						for _, url := range Urls {
							portStr := portRegex.FindAllString(url, 1)[0]
							port, err := strconv.ParseInt(strings.TrimPrefix(portStr, ":"), 10, 32)
							if err != nil {
								logrus.Errorf("Error while converting the port from string to int : %s", err)
							} else if strings.HasPrefix(url, "https://") {
								dotNetCoreTemplateConfig.HTTPSPort = int32(port)
							} else if strings.HasPrefix(url, "http://") {
								dotNetCoreTemplateConfig.HTTPPort = int32(port)
							}
						}
						if !common.IsPresent(ports, dotNetCoreTemplateConfig.HTTPPort) {
							ports = append(ports, dotNetCoreTemplateConfig.HTTPPort)
						}
					}
				}
			}
			if filepath.Base(jsonFile) == packageJSONFile {
				var packageJSON PackageJSON
				if err := common.ReadJSON(jsonFile, &packageJSON); err != nil {
					logrus.Debugf("Unable to read the package.json file: %s", err)
				} else {
					dotNetCoreTemplateConfig.IsNodeJSProject = true
				}
			}
		}
		dotNetCoreTemplateConfig.Ports = commonqa.GetPortsForService(ports, a.Name)
		csprojConfiguration := &dotnet.CSProj{}
		err = common.ReadXML(filepath.Join(a.Paths[artifacts.ServiceDirPathType][0], csprojRelFilePath), csprojConfiguration)
		if err != nil {
			logrus.Errorf("Could not read the project file (%s) : %s", filepath.Join(a.Paths[artifacts.ServiceDirPathType][0], csprojRelFilePath), err)
			continue
		}
		idx := common.FindIndex(csprojConfiguration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.TargetFramework != "" })
		if idx == -1 {
			logrus.Errorf("failed to find the target framework in any of the property groups inside the .csproj file")
			continue
		}
		cand := csprojConfiguration.PropertyGroups[idx]
		frameworkVersion := dotnetcoreRegex.FindAllStringSubmatch(cand.TargetFramework, -1)
		if len(frameworkVersion) != 0 && len(frameworkVersion[0]) == 2 {
			dotNetCoreTemplateConfig.DotNetVersion = frameworkVersion[0][1]
		}
		if dotNetCoreTemplateConfig.DotNetVersion == "" {
			logrus.Debugf("unable to find compatible version for the '%s' service. Using default version: %s . Actual: %#v", a.Name, t.DotNetCoreConfig.DefaultDotNetCoreVersion, csprojConfiguration.PropertyGroups)
			dotNetCoreTemplateConfig.DotNetVersion = t.DotNetCoreConfig.DefaultDotNetCoreVersion
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
			TemplateConfig: dotNetCoreTemplateConfig,
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
