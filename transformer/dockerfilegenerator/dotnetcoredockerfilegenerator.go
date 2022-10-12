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
	"net/url"
	"path/filepath"
	"regexp"
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
	"github.com/spf13/cast"
)

const (
	buildStageC                = "dotnetcorebuildstage"
	defaultBuildOutputDir      = "bin/Release"
	defaultBuildOutputSubDir   = "publish"
	defaultDotNetCoreVersion   = "6.0"
	defaultDotNetCoreFramework = "net" + defaultDotNetCoreVersion
)

var (
	dotnetcoreRegex = regexp.MustCompile(`net(?:(?:coreapp)|(?:standard))?(\d+\.\d+)`)
)

// DotNetCoreTemplateConfig implements DotNetCore config interface
type DotNetCoreTemplateConfig struct {
	IncludeBuildStage     bool
	BuildStageImageTag    string
	BuildContainerName    string
	IsNodeJSProject       bool
	PublishProfilePath    string
	IncludeRunStage       bool
	RunStageImageTag      string
	Ports                 []int32
	EntryPointPath        string
	CopyFrom              string
	EnvVariables          map[string]string
	NodeVersion           string
	NodeVersionProperties map[string]string
	PackageManager        string
}

// -----------------------------------------------------------------------------------
// C Sharp Project XML file
// -----------------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------------
// Visual Studio launch settings json file
// -----------------------------------------------------------------------------------

// LaunchSettings is to load the launchSettings.json properties
type LaunchSettings struct {
	Profiles map[string]LaunchProfile `json:"profiles"`
}

