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
	"fmt"
	"path/filepath"
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
	buildStageC             = "dotnetwebbuildstage"
	defaultBuildOutputDir   = "bin"
	defaultRunStageImageTag = "4.8"
)

func (t *WinWebAppDockerfileGenerator) getImageTagFromVersion(version string) string {
	for imageTag, versions := range t.imageTagToSupportedVersions {
		if common.FindIndex(versions, func(v string) bool { return v == version || "v"+v == version }) != -1 {
			return imageTag
		}
	}
	return defaultRunStageImageTag
}

// WebTemplateConfig contains the data to fill the Dockerfile template
type WebTemplateConfig struct {
	AppName            string
	Ports              []int32
	IncludeBuildStage  bool
	BuildStageImageTag string
	BuildContainerName string
	IncludeRunStage    bool
	RunStageImageTag   string
	CopyFrom           string
}

// WinWebAppDockerfileGenerator implements the Transformer interface
type WinWebAppDockerfileGenerator struct {
	Config                      transformertypes.Transformer
	Env                         *environment.Environment
	imageTagToSupportedVersions map[string][]string
}

// Init Initializes the transformer
func (t *WinWebAppDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.Env = env
	mappingFile := DotNetWindowsVersionMapping{}
	mappingFilePath := filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath)
	if err := common.ReadMove2KubeYaml(mappingFilePath, &mappingFile); err != nil {
		return fmt.Errorf("failed to load the dot net windows version mapping file at path %s . Error: %q", mappingFilePath, err)
	}
	if len(mappingFile.Spec.ImageTagToSupportedVersions) == 0 {
		return fmt.Errorf("the mapping file at path %s is invalid", mappingFilePath)
	}
	t.imageTagToSupportedVersions = mappingFile.Spec.ImageTagToSupportedVersions
	return nil
}

