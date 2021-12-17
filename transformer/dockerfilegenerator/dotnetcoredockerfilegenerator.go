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
	"github.com/konveyor/move2kube/types"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/konveyor/move2kube/types/source/dotnet"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

var (
	portRegex        = regexp.MustCompile(`:(\d+)`)
	dotnetcoreRegex1 = regexp.MustCompile(`net\d+\.\d+`)
	dotnetcoreRegex2 = regexp.MustCompile(`netcoreapp\d+\.\d+`)
	dotnetcoreRegex3 = regexp.MustCompile(`netstandard\d+\.\d+`)
)

const (
	csproj             = ".csproj"
	launchSettingsJSON = "launchSettings.json"
	// DotNetCoreCsprojFilesPathType points to the csproj files path of dotnetcore projects
	DotNetCoreCsprojFilesPathType transformertypes.PathType = "DotNetCoreCsprojPathType"
	// DotNetCoreSolutionFilePathType points to the solution file path of dotnetcore project
	DotNetCoreSolutionFilePathType transformertypes.PathType = "DotNetCoreSolutionPathType"
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
	DotNetCoreConfig DotNetCoreDockerfileYamlConfig
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
	t.DotNetCoreConfig = DotNetCoreDockerfileYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, &t.DotNetCoreConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.DotNetCoreConfig, err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *DotNetCoreDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// getDotNetCoreVersion fetches the dotnetcore version from the name-version mapping
func getDotNetCoreVersion(mappingFile string, name string) (version string, err error) {
	var dotNetCoreNameVersionMapping DotNetCoreNameVersionMapping
	if err := common.ReadMove2KubeYaml(mappingFile, &dotNetCoreNameVersionMapping); err != nil {
		logrus.Debugf("Could not load mapping at %s", mappingFile)
		return "", err
	}
	var dotNetCoreFramework string
	if dotnetcoreRegex1.MatchString(name) {
		dotNetCoreFramework = dotnetcoreRegex1.FindString(name)
	} else if dotnetcoreRegex2.MatchString(name) {
		dotNetCoreFramework = dotnetcoreRegex2.FindString(name)
	} else if dotnetcoreRegex3.MatchString(name) {
		dotNetCoreFramework = dotnetcoreRegex3.FindString(name)
	}
	if _, ok := dotNetCoreNameVersionMapping.Spec.NameVersion[dotNetCoreFramework]; ok {
		return dotNetCoreNameVersionMapping.Spec.NameVersion[dotNetCoreFramework], nil
	}
	return "", fmt.Errorf("could not find compatible .NET Core framework version")
}

// DotNetCoreNameVersionMapping stores the dotnetcore name version mappings
type DotNetCoreNameVersionMapping struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             DotNetCoreNameVersionMappingSpec `yaml:"spec,omitempty"`
}

// DotNetCoreNameVersionMappingSpec stores the dotnetcore name version spec
type DotNetCoreNameVersionMappingSpec struct {
	NameVersion map[string]string `yaml:"nameVersions"`
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
		publishProfileFileRelPath = qaengine.FetchSelectAnswer(common.ConfigServicesKey+common.Delim+serviceName+common.Delim+common.ConfigPublishProfileForServiceKeySegment, fmt.Sprintf("Select the publish profile to be used for the service %s :", serviceName), []string{fmt.Sprintf("Selected publish profile will be used for publishing the service %s", serviceName)}, publishProfileFilesRelPath[0], publishProfileFilesRelPath)
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
	return qaengine.FetchSelectAnswer(common.ConfigServicesKey+common.Delim+serviceName+common.Delim+common.ConfigCsprojFileForServiceKeySegment, fmt.Sprintf("Select the csproj file to be used for the service %s :", serviceName), []string{fmt.Sprintf("Selected csproj file will be used for starting the service %s", serviceName)}, csprojFilesRelPath[defaultCsprojFileIndex], csprojFilesRelPath)
}

