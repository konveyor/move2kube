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
	"github.com/konveyor/move2kube-wasm/qaengine"
	irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	"github.com/konveyor/move2kube-wasm/types/qaengine/commonqa"
	"path/filepath"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	dotnetutils "github.com/konveyor/move2kube-wasm/transformer/dockerfilegenerator/dotnet"
	//irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	//"github.com/konveyor/move2kube-wasm/types/qaengine/commonqa"
	"github.com/konveyor/move2kube-wasm/types/source/dotnet"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/konveyor/move2kube-wasm/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
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
			configuration, err := dotnetutils.ParseCSProj(csProjPath)
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
			if !dotnet.Version4And3_5.MatchString(targetFrameworkVersion) {
				logrus.Errorf("the c sharp project file at path %s does not have a supported framework version. Actual version: %s", csProjPath, targetFrameworkVersion)
				continue
			}
			childProjectName := dotnetutils.GetChildProjectName(csProjPath)
			normalizedChildProjectName := common.MakeStringK8sServiceNameCompliant(childProjectName)
			childProjects = append(childProjects, artifacts.DotNetChildProject{
				Name:            normalizedChildProjectName,
				OriginalName:    childProjectName,
				RelCSProjPath:   filepath.Base(csProjPath),
				TargetFramework: targetFrameworkVersion,
			})
		}
		appName := dotnetutils.GetChildProjectName(csProjPaths[0])
		normalizedAppName := common.MakeStringK8sServiceNameCompliant(appName)
		namedServices := map[string][]transformertypes.Artifact{
			normalizedAppName: {{
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
	foundNonSilverLightWebProject := false
	for _, relCSProjPath := range relCSProjPaths {
		csProjPath := filepath.Join(dir, relCSProjPath)
		configuration, err := dotnetutils.ParseCSProj(csProjPath)
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
		if !dotnet.Version4And3_5.MatchString(targetFrameworkVersion) {
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
		childProjectName := dotnetutils.GetChildProjectName(csProjPath)
		normalizedChildProjectName := common.MakeStringK8sServiceNameCompliant(childProjectName)
		csProjPaths = append(csProjPaths, csProjPath)
		childProjects = append(childProjects, artifacts.DotNetChildProject{
			Name:          normalizedChildProjectName,
			OriginalName:  childProjectName,
			RelCSProjPath: relCSProjPath,
		})
	}
	if !foundNonSilverLightWebProject {
		return nil, nil
	}
	namedServices := map[string][]transformertypes.Artifact{
		normalizedAppName: {{
			Name: normalizedAppName,
			Type: artifacts.ServiceArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceRootDirPathType:           {dir},
				artifacts.ServiceDirPathType:               {dir},
				dotnetutils.DotNetCoreSolutionFilePathType: {slnPath},
				dotnetutils.DotNetCoreCsprojFilesPathType:  csProjPaths,
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.DotNetConfigType: artifacts.DotNetConfig{
					IsDotNetCore:          false,
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
		if err := newArtifact.GetConfig(artifacts.DotNetConfigType, &dotNetConfig); err != nil || dotNetConfig.IsDotNetCore {
			continue
		}
		if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 || len(newArtifact.Paths[dotnetutils.DotNetCoreCsprojFilesPathType]) == 0 {
			logrus.Errorf("the service directory is missing from the dot net artifact %+v", newArtifact)
			continue
		}
		selectedBuildOption, err := dotnetutils.AskUserForDockerfileType(newArtifact.Name)
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
			quesKey := fmt.Sprintf(common.ConfigServicesDotNetChildProjectsNamesKey, `"`+newArtifact.Name+`"`)
			desc := fmt.Sprintf("For the multi-project Dot Net app '%s', please select all the child projects that should be run as services in the cluster:", newArtifact.Name)
			hints := []string{"deselect any child project that should not be run (example: libraries)"}
			selectedChildProjectNames = qaengine.FetchMultiSelectAnswer(quesKey, desc, hints, selectedChildProjectNames, selectedChildProjectNames, nil)
			if len(selectedChildProjectNames) == 0 {
				return pathMappings, artifactsCreated, fmt.Errorf("user deselected all the child projects of the dot net multi-project app '%s'", newArtifact.Name)
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
			logrus.Debugf("failed to load the IR config from the dot net artifact. Error: %q Artifact: %+v", err, newArtifact)
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

		imageToCopyFrom := common.MakeStringContainerImageNameCompliant(newArtifact.Name + "-" + buildStageC)
		if selectedBuildOption == dotnetutils.NO_BUILD_STAGE {
			imageToCopyFrom = "" // files will be copied from the local file system instead of a builder image
		}

		highestFrameworkVersion := ""
		currPathMappings := []transformertypes.PathMapping{}
		currArtifactsCreated := []transformertypes.Artifact{}

		for _, childProject := range dotNetConfig.ChildProjects {

			// only look at the child modules the user selected

			if !common.IsPresent(selectedChildProjectNames, childProject.Name) {
				continue
			}

			// parse the .csproj file to get the output path
			csProjPath := filepath.Join(serviceDir, childProject.RelCSProjPath)
			configuration, err := dotnetutils.ParseCSProj(csProjPath)
			if err != nil {
				logrus.Errorf("failed to parse the c sharp project file at path %s . Error: %q", csProjPath, err)
				continue
			}

			// have the user select the ports to use for the child project
			selectedPorts := commonqa.GetPortsForService(detectedPorts, common.JoinQASubKeys(`"`+newArtifact.Name+`"`, "childProjects", `"`+childProject.Name+`"`))
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

			// keep track of the highest version among child projects
			if targetFrameworkVersion != "" {
				if highestFrameworkVersion == "" || (semver.IsValid(targetFrameworkVersion) && semver.Compare(highestFrameworkVersion, targetFrameworkVersion) == -1) {
					highestFrameworkVersion = targetFrameworkVersion
					logrus.Infof("Found a higher dot net framework version '%s'", highestFrameworkVersion)
				}
			}

			webConfig := WebTemplateConfig{
				Ports:              selectedPorts,
				BuildStageImageTag: t.getImageTagFromVersion(targetFrameworkVersion),
				IncludeBuildStage:  selectedBuildOption == dotnetutils.BUILD_IN_EVERY_IMAGE,
				IncludeRunStage:    true,
				BuildContainerName: imageToCopyFrom,
				CopyFrom:           common.GetUnixPath(copyFrom),
				RunStageImageTag:   t.getImageTagFromVersion(targetFrameworkVersion),
			}

			// path mapping to generate the Dockerfile for the child project

			dockerfilePath := filepath.Join(common.DefaultSourceDir, relServiceDir, relCSProjDir, common.DefaultDockerfileName)
			currPathMappings = append(currPathMappings, transformertypes.PathMapping{
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
			currArtifactsCreated = append(currArtifactsCreated, dockerfileArtifact, dockerfileServiceArtifact)
		}

		// generate the base image Dockerfile

		if selectedBuildOption == dotnetutils.BUILD_IN_BASE_IMAGE {
			if highestFrameworkVersion == "" {
				highestFrameworkVersion = dotnet.DefaultBaseImageVersion
			}
			logrus.Infof("Using the highest dot net framework version found '%s' for the build stage.", highestFrameworkVersion)
			webConfig := WebTemplateConfig{
				BuildStageImageTag: t.getImageTagFromVersion(highestFrameworkVersion),
				IncludeBuildStage:  true,
				IncludeRunStage:    false,
				BuildContainerName: imageToCopyFrom,
			}

			// path mapping to generate the Dockerfile for the child project

			dockerfilePath := filepath.Join(common.DefaultSourceDir, relServiceDir, common.DefaultDockerfileName+"."+buildStageC)
			currPathMappings = append([]transformertypes.PathMapping{{
				Type:           transformertypes.TemplatePathMappingType,
				SrcPath:        common.DefaultDockerfileName,
				DestPath:       dockerfilePath,
				TemplateConfig: webConfig,
			}}, currPathMappings...)

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
			currArtifactsCreated = append([]transformertypes.Artifact{dockerfileArtifact}, currArtifactsCreated...)
		}
		pathMappings = append(pathMappings, currPathMappings...)
		artifactsCreated = append(artifactsCreated, currArtifactsCreated...)
	}
	return pathMappings, artifactsCreated, nil
}