// GetConfig returns the transformer config
func (t *WinWebAppDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *WinWebAppDockerfileGenerator) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	slnPaths, err := common.GetFilesByExtInCurrDir(dir, []string{dotnet.VISUAL_STUDIO_SOLUTION_FILE_EXT})
	if err != nil {
		return nil, fmt.Errorf("failed to list the dot net visual studio solution files in the directory %s . Error: %q", dir, err)
	}
	appName := filepath.Base(dir)
	if len(slnPaths) == 0 {
		csProjPaths, err := common.GetFilesByExtInCurrDir(dir, []string{dotnetutils.CSPROJ_FILE_EXT})
		if err != nil {
			return nil, fmt.Errorf("failed to list the dot net c sharp project files in the directory %s . Error: %q", dir, err)
		}
		if len(csProjPaths) == 0 {
			return nil, nil
		}
		childProjects := []artifacts.DotNetChildProject{}
		for _, csProjPath := range csProjPaths {
			configuration, err := parseCSProj(csProjPath)
			if err != nil {
				logrus.Errorf("failed to parse the c sharp project file at path %s . Error: %q", csProjPath, err)
				continue
			}
			idx := common.FindIndex(configuration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.TargetFrameworkVersion != "" })
			if idx == -1 {
				logrus.Errorf("failed to find the target framework in any of the property groups inside the c sharp project file at path %s", csProjPath)
				continue
			}
			targetFrameworkVersion := configuration.PropertyGroups[idx].TargetFrameworkVersion
			if !dotnet.Version4.MatchString(targetFrameworkVersion) {
				logrus.Errorf("the c sharp project file at path %s does not have a supported framework version. Actual version: %s", csProjPath, targetFrameworkVersion)
				continue
			}
			childProjects = append(childProjects, artifacts.DotNetChildProject{
				Name:          getChildProjectName(csProjPath),
				RelCSProjPath: filepath.Base(csProjPath),
			})
		}
		namedServices := map[string][]transformertypes.Artifact{
			appName: {{
				Name: appName,
				Type: artifacts.ServiceArtifactType,
				Paths: map[transformertypes.PathType][]string{
					artifacts.ServiceRootDirPathType:          {dir},
					artifacts.ServiceDirPathType:              {dir},
					dotnetutils.DotNetCoreCsprojFilesPathType: csProjPaths,
				},
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.DotNetConfigType: artifacts.DotNetConfig{
						DotNetAppName:         appName,
						IsSolutionFilePresent: false,
						ChildProjects:         childProjects,
					},
				},
			}},
		}
		return namedServices, nil
	}
	if len(slnPaths) > 1 {
		logrus.Debugf("more than one visual studio solution file detected. Number of .sln files %d", len(slnPaths))
	}
	slnPath := slnPaths[0]
	relCSProjPaths, err := getCSProjPathsFromSlnFile(slnPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the vs solution file at path %s . Error: %q", slnPath, err)
	}
	if len(relCSProjPaths) == 0 {
		return nil, fmt.Errorf("no child projects found in the solution file at the path: %s", slnPath)
	}
	childProjects := []artifacts.DotNetChildProject{}
	csProjPaths := []string{}
	foundNonSilverLightWebProject := false
	for _, relCSProjPath := range relCSProjPaths {
		logrus.Debugf("looking at the c sharp project file at path %s", relCSProjPath)
		csProjPath := filepath.Join(dir, relCSProjPath)
		configuration, err := parseCSProj(csProjPath)
		if err != nil {
			logrus.Errorf("failed to parse the c sharp project file at path %s . Error: %q", csProjPath, err)
			continue
		}
		idx := common.FindIndex(configuration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.TargetFrameworkVersion != "" })
		if idx == -1 {
			logrus.Debugf("failed to find the target framework in any of the property groups inside the c sharp project file at path %s", csProjPath)
			continue
		}
		targetFrameworkVersion := configuration.PropertyGroups[idx].TargetFrameworkVersion
		if !dotnet.Version4.MatchString(targetFrameworkVersion) {
			logrus.Errorf("webapp dot net tranformer: the c sharp project file at path %s does not have a supported framework version. Actual version: %s", csProjPath, targetFrameworkVersion)
			continue
		}
		isWebProj, err := isWeb(configuration)
		if err != nil {
			logrus.Errorf("failed to determine if the c sharp project file at the path %s is a web project. Error: %q", csProjPath, err)
			continue
		}
		if !isWebProj {
			logrus.Debugf("the c sharp project file at path %s is not a web project", csProjPath)
			continue
		}
		isSLProj, err := isSilverlight(configuration)
		if err != nil {
			logrus.Errorf("failed to determine if the c sharp project file at the path %s is a SilverLight project. Error: %q", csProjPath, err)
			continue
		}
		if isSLProj {
			logrus.Debugf("the c sharp project file at path %s is a SilverLight project", csProjPath)
			continue
		}
		foundNonSilverLightWebProject = true
		childProjectName := getChildProjectName(csProjPath)
		csProjPaths = append(csProjPaths, csProjPath)
		childProjects = append(childProjects, artifacts.DotNetChildProject{
			Name:          childProjectName,
			RelCSProjPath: relCSProjPath,
		})
	}
	if !foundNonSilverLightWebProject {
		return nil, nil
	}
	namedServices := map[string][]transformertypes.Artifact{
		appName: {{
			Name: appName,
			Type: artifacts.ServiceArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceRootDirPathType:          {dir},
				artifacts.ServiceDirPathType:              {dir},
				dotnetutils.DotNetCoreCsprojFilesPathType: csProjPaths,
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.DotNetConfigType: artifacts.DotNetConfig{
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
func (t *WinWebAppDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		dotNetConfig := artifacts.DotNetConfig{}
		if err := newArtifact.GetConfig(artifacts.DotNetConfigType, &dotNetConfig); err != nil {
			continue
		}
		if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 || len(newArtifact.Paths[dotnetutils.DotNetCoreCsprojFilesPathType]) == 0 {
			logrus.Errorf("service directories are missing from the dot net artifact")
			continue
		}
		selectedBuildOption, err := dotnetutils.AskUserForDockerfileType(dotNetConfig.DotNetAppName)
		if err != nil {
			logrus.Errorf("failed to ask the user what type of dockerfile they prefer. Error: %q", err)
			continue
		}
		logrus.Debugf("user chose to generate Dockefiles that have '%s'", selectedBuildOption)

		// ask the user which child projects should be run in the K8s cluster

		selectedChildProjectNames := []string{}
		for _, childProject := range dotNetConfig.ChildProjects {
			selectedChildProjectNames = append(selectedChildProjectNames, childProject.Name)
		}
		if len(selectedChildProjectNames) > 1 {
			quesKey := fmt.Sprintf(common.ConfigServicesDotNetChildProjectsNamesKey, dotNetConfig.DotNetAppName)
			desc := fmt.Sprintf("For the multi-project Dot Net app '%s', please select all the child projects that should be run as services in the cluster:", dotNetConfig.DotNetAppName)
			hints := []string{"deselect any child project that should not be run (example: libraries)"}
			selectedChildProjectNames = qaengine.FetchMultiSelectAnswer(quesKey, desc, hints, selectedChildProjectNames, selectedChildProjectNames)
			if len(selectedChildProjectNames) == 0 {
				return pathMappings, artifactsCreated, fmt.Errorf("user deselected all the child projects of the dot net multi-project app '%s'", dotNetConfig.DotNetAppName)
			}
		}

		serviceDir := newArtifact.Paths[artifacts.ServiceRootDirPathType][0]
		relServiceDir, err := filepath.Rel(t.Env.GetEnvironmentSource(), serviceDir)
		if err != nil {
			logrus.Errorf("failed to make the service directory path %s relative to the source directory %s . Error: %q", serviceDir, t.Env.GetEnvironmentSource(), err)
			continue
		}

		ir := irtypes.IR{}
		irPresent := true
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T . Error: %q", ir, err)
		}

		detectedPorts := ir.GetAllServicePorts()
		if len(detectedPorts) == 0 {
			detectedPorts = append(detectedPorts, common.DefaultServicePort)
		}

		// copy over the source dir to hold the dockerfiles we genrate

		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		})

		// build is always done at the top level using the .sln file regardless of the build option selected

		imageToCopyFrom := common.MakeStringContainerImageNameCompliant(dotNetConfig.DotNetAppName + "-" + buildStageC)
		if selectedBuildOption == dotnetutils.NO_BUILD_STAGE {
			imageToCopyFrom = "" // files will be copied from the local file system instead of a builder image
		}

		// generate the base image Dockerfile

		if selectedBuildOption == dotnetutils.BUILD_IN_BASE_IMAGE {
			webConfig := WebTemplateConfig{
				AppName:            dotNetConfig.DotNetAppName,
				BuildStageImageTag: dotnet.DefaultBaseImageVersion,
				IncludeBuildStage:  true,
				IncludeRunStage:    false,
				BuildContainerName: imageToCopyFrom,
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
			serviceName := artifacts.ServiceConfig{ServiceName: dotNetConfig.DotNetAppName}
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

			if !common.IsStringPresent(selectedChildProjectNames, childProject.Name) {
				continue
			}

			// parse the .csproj file to get the output path
			csProjPath := filepath.Join(serviceDir, childProject.RelCSProjPath)
			configuration, err := parseCSProj(csProjPath)
			if err != nil {
				logrus.Errorf("failed to parse the c sharp project file at path %s . Error: %q", csProjPath, err)
				continue
			}

			// have the user select the ports to use for the child project

			selectedPorts := commonqa.GetPortsForService(detectedPorts, common.JoinQASubKeys(dotNetConfig.DotNetAppName, "childProjects", childProject.Name))

			// data to fill the Dockerfile template

			relCSProjDir := filepath.Dir(childProject.RelCSProjPath)
			buildOutputDir := defaultBuildOutputDir
			if idx := common.FindIndex(configuration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.OutputPath != "" }); idx != -1 {
				buildOutputDir = filepath.Clean(common.GetUnixPath(configuration.PropertyGroups[idx].OutputPath))
			}
			copyFrom := filepath.Join("/app", relCSProjDir, buildOutputDir) + "/"
			if selectedBuildOption == dotnetutils.NO_BUILD_STAGE {
				copyFrom = buildOutputDir + "/" // files will be copied from the local file system instead of a builder image
			}

			targetFrameworkVersion := ""
			if idx := common.FindIndex(configuration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.TargetFrameworkVersion != "" }); idx != -1 {
				targetFrameworkVersion = configuration.PropertyGroups[idx].TargetFrameworkVersion
			}

			webConfig := WebTemplateConfig{
				AppName:            childProject.Name,
				Ports:              selectedPorts,
				BuildStageImageTag: dotnet.DefaultBaseImageVersion,
				IncludeBuildStage:  selectedBuildOption == dotnetutils.BUILD_IN_EVERY_IMAGE,
				IncludeRunStage:    true,
				BuildContainerName: imageToCopyFrom,
				CopyFrom:           copyFrom,
				RunStageImageTag:   t.getImageTagFromVersion(targetFrameworkVersion),
			}

			// path mapping to generate the Dockerfile for the child project

			dockerfilePath := filepath.Join(common.DefaultSourceDir, relServiceDir, relCSProjDir, common.DefaultDockerfileName)
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:           transformertypes.TemplatePathMappingType,
				SrcPath:        common.DefaultDockerfileName,
				DestPath:       dockerfilePath,
				TemplateConfig: webConfig,
			})

			// artifacts to inform other transformers of the Dockerfile we generated

			paths := map[transformertypes.PathType][]string{artifacts.DockerfilePathType: {dockerfilePath}}
			serviceName := artifacts.ServiceConfig{ServiceName: childProject.Name}
			imageName := artifacts.ImageName{ImageName: common.MakeStringContainerImageNameCompliant(childProject.Name)}
			dockerfileArtifact := transformertypes.Artifact{
				Name:  imageName.ImageName,
				Type:  artifacts.DockerfileArtifactType,
				Paths: paths,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.ServiceConfigType:   serviceName,
					artifacts.ImageNameConfigType: imageName,
				},
			}
			dockerfileServiceArtifact := transformertypes.Artifact{
				Name:  imageName.ImageName,
				Type:  artifacts.DockerfileForServiceArtifactType,
				Paths: paths,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.ServiceConfigType:   serviceName,
					artifacts.ImageNameConfigType: imageName,
				},
			}
			if irPresent {
				dockerfileServiceArtifact.Configs[irtypes.IRConfigType] = ir
			}
			artifactsCreated = append(artifactsCreated, dockerfileArtifact, dockerfileServiceArtifact)
		}
	}
	return pathMappings, artifactsCreated, nil
}

func getChildProjectName(csProjPath string) string {
	return strings.TrimSuffix(filepath.Base(csProjPath), filepath.Ext(csProjPath))
}