// getCsprojPathsFromSolutionFile parses the solution file for cs project file paths
func getCsprojPathsFromSolutionFile(inputPath string) ([]string, error) {
	solFileTxt, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("could not open the solution file: %s", err)
	}
	projectPaths := make([]string, 0)
	matches := dotnet.ProjBlockRegex.FindAllStringSubmatch(string(solFileTxt), -1)
	for _, match := range matches {
		projectPath := match[1]
		projectPath = strings.TrimSpace(projectPath)
		projectPath = common.GetUnixPath(projectPath)
		projectPaths = append(projectPaths, projectPath)
	}
	return projectPaths, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *DotNetCoreDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	var solutionFilePath string
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Error while trying to read directory: %s", err)
		return nil, err
	}
	appName := ""
	dotNetCoreCsprojPaths := []string{}
	for _, de := range dirEntries {
		if filepath.Ext(de.Name()) != dotnet.CsSln {
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
			err := common.ReadXML(projPath, csprojConfiguration)
			if err != nil {
				logrus.Debugf("Error while reading the csproj file (%s) : %s", projPath, err)
				continue
			}
			if dotnetcoreRegex1.MatchString(csprojConfiguration.PropertyGroup.TargetFramework) || dotnetcoreRegex2.MatchString(csprojConfiguration.PropertyGroup.TargetFramework) || dotnetcoreRegex3.MatchString(csprojConfiguration.PropertyGroup.TargetFramework) {
				dotNetCoreCsprojPaths = append(dotNetCoreCsprojPaths, projPath)
			} else {
				logrus.Warnf("Unable to find compatible ASP.NET Core target framework %s hence skipping.", csprojConfiguration.PropertyGroup.TargetFramework)
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
			logrus.Errorf("Error while trying to read directory : %s", err)
			return nil, err
		}
		for _, de := range dirEntries {
			if de.IsDir() {
				continue
			}
			if filepath.Ext(de.Name()) != csproj {
				continue
			}
			csprojFile := filepath.Join(dir, de.Name())
			csprojConfiguration := &dotnet.CSProj{}
			err := common.ReadXML(csprojFile, csprojConfiguration)
			if err != nil {
				logrus.Errorf("Unable to read the csproj file (%s) : %s", csprojFile, err)
				continue
			}
			if dotnetcoreRegex1.MatchString(csprojConfiguration.PropertyGroup.TargetFramework) || dotnetcoreRegex2.MatchString(csprojConfiguration.PropertyGroup.TargetFramework) || dotnetcoreRegex3.MatchString(csprojConfiguration.PropertyGroup.TargetFramework) {
				dotNetCoreCsprojPaths = append(dotNetCoreCsprojPaths, csprojFile)
				appName = strings.TrimSuffix(filepath.Base(csprojFile), filepath.Ext(csprojFile))
				break
				// Exit soon of after the valid csproj file is found
			} else {
				logrus.Warnf("Unable to find compatible ASP.NET Core target framework %s hence skipping.", csprojConfiguration.PropertyGroup.TargetFramework)
				continue
			}
		}
	}

	if appName == "" {
		return nil, nil
	}
	appName = common.NormalizeForServiceName(appName)

	dotnetcoreService := transformertypes.Artifact{
		Paths: map[transformertypes.PathType][]string{
			artifacts.ProjectPathPathType: {dir},
			DotNetCoreCsprojFilesPathType: dotNetCoreCsprojPaths,
		},
	}
	if solutionFilePath != "" {
		dotnetcoreService.Paths[DotNetCoreSolutionFilePathType] = []string{solutionFilePath}
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
		if len(a.Paths[DotNetCoreCsprojFilesPathType]) == 1 {
			csprojRelFilePath, err = filepath.Rel(a.Paths[artifacts.ProjectPathPathType][0], a.Paths[DotNetCoreCsprojFilesPathType][0])
			if err != nil {
				logrus.Errorf("Unable to convert csproj file path %s to be relative : %s", a.Paths[DotNetCoreCsprojFilesPathType][0], err)
				continue
			}
		} else {
			csprojRelFilePath = getCsprojFileForService(a.Paths[DotNetCoreCsprojFilesPathType], a.Paths[artifacts.ProjectPathPathType][0], a.Name)
		}
		publishProfileFilesPath := findPublishProfiles(filepath.Join(a.Paths[artifacts.ProjectPathPathType][0], csprojRelFilePath))
		dotNetCoreTemplateConfig.PublishProfileFilePath, dotNetCoreTemplateConfig.PublishUrl = getPublishProfile(publishProfileFilesPath, a.Name, a.Paths[artifacts.ProjectPathPathType][0])
		dotNetCoreTemplateConfig.CsprojFilePath = csprojRelFilePath
		dotnetProjectName := strings.TrimSuffix(filepath.Base(dotNetCoreTemplateConfig.CsprojFilePath), filepath.Ext(dotNetCoreTemplateConfig.CsprojFilePath))
		dotNetCoreTemplateConfig.CsprojFileName = dotnetProjectName
		jsonFiles, err := common.GetFilesByName(a.Paths[artifacts.ProjectPathPathType][0], []string{launchSettingsJSON, packageJSONFile}, nil)
		if err != nil {
			logrus.Debugf("Error while finding json files %s", err)
		}
		for _, jsonFile := range jsonFiles {
			if filepath.Base(jsonFile) == launchSettingsJSON {
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
						if !common.IsInt32Present(ports, dotNetCoreTemplateConfig.HTTPPort) {
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
		err = common.ReadXML(filepath.Join(a.Paths[artifacts.ProjectPathPathType][0], csprojRelFilePath), csprojConfiguration)
		if err != nil {
			logrus.Errorf("Could not read the project file (%s) : %s", filepath.Join(a.Paths[artifacts.ProjectPathPathType][0], csprojRelFilePath), err)
			continue
		}
		dotNetCoreTemplateConfig.DotNetVersion, err = getDotNetCoreVersion(filepath.Join(t.Env.GetEnvironmentContext(), "mappings/dotnetcoreversions.yaml"), csprojConfiguration.PropertyGroup.TargetFramework)
		if err != nil || dotNetCoreTemplateConfig.DotNetVersion == "" {
			logrus.Warnf("Unable to find compatible version for %s service %s target framework. Using default version: %s", a.Name, csprojConfiguration.PropertyGroup.TargetFramework, t.DotNetCoreConfig.DefaultDotNetCoreVersion)
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