// LaunchProfile implements launch profile properties
type LaunchProfile struct {
	CommandName          string            `json:"CommandName"`
	LaunchBrowser        bool              `json:"launchBrowser"`
	ApplicationURL       string            `json:"applicationUrl"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
}

// -----------------------------------------------------------------------------------
// Transformer
// -----------------------------------------------------------------------------------

// DotNetCoreDockerfileGenerator implements the Transformer interface
type DotNetCoreDockerfileGenerator struct {
	Config           transformertypes.Transformer
	Env              *environment.Environment
	DotNetCoreConfig *DotNetCoreDockerfileYamlConfig
	Spec             NodeVersionsMappingSpec
	SortedVersions   []string
}

// DotNetCoreDockerfileYamlConfig represents the configuration of the DotNetCore dockerfile
type DotNetCoreDockerfileYamlConfig struct {
	DefaultDotNetCoreVersion string `yaml:"defaultDotNetCoreVersion"`
	NodejsDockerfileYamlConfig
}

// Init Initializes the transformer
func (t *DotNetCoreDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.Env = env

	// load the config
	t.DotNetCoreConfig = &DotNetCoreDockerfileYamlConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.DotNetCoreConfig); err != nil {
		return fmt.Errorf("failed to load the config for the transformer %+v into %T . Error: %q", t.Config.Spec.Config, t.DotNetCoreConfig, err)
	}
	if t.DotNetCoreConfig.DefaultDotNetCoreVersion == "" {
		t.DotNetCoreConfig.DefaultDotNetCoreVersion = defaultDotNetCoreVersion
	}
	if t.DotNetCoreConfig.DefaultPackageManager == "" {
		t.DotNetCoreConfig.DefaultPackageManager = defaultPackageManager
	}

	// load the version mapping file
	mappingFilePath := filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath)
	spec, err := LoadNodeVersionMappingsFile(mappingFilePath)
	if err != nil {
		return fmt.Errorf("failed to load the node version mappings file at path %s . Error: %q", versionMappingFilePath, err)
	}
	t.Spec = spec
	if t.DotNetCoreConfig.DefaultNodejsVersion == "" {
		t.DotNetCoreConfig.DefaultNodejsVersion = t.Spec.NodeVersions[0][versionKey]
	}
	return nil
}

// GetConfig returns the transformer config
func (t *DotNetCoreDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *DotNetCoreDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
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
		return nil, fmt.Errorf("failed to parse the vs solution file at path %s . Error: %q", slnPath, err)
	}
	if len(relCSProjPaths) == 0 {
		return nil, fmt.Errorf("no child projects found in the solution file at the path: %s", slnPath)
	}
	childProjects := []artifacts.DotNetChildProject{}
	csProjPaths := []string{}

	for _, relCSProjPath := range relCSProjPaths {
		csProjPath := filepath.Join(dir, relCSProjPath)
		configuration, err := dotnetutils.ParseCSProj(csProjPath)
		if err != nil {
			logrus.Errorf("failed to parse the c sharp project file at path %s . Error: %q", csProjPath, err)
			continue
		}
		idx := common.FindIndex(configuration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.TargetFramework != "" })
		if idx == -1 {
			logrus.Debugf("failed to find the target framework in any of the property groups inside the c sharp project file at path %s", csProjPath)
			continue
		}
		targetFramework := configuration.PropertyGroups[idx].TargetFramework
		if !dotnetcoreRegex.MatchString(targetFramework) {
			logrus.Errorf("dot net core tranformer: the c sharp project file at path %s does not have a supported asp.net framework version. Actual version: %s", csProjPath, targetFramework)
			continue
		}
		isAspNet, err := isAspNet(configuration)
		if err != nil {
			logrus.Errorf("failed to determine if the c sharp project file at the path %s is a asp net web project. Error: %q", csProjPath, err)
			continue
		}
		if !isAspNet {
			logrus.Debugf("the c sharp project file at path %s is not an asp net web project. skipping.", csProjPath)
			continue
		}
		// foundNonSilverLightWebProject = true
		childProjectName := dotnetutils.GetChildProjectName(csProjPath)
		normalizedChildProjectName := common.MakeStringK8sServiceNameCompliant(childProjectName)
		csProjPaths = append(csProjPaths, csProjPath)
		childProjects = append(childProjects, artifacts.DotNetChildProject{
			Name:            normalizedChildProjectName,
			OriginalName:    childProjectName,
			RelCSProjPath:   relCSProjPath,
			TargetFramework: targetFramework,
		})
	}
	if len(csProjPaths) == 0 {
		return nil, nil
	}
	namedServices := map[string][]transformertypes.Artifact{
		normalizedAppName: {{
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceRootDirPathType:           {dir},
				artifacts.ServiceDirPathType:               {dir},
				dotnetutils.DotNetCoreSolutionFilePathType: {slnPath},
				dotnetutils.DotNetCoreCsprojFilesPathType:  csProjPaths,
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.DotNetConfigType: artifacts.DotNetConfig{
					IsDotNetCore:          true,
					DotNetAppName:         appName,
					IsSolutionFilePresent: true,
					ChildProjects:         childProjects,
				},
			},
		}},
	}
	return namedServices, nil
}

// Transform transforms the artifacts
func (t *DotNetCoreDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		dotNetConfig := artifacts.DotNetConfig{}
		if err := newArtifact.GetConfig(artifacts.DotNetConfigType, &dotNetConfig); err != nil || !dotNetConfig.IsDotNetCore {
			continue
		}
		if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 || len(newArtifact.Paths[dotnetutils.DotNetCoreCsprojFilesPathType]) == 0 {
			logrus.Errorf("the service directory is missing from the dot net core artifact: %+v", newArtifact)
			continue
		}
		t1, t2, err := t.TransformArtifact(newArtifact, oldArtifacts, dotNetConfig)
		if err != nil {
			logrus.Errorf("failed to trasnform the dot net core artifact: %+v . Error: %q", newArtifact, err)
			continue
		}
		pathMappings = append(pathMappings, t1...)
		artifactsCreated = append(artifactsCreated, t2...)
	}
	return pathMappings, artifactsCreated, nil
}

// TransformArtifact transforms a single artifact
func (t *DotNetCoreDockerfileGenerator) TransformArtifact(newArtifact transformertypes.Artifact, oldArtifacts []transformertypes.Artifact, dotNetConfig artifacts.DotNetConfig) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}

	selectedBuildOption, err := dotnetutils.AskUserForDockerfileType(newArtifact.Name)
	if err != nil {
		return pathMappings, artifactsCreated, fmt.Errorf("failed to ask the user what type of dockerfile they prefer. Error: %q", err)
	}
	logrus.Debugf("user chose to generate Dockefiles that have '%s'", selectedBuildOption)

	// ask the user which child projects should be run in the K8s cluster

	selectedChildProjectNames := []string{}
	for _, childProject := range dotNetConfig.ChildProjects {
		selectedChildProjectNames = append(selectedChildProjectNames, childProject.Name)
	}
	if len(selectedChildProjectNames) > 1 {
		quesKey := fmt.Sprintf(common.ConfigServicesDotNetChildProjectsNamesKey, `"`+newArtifact.Name+`"`)
		desc := fmt.Sprintf("For the multi-project Dot Net Core app '%s', please select all the child projects that should be run as services in the cluster:", newArtifact.Name)
		hints := []string{"deselect any child project that should not be run (example: libraries)"}
		selectedChildProjectNames = qaengine.FetchMultiSelectAnswer(quesKey, desc, hints, selectedChildProjectNames, selectedChildProjectNames, nil)
		if len(selectedChildProjectNames) == 0 {
			return pathMappings, artifactsCreated, fmt.Errorf("user deselected all the child projects of the dot net core multi-project app '%s'", newArtifact.Name)
		}
	}

	serviceDir := newArtifact.Paths[artifacts.ServiceRootDirPathType][0]
	relServiceDir, err := filepath.Rel(t.Env.GetEnvironmentSource(), serviceDir)
	if err != nil {
		return pathMappings, artifactsCreated, fmt.Errorf("failed to make the service directory path %s relative to the source directory %s . Error: %q", serviceDir, t.Env.GetEnvironmentSource(), err)
	}

	ir := irtypes.IR{}
	irPresent := true
	if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err != nil {
		irPresent = false
		logrus.Debugf("failed to load the IR config from the dot net artifact. Error: %q Artifact: %+v", err, newArtifact)
	}

	detectedPorts := ir.GetAllServicePorts()

	// copy over the source dir to hold the dockerfiles we genrate

	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:     transformertypes.SourcePathMappingType,
		DestPath: common.DefaultSourceDir,
	})

	// build is always done at the top level using the .sln file regardless of the build option selected

	imageToCopyFrom := common.MakeStringContainerImageNameCompliant(newArtifact.Name + "-" + buildStageC)
	if selectedBuildOption == dotnetutils.NO_BUILD_STAGE {
		imageToCopyFrom = "" // files will be copied from the local file system instead of a builder image
	}

	// look for a package.json file to see if the project requires nodejs installed in order to build

	isNodeJSProject := false
	nodeVersion := t.DotNetCoreConfig.DefaultNodejsVersion
	packageManager := t.DotNetCoreConfig.DefaultPackageManager
	packageJsonPaths, err := common.GetFilesByName(serviceDir, []string{packageJSONFile}, nil)
	if err != nil {
		return pathMappings, artifactsCreated, fmt.Errorf("failed to look for package.json files in the directory %s . Error: %q", serviceDir, err)
	}
	if len(packageJsonPaths) > 0 {
		versionConstraints := []string{}
		packageManagers := []string{}
		for _, packageJsonPath := range packageJsonPaths {
			packageJson := PackageJSON{}
			if err := common.ReadJSON(packageJsonPath, &packageJson); err != nil {
				logrus.Errorf("failed to parse the package.json file at path %s . Error: %q", packageJsonPath, err)
				continue
			}
			isNodeJSProject = true
			if versionConstraint, ok := packageJson.Engines["node"]; ok {
				versionConstraints = append(versionConstraints, versionConstraint)
			}
			if packageJson.PackageManager != "" {
				parts := strings.Split(packageJson.PackageManager, "@")
				if len(parts) > 0 {
					packageManagers = append(packageManagers, parts[0])
				}
			}
		}
		if len(versionConstraints) > 0 {
			nodeVersion = getNodeVersion(versionConstraints[0], t.DotNetCoreConfig.DefaultNodejsVersion, t.SortedVersions)
		}
		if len(packageManagers) > 0 {
			packageManager = packageManagers[0]
		}
	}

	// generate the base image Dockerfile
	var props map[string]string
	if idx := common.FindIndex(t.Spec.NodeVersions, func(x map[string]string) bool { return x[versionKey] == nodeVersion }); idx != -1 {
		props = t.Spec.NodeVersions[idx]
	}
	if selectedBuildOption == dotnetutils.BUILD_IN_BASE_IMAGE {
		webConfig := DotNetCoreTemplateConfig{
			IncludeBuildStage:     true,
			BuildStageImageTag:    defaultDotNetCoreVersion,
			BuildContainerName:    imageToCopyFrom,
			IsNodeJSProject:       isNodeJSProject,
			NodeVersion:           nodeVersion,
			NodeVersionProperties: props,
			PackageManager:        packageManager,
		}

		// path mapping to generate the Dockerfile for the child project

		dockerfilePath := filepath.Join(common.DefaultSourceDir, relServiceDir, common.DefaultDockerfileName+"."+buildStageC)
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        common.DefaultDockerfileName,
			DestPath:       dockerfilePath,
			TemplateConfig: webConfig,
		})

		// artifacts to inform other transformers of the Dockerfile we generated

		paths := map[transformertypes.PathType][]string{artifacts.DockerfilePathType: {dockerfilePath}}
		serviceName := artifacts.ServiceConfig{ServiceName: newArtifact.Name}
		imageName := artifacts.ImageName{ImageName: imageToCopyFrom}
		dockerfileArtifact := transformertypes.Artifact{
			Name:  imageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ServiceConfigType:   serviceName,
				artifacts.ImageNameConfigType: imageName,
			},
		}
		artifactsCreated = append(artifactsCreated, dockerfileArtifact)
	}

	for _, childProject := range dotNetConfig.ChildProjects {

		// only look at the child modules the user selected

		if !common.IsPresent(selectedChildProjectNames, childProject.Name) {
			logrus.Debugf("skipping the child project '%s' because it wasn't selected", childProject.Name)
			continue
		}

		// parse the .csproj file to get the build output path

		csProjPath := filepath.Join(serviceDir, childProject.RelCSProjPath)
		configuration, err := dotnetutils.ParseCSProj(csProjPath)
		if err != nil {
			logrus.Errorf("failed to parse the c sharp project file at path %s . Error: %q", csProjPath, err)
			continue
		}

		// data to fill the Dockerfile template

		relCSProjDir := filepath.Dir(childProject.RelCSProjPath)
		targetFramework := defaultDotNetCoreFramework
		targetFrameworkVersion := ""
		if idx := common.FindIndex(configuration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.TargetFramework != "" }); idx != -1 {
			targetFramework = configuration.PropertyGroups[idx].TargetFramework
			version := dotnetcoreRegex.FindAllStringSubmatch(targetFramework, -1)
			if len(version) == 0 || len(version[0]) != 2 {
				logrus.Warnf("unable to find compatible version for the '%s' service. Using default version: %s . Actual: %+v", newArtifact.Name, t.DotNetCoreConfig.DefaultDotNetCoreVersion, configuration.PropertyGroups)
				targetFrameworkVersion = t.DotNetCoreConfig.DefaultDotNetCoreVersion
			} else {
				targetFrameworkVersion = version[0][1]
			}
		}
		buildOutputDir := defaultBuildOutputDir
		if idx := common.FindIndex(configuration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.OutputPath != "" }); idx != -1 {
			buildOutputDir = filepath.Clean(common.GetUnixPath(configuration.PropertyGroups[idx].OutputPath))
		}
		relBuildOutputDir := filepath.Join(buildOutputDir, targetFramework, defaultBuildOutputSubDir)
		copyFrom := filepath.Join("/src", relCSProjDir, relBuildOutputDir)
		if selectedBuildOption == dotnetutils.BUILD_IN_EVERY_IMAGE {
			copyFrom = filepath.Join("/src", relBuildOutputDir) // only the files for the child project are in the builder image
		} else if selectedBuildOption == dotnetutils.NO_BUILD_STAGE {
			copyFrom = relBuildOutputDir // files will be copied from the local file system instead of a builder image
		}

		// find all the publish profile files for this child project

		childProjectDir := filepath.Join(serviceDir, relCSProjDir)
		publishProfilePaths, err := common.GetFilesByExt(childProjectDir, []string{".pubxml"})
		if err != nil {
			logrus.Errorf("failed to look for asp net publish profile (.pubxml) files in the directory %s . Error: %q", childProjectDir, err)
			continue
		}

		// select a profile to use for publishing the child project

		qaSubKey := common.JoinQASubKeys(`"`+newArtifact.Name+`"`, "childProjects", `"`+childProject.Name+`"`)
		relSelectedProfilePath, _, err := getPublishProfile(publishProfilePaths, qaSubKey, serviceDir)
		if err != nil {
			logrus.Errorf("failed to select one of the publish profiles for the asp net app. Error: %q Profiles: %+v", err, publishProfilePaths)
			continue
		}

		nodeVersion := t.DotNetCoreConfig.DefaultNodejsVersion
		var props map[string]string
		if idx := common.FindIndex(t.Spec.NodeVersions, func(x map[string]string) bool { return x[versionKey] == nodeVersion }); idx != -1 {
			props = t.Spec.NodeVersions[idx]
		}

		templateConfig := DotNetCoreTemplateConfig{
			IncludeBuildStage:     selectedBuildOption == dotnetutils.BUILD_IN_EVERY_IMAGE,
			BuildStageImageTag:    defaultDotNetCoreVersion,
			BuildContainerName:    imageToCopyFrom,
			EntryPointPath:        childProject.Name + ".dll",
			RunStageImageTag:      targetFrameworkVersion,
			IncludeRunStage:       true,
			CopyFrom:              common.GetUnixPath(copyFrom),
			PublishProfilePath:    common.GetUnixPath(relSelectedProfilePath),
			EnvVariables:          map[string]string{},
			NodeVersion:           nodeVersion,
			NodeVersionProperties: props,
			PackageManager:        t.DotNetCoreConfig.DefaultPackageManager,
		}

		// look for a package.json file to see if the project requires nodejs installed in order to build

		if isNodeJSProject {
			childPackageJsonPaths := common.Filter(packageJsonPaths, func(x string) bool { return common.IsParent(x, childProjectDir) })
			if len(childPackageJsonPaths) > 0 {
				if len(childPackageJsonPaths) > 1 {
					logrus.Warnf("found multiple package.json files for the child project %s . Actual: %+v", childProject.Name, childPackageJsonPaths)
				}
				packageJsonPath := childPackageJsonPaths[0]
				packageJson := PackageJSON{}
				if err := common.ReadJSON(packageJsonPath, &packageJson); err != nil {
					logrus.Errorf("failed to parse the package.json file at path %s . Error: %q", packageJsonPath, err)
				} else {
					templateConfig.IsNodeJSProject = true
					if nodeVersionConstraint, ok := packageJson.Engines["node"]; ok {
						nodeVersion = getNodeVersion(nodeVersionConstraint, t.DotNetCoreConfig.DefaultNodejsVersion, t.SortedVersions)
						var props map[string]string
						if idx := common.FindIndex(t.Spec.NodeVersions, func(x map[string]string) bool { return x[versionKey] == nodeVersion }); idx != -1 {
							props = t.Spec.NodeVersions[idx]
						}
						templateConfig.NodeVersion = nodeVersion
						templateConfig.NodeVersionProperties = props
					}
					if packageJson.PackageManager != "" {
						parts := strings.Split(packageJson.PackageManager, "@")
						if len(parts) > 0 {
							templateConfig.PackageManager = parts[0]
						}
					}
				}
			}
		}

		// look for a launchSettings.json file to get port numbers the app listens on

		launchJsonPaths, err := common.GetFilesByName(childProjectDir, []string{dotnetutils.LaunchSettingsJSON}, nil)
		if err != nil {
			logrus.Errorf("failed to look for launchSettings.json files in the directory %s . Error: %q", childProjectDir, err)
			continue
		}

		childProjectPorts := append([]int32{}, detectedPorts...)
		if len(launchJsonPaths) > 0 {
			if len(launchJsonPaths) > 1 {
				logrus.Warnf("found multiple launchSettings.json files. Actual: %+v", launchJsonPaths)
			}
			launchJsonPath := launchJsonPaths[0]
			launchSettings := LaunchSettings{}
			if err := common.ReadJSON(launchJsonPath, &launchSettings); err != nil {
				logrus.Errorf("failed to parse the launchSettings.json file at path %s . Error: %q", launchJsonPath, err)
				continue
			}
			if v, ok := launchSettings.Profiles[childProject.Name]; ok && v.ApplicationURL != "" {

				// extract ports and set environment variables

				newAppUrls, ports, err := modifyUrlsToListenOnAllAddresses(v.ApplicationURL)
				if err != nil {
					logrus.Errorf("failed to parse and modify the application listen urls: '%s' . Error: %q", v.ApplicationURL, err)
					continue
				}
				templateConfig.EnvVariables["ASPNETCORE_URLS"] = newAppUrls
				for _, port := range ports {
					childProjectPorts = common.AppendIfNotPresent(childProjectPorts, port)
				}
			}
		}
		if len(childProjectPorts) == 0 {
			childProjectPorts = append(childProjectPorts, common.DefaultServicePort)
		}

		// have the user select the ports to use for the child project

		templateConfig.Ports = commonqa.GetPortsForService(childProjectPorts, qaSubKey)

		dockerfilePath := filepath.Join(common.DefaultSourceDir, relServiceDir, relCSProjDir, common.DefaultDockerfileName)
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        common.DefaultDockerfileName,
			DestPath:       dockerfilePath,
			TemplateConfig: templateConfig,
		})

		// artifacts to inform other transformers of the Dockerfile we generated

		paths := map[transformertypes.PathType][]string{artifacts.DockerfilePathType: {dockerfilePath}}
		serviceConfig := artifacts.ServiceConfig{ServiceName: childProject.Name}
		imageName := artifacts.ImageName{ImageName: common.MakeStringContainerImageNameCompliant(childProject.Name)}
		dockerfileArtifact := transformertypes.Artifact{
			Name:  imageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ServiceConfigType:   serviceConfig,
				artifacts.ImageNameConfigType: imageName,
			},
		}
		dockerfileServiceArtifact := transformertypes.Artifact{
			Name:  imageName.ImageName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ServiceConfigType:   serviceConfig,
				artifacts.ImageNameConfigType: imageName,
			},
		}
		if irPresent {
			dockerfileServiceArtifact.Configs[irtypes.IRConfigType] = ir
		}
		artifactsCreated = append(artifactsCreated, dockerfileArtifact, dockerfileServiceArtifact)
	}

	return pathMappings, artifactsCreated, nil
}

// utility functions

// isAspNet checks if the given app is an asp net web app
func isAspNet(configuration dotnet.CSProj) (bool, error) {
	if configuration.ItemGroups == nil || len(configuration.ItemGroups) == 0 {
		if strings.HasSuffix(configuration.Sdk, ".NET.Sdk.Web") {
			return true, nil
		}
		return false, fmt.Errorf("no item groups in project file to parse")
	}
	for _, ig := range configuration.ItemGroups {
		if len(ig.References) > 0 {
			for _, r := range ig.References {
				if dotnet.AspNetWebLib.MatchString(r.Include) {
					return true, nil
				}
			}
		}
		if len(ig.PackageReferences) > 0 {
			for _, r := range ig.PackageReferences {
				if dotnet.AspNetWebLib.MatchString(r.Include) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// getPublishProfile asks the user to select one of the publish profiles for the child project
func getPublishProfile(profilePaths []string, subKey, baseDir string) (string, string, error) {
	if len(profilePaths) == 0 {
		return "", "", nil
	}
	relProfilePaths := []string{}
	for _, profilePath := range profilePaths {
		relProfilePath, err := filepath.Rel(baseDir, profilePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to make the path %s relative to the directory %s . Error: %q", profilePath, baseDir, err)
		}
		relProfilePaths = append(relProfilePaths, relProfilePath)
	}
	relSelectedProfilePath := relProfilePaths[0]
	if len(relProfilePaths) > 1 {
		quesKey := common.JoinQASubKeys(common.ConfigServicesKey, subKey, common.ConfigPublishProfileForServiceKeySegment)
		desc := fmt.Sprintf("Select the profile to be use for publishing the ASP.NET child project %s :", subKey)
		relSelectedProfilePath = qaengine.FetchSelectAnswer(quesKey, desc, nil, relSelectedProfilePath, relProfilePaths, nil)
	}
	selectedProfilePath := filepath.Join(baseDir, relSelectedProfilePath)
	publishUrl, err := parsePublishProfileFile(selectedProfilePath)
	if err != nil {
		return relSelectedProfilePath, "", fmt.Errorf("failed to parse the publish profile file at path %s . Error: %q", selectedProfilePath, err)
	}
	return relSelectedProfilePath, publishUrl, nil
}

// parsePublishProfileFile parses the publish profile to get the PublishUrl
func parsePublishProfileFile(profilePath string) (string, error) {
	publishProfile := PublishProfile{}
	if err := common.ReadXML(profilePath, publishProfile); err != nil {
		return "", fmt.Errorf("failed to read publish profile file at path %s as xml. Error: %q", profilePath, err)
	}
	if publishProfile.PropertyGroup.PublishUrl != "" {
		return common.GetUnixPath(publishProfile.PropertyGroup.PublishUrl), nil
	}
	if publishProfile.PropertyGroup.PublishUrlS != "" {
		return common.GetUnixPath(publishProfile.PropertyGroup.PublishUrlS), nil
	}
	return "", nil
}

func modifyUrlsToListenOnAllAddresses(src string) (string, []int32, error) {
	srcs := strings.Split(src, ";")
	newUrls := []string{}
	ports := []int32{}
	for _, s := range srcs {
		u, err := url.Parse(s)
		if err != nil {
			return "", nil, fmt.Errorf("failed to parse the string '%s' as a url. Error: %q", s, err)
		}
		if u.Scheme == "https" {
			u.Scheme = "http"
		}
		if u.Scheme != "http" {
			continue
		}
		parts := strings.Split(u.Host, ":")
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("expected there to be a host and port separated by a colon. Actual: %#v", u)
		}
		u.Host = "*:" + parts[1]
		newUrls = append(newUrls, u.String())
		if len(parts[1]) == 0 {
			ports = append(ports, 80)
		}
		if port, err := cast.ToInt32E(parts[1]); err == nil {
			ports = append(ports, port)
		}
	}
	return strings.Join(newUrls, ";"), ports, nil
}
