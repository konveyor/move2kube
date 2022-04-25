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

package java

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer/dockerfilegenerator/java/gradle"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// GradleAnalyser implements Transformer interface
type GradleAnalyser struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	GradleConfig *GradleYamlConfig
}

// GradleYamlConfig stores the Gradle related information
type GradleYamlConfig struct {
	GradleVersion           string `yaml:"defaultGradleVersion"`
	JavaVersion             string `yaml:"defaultJavaVersion"`
	AppPathInBuildContainer string `yaml:"appPathInBuildContainer"`
}

// GradleBuildDockerfileTemplate defines the information for the build dockerfile template
type GradleBuildDockerfileTemplate struct {
	GradlewPresent     bool
	JavaPackageName    string
	GradleVersion      string
	BuildContainerName string
	GradleProperties   map[string]string
	EnvVariables       map[string]string
}

type gradleInfoT struct {
	Name               string
	Type               transformertypes.ArtifactType
	IsParentBuild      bool
	IsGradlewPresent   bool
	JavaVersion        string
	DeploymentFilePath string
	GradleProperties   map[string]string
	ChildModules       []artifacts.GradleChildModule
	SpringBoot         *artifacts.SpringBootConfig
}

const (
	gradleBuildFileName                     = "build.gradle"
	gradleSettingsFileName                  = "settings.gradle"
	archiveFileC                            = "archiveFile"
	archiveFileNameC                        = "archiveFileName"
	archiveBaseNameC                        = "archiveBaseName"
	archiveAppendixC                        = "archiveAppendix"
	archiveClassifierC                      = "archiveClassifier"
	archiveVersionC                         = "archiveVersion"
	archiveExtensionC                       = "archiveExtension"
	archiveNameC                            = "archiveName"
	archiveBaseNameOldC                     = "baseName"
	archiveAppendixOldC                     = "appendix"
	archiveClassifierOldC                   = "classifier"
	archiveVersionOldC                      = "version"
	archiveExtensionOldC                    = "extension"
	projectPrefixC                          = "project."
	rootProjectPrefixC                      = "rootProject."
	rootProjectNameC                        = rootProjectPrefixC + "name"
	destinationDirectoryC                   = "destinationDirectory"
	destinationDirOldC                      = "destinationDir"
	projectLibsDirNameC                     = "libsDirName"
	buildDirC                               = "buildDir"
	languageVersionC                        = "languageVersion"
	dirFnNameC                              = "layout.buildDirectory.dir"
	projectNameC                            = "name"
	gradleDefaultBuildDirC                  = "build" // https://docs.gradle.org/current/userguide/writing_build_scripts.html#sec:standard_project_properties
	gradleDefaultLibsDirC                   = "libs"  // https://stackoverflow.com/questions/41309257/why-gradle-jars-are-written-in-build-libs
	gradleShadowJarPluginC                  = "com.github.johnrengelman.shadow"
	gradleShadowJarPluginBlockC             = "shadowJar"
	gradleShadowJarPluginDefaultClassifierC = "all" // https://imperceptiblethoughts.com/shadow/configuration/#configuring-output-name
	buildStageC                             = "buildstage"
)

var (
	// gradleSettingsIncludeRegex is used to match lines containing child module/project paths in settings.gradle. Example: include('web', 'api')
	gradleSettingsIncludeRegex = regexp.MustCompile(`^\s*include\(?\s*(?:"[^"]+"|'[^']+')(?:\s*,\s*(?:"[^"]+"|'[^']+'))*\)?$`)
	// gradleIndividualProjectRegex is used to extract individual child module/project paths. Example: get 'web' and 'api' from include('web', 'api')
	gradleIndividualProjectRegex = regexp.MustCompile(`("[^"]+"|'[^']+')`)
)

// Init Initializes the transformer
func (t *GradleAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.GradleConfig = &GradleYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.GradleConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.GradleConfig, err)
		return err
	}
	if t.GradleConfig.JavaVersion == "" {
		t.GradleConfig.JavaVersion = defaultJavaVersion
	}
	if t.GradleConfig.GradleVersion == "" {
		t.GradleConfig.GradleVersion = "7.3"
	}
	if t.GradleConfig.AppPathInBuildContainer == "" {
		t.GradleConfig.AppPathInBuildContainer = defaultAppPathInContainer
	}
	return nil
}

// GetConfig returns the transformer config
func (t *GradleAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *GradleAnalyser) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {

	// look for settings.gradle

	// There will be at most one file path because GetFilesInCurrentDirectory does not check subdirectories.
	gradleSettingsFilePaths, err := common.GetFilesInCurrentDirectory(dir, []string{gradleSettingsFileName}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to look for %s files in the directory %s . Error: %q", gradleSettingsFileName, dir, err)
	}

	// look for a build.gradle as well, in case the root project does not use settings.gradle

	gradleBuildFilePaths, err := common.GetFilesInCurrentDirectory(dir, []string{gradleBuildFileName}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to look for %s files in the directory %s . Error: %q", gradleBuildFileName, dir, err)
	}

	// if both are missing then skip

	if len(gradleSettingsFilePaths) == 0 && len(gradleBuildFilePaths) == 0 {
		return nil, nil
	}

	// start filling in the paths for the service artifact

	paths := map[transformertypes.PathType][]string{artifacts.ServiceRootDirPathType: {dir}}
	if len(gradleSettingsFilePaths) > 0 {
		paths[artifacts.GradleSettingsFilePathType] = gradleSettingsFilePaths
	}
	if len(gradleBuildFilePaths) > 0 {
		paths[artifacts.GradleBuildFilePathType] = gradleBuildFilePaths
	}

	// start filling in the gradle config object for the service artifact

	gradleConfig := artifacts.GradleConfig{}

	// check if gradle wrapper script is present

	if gradleWrapperFilePaths, err := common.GetFilesInCurrentDirectory(dir, []string{"gradlew"}, nil); err == nil && len(gradleWrapperFilePaths) != 0 {
		gradleConfig.IsGradlewPresent = true
	}

	if len(gradleSettingsFilePaths) != 0 {
		// found a settings.gradle

		gradleSettingsFilePath := gradleSettingsFilePaths[0]

		// parse the settings.gradle to get the name and child modules of the root project

		gradleSettings, err := gradle.ParseGardleBuildFile(gradleSettingsFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the gradle settings script at path %s . Error: %q", gradleSettingsFilePath, err)
		}

		// get the name of the root project

		if gradleSettings.Metadata != nil && len(gradleSettings.Metadata[rootProjectNameC]) != 0 {
			gradleConfig.RootProjectName = gradleSettings.Metadata[rootProjectNameC][0]
		}

		// get the child modules of the root project

		gradleConfig.ChildModules, err = getChildModules(gradleSettingsFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get the child modules from the %s file at path %s . Error: %q", gradleSettingsFileName, gradleSettingsFilePath, err)
		}
	}

	// if the root project name is not set in settings.gradle then by default the directory name is used

	if gradleConfig.RootProjectName == "" {
		gradleConfig.RootProjectName = filepath.Base(dir)
	}

	// Get the paths to each of the child module directories (relative to the root project directory).
	// These directories will be ignored during the rest of the planning (directory detect) phase.

	if len(gradleConfig.ChildModules) != 0 {
		gradleConfig.PackagingType = artifacts.PomPackaging // same packaging as maven multi-module project
		for _, childModule := range gradleConfig.ChildModules {
			childBuildScriptPath := filepath.Join(dir, childModule.RelBuildScriptPath)
			paths[artifacts.ServiceDirPathType] = append(paths[artifacts.ServiceDirPathType], filepath.Dir(childBuildScriptPath))
		}
	} else {

		// if there are no child modules then there must be a build.gradle right next to the settings.gradle

		if len(gradleBuildFilePaths) == 0 {
			return nil, fmt.Errorf("expected to find a %s file in the directory %s", gradleBuildFileName, dir)
		}
		paths[artifacts.ServiceDirPathType] = []string{dir}
		gradleConfig.ChildModules = []artifacts.GradleChildModule{{Name: gradleConfig.RootProjectName, RelBuildScriptPath: gradleBuildFileName}}

		// we can get the packaging type (jar/war/ear) by parsing this build.gradle file

		gradleBuildFilePath := gradleBuildFilePaths[0]
		gradleBuild, err := gradle.ParseGardleBuildFile(gradleBuildFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the gradle build script at path %s . Error: %q", gradleBuildFilePath, err)
		}
		gradleConfig.PackagingType = getPackagingFromGradle(&gradleBuild)

		// if nothing was found in the build.gradle file then assume jar packaging

		if gradleConfig.PackagingType == "" {
			gradleConfig.PackagingType = artifacts.JarPackaging
		}
	}

	serviceArtifact := transformertypes.Artifact{
		Name:    gradleConfig.RootProjectName,
		Type:    artifacts.ServiceArtifactType,
		Paths:   paths,
		Configs: map[transformertypes.ConfigType]interface{}{artifacts.GradleConfigType: gradleConfig},
	}
	services := map[string][]transformertypes.Artifact{gradleConfig.RootProjectName: {serviceArtifact}}
	return services, nil
}

// Transform transforms the input artifacts, mostly handles artifacts created during the plan phase.
func (t *GradleAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}

	for _, newArtifact := range newArtifacts {

		// only process service artifacts

		if newArtifact.Type != artifacts.ServiceArtifactType {
			continue
		}

		// only process artifacts that have gradle config

		gradleConfig := artifacts.GradleConfig{}
		if err := newArtifact.GetConfig(artifacts.GradleConfigType, &gradleConfig); err != nil {
			logrus.Debugf("failed to load the gradle config object from the artifact %+v . Error: %q", newArtifact, err)
			continue
		}

		// there must be a settings.gradle or a build.gradle in the root project directory

		if len(newArtifact.Paths[artifacts.GradleSettingsFilePathType]) == 0 && len(newArtifact.Paths[artifacts.GradleBuildFilePathType]) == 0 {
			logrus.Errorf("the artifact doesn't contain any settings.gradle or build.gradle file paths. Artifact: %+v", newArtifact)
			continue
		}

		// there must be a root project directory in the list of paths

		if len(newArtifact.Paths[artifacts.ServiceRootDirPathType]) == 0 {
			logrus.Errorf("the service root directory is missing for the artifact: %+v", newArtifact)
			continue
		}

		// transform a single service artifact (probably created during the plan phase by GradleAnalyser.DirectoryDetect)

		currPathMappings, currCreatedArtifacts, err := t.TransformArtifact(newArtifact, alreadySeenArtifacts, gradleConfig)
		if err != nil {
			logrus.Errorf("failed to transform the artifact: %+v . Error: %q", newArtifact, err)
			continue
		}
		pathMappings = append(pathMappings, currPathMappings...)
		createdArtifacts = append(createdArtifacts, currCreatedArtifacts...)
	}

	return pathMappings, createdArtifacts, nil
}

// TransformArtifact transforms a single artifact.
func (t *GradleAnalyser) TransformArtifact(newArtifact transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact, gradleConfig artifacts.GradleConfig) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}

	selectedBuildOption, err := askUserForDockerfileType(gradleConfig.RootProjectName)
	if err != nil {
		return pathMappings, createdArtifacts, err
	}
	logrus.Debugf("user chose to generate Dockefiles that have '%s'", selectedBuildOption)

	// ask the user which child modules should be run in the K8s cluster

	selectedChildModuleNames := []string{}
	for _, childModule := range gradleConfig.ChildModules {
		selectedChildModuleNames = append(selectedChildModuleNames, childModule.Name)
	}
	if len(selectedChildModuleNames) > 1 {
		quesKey := fmt.Sprintf(common.ConfigServicesChildModulesNamesKey, gradleConfig.RootProjectName)
		desc := fmt.Sprintf("For the multi-module Gradle project '%s', please select all the child modules that should be run as services in the cluster:", gradleConfig.RootProjectName)
		hints := []string{"deselect child modules that should not be run (like libraries)"}
		selectedChildModuleNames = qaengine.FetchMultiSelectAnswer(quesKey, desc, hints, selectedChildModuleNames, selectedChildModuleNames)
		if len(selectedChildModuleNames) == 0 {
			return pathMappings, createdArtifacts, fmt.Errorf("user deselected all the child modules of the gradle multi-module project '%s'", gradleConfig.RootProjectName)
		}
	}

	// have jar/war/ear analyzer transformers generate a Dockerfile with only the run stage for each of the child modules

	lowestJavaVersion := ""
	imageToCopyFrom := gradleConfig.RootProjectName + "-" + buildStageC
	serviceRootDir := newArtifact.Paths[artifacts.ServiceRootDirPathType][0]

	for _, childModule := range gradleConfig.ChildModules {

		// only look at the child modules the user selected

		if !common.IsStringPresent(selectedChildModuleNames, childModule.Name) {
			continue
		}

		// parse the build.gradle of this child module

		childGradleBuildFilePath := filepath.Join(serviceRootDir, childModule.RelBuildScriptPath)
		childGradleBuild, err := gradle.ParseGardleBuildFile(childGradleBuildFilePath)
		if err != nil {
			logrus.Errorf("failed to parse the gradle build script at path %s . Error: %q", childGradleBuildFilePath, err)
			continue
		}

		// get some info about the child module to fill the artifact

		childModuleInfo, err := getInfoFromBuildGradle(&childGradleBuild, childGradleBuildFilePath, gradleConfig, childModule)
		if err != nil {
			logrus.Errorf("failed to get information from the child build.gradle %+v . Error: %q", childGradleBuild, err)
			continue
		}

		// Find the lowest java version among all of the child modules.
		// We will use this java version while doing the build.

		if lowestJavaVersion == "" {
			lowestJavaVersion = childModuleInfo.JavaVersion
		}

		// have the user select which spring boot profiles to use and find a suitable list of ports

		desc := fmt.Sprintf("Select the spring boot profiles for the service '%s' :", childModule.Name)
		hints := []string{"select all the profiles that are applicable"}
		detectedPorts := []int32{}
		envVarsMap := map[string]string{}
		if childModuleInfo.SpringBoot != nil {
			if childModuleInfo.SpringBoot.SpringBootProfiles != nil && len(*childModuleInfo.SpringBoot.SpringBootProfiles) != 0 {
				quesKey := fmt.Sprintf(common.ConfigServicesChildModulesSpringProfilesKey, gradleConfig.RootProjectName, childModule.Name)
				selectedSpringProfiles := qaengine.FetchMultiSelectAnswer(quesKey, desc, hints, *childModuleInfo.SpringBoot.SpringBootProfiles, *childModuleInfo.SpringBoot.SpringBootProfiles)
				for _, selectedSpringProfile := range selectedSpringProfiles {
					detectedPorts = append(detectedPorts, childModuleInfo.SpringBoot.SpringBootProfilePorts[selectedSpringProfile]...)
				}
				envVarsMap["SPRING_PROFILES_ACTIVE"] = strings.Join(selectedSpringProfiles, ",")
			} else {
				detectedPorts = childModuleInfo.SpringBoot.SpringBootProfilePorts[defaultSpringProfile]
			}
		}

		// have the user select the port to use

		selectedPort := commonqa.GetPortForService(detectedPorts, gradleConfig.RootProjectName+common.Delim+"childModules"+common.Delim+childModule.Name)
		if childModuleInfo.SpringBoot != nil {
			envVarsMap["SERVER_PORT"] = fmt.Sprintf("%d", selectedPort)
		} else {
			envVarsMap["PORT"] = fmt.Sprintf("%d", selectedPort)
		}

		// find the path to the artifact (jar/war/ear) which should get copied into the run stage

		relDeploymentFilePath, err := filepath.Rel(serviceRootDir, childModuleInfo.DeploymentFilePath)
		if err != nil {
			logrus.Errorf("failed to make the path %s relative to the base directory %s . Error: %q", childModuleInfo.DeploymentFilePath, serviceRootDir, err)
			continue
		}
		insideContainerDepFilePath := filepath.Join(t.GradleConfig.AppPathInBuildContainer, relDeploymentFilePath)
		childModuleDir := filepath.Dir(childGradleBuildFilePath)
		if selectedBuildOption == NO_BUILD_STAGE {
			imageToCopyFrom = ""
			relDeploymentFilePath, err = filepath.Rel(childModuleDir, childModuleInfo.DeploymentFilePath)
			if err != nil {
				logrus.Errorf("failed to make the jar/war/ear archive path %s relative to the child module service directory %s . Error: %q", childModuleInfo.DeploymentFilePath, childModuleDir, err)
				continue
			}
			insideContainerDepFilePath = relDeploymentFilePath
		}

		// create an artifact that will get picked up by the jar/war/ear analyzer transformers

		runStageArtifact := transformertypes.Artifact{
			Name:  childModule.Name,
			Type:  childModuleInfo.Type,
			Paths: map[transformertypes.PathType][]string{artifacts.ServiceDirPathType: {childModuleDir}},
			Configs: map[transformertypes.ConfigType]interface{}{
				transformertypes.ConfigType(childModuleInfo.Type): artifacts.JarArtifactConfig{
					Port:               selectedPort,
					JavaVersion:        childModuleInfo.JavaVersion,
					BuildContainerName: imageToCopyFrom,
					DeploymentFilePath: insideContainerDepFilePath,
					EnvVariables:       envVarsMap,
				},
				artifacts.ImageNameConfigType: artifacts.ImageName{ImageName: childModule.Name},
				artifacts.ServiceConfigType:   artifacts.ServiceConfig{ServiceName: childModule.Name},
			},
		}
		createdArtifacts = append(createdArtifacts, runStageArtifact)
	}

	if selectedBuildOption == NO_BUILD_STAGE {
		return pathMappings, createdArtifacts, nil
	}

	// Find the java package corresponding to the java version.
	// This will be installed inside the build stage Dockerfile.

	if lowestJavaVersion == "" {
		lowestJavaVersion = t.GradleConfig.JavaVersion
	}
	javaPackageName, err := t.getJavaPackage(lowestJavaVersion)
	if err != nil {
		return pathMappings, createdArtifacts, fmt.Errorf("failed to get the java package for the java version %s . Error: %q", lowestJavaVersion, err)
	}

	// write the build stage Dockerfile template to a temporary file for the pathmapping to pick it up

	dockerfileTemplate, err := t.getDockerfileTemplate()
	if err != nil {
		return pathMappings, createdArtifacts, fmt.Errorf("failed to get the Dockerfile template. Error: %q", err)
	}
	tempDir, err := os.MkdirTemp(t.Env.TempPath, "gradle-transformer-build-*")
	if err != nil {
		return pathMappings, createdArtifacts, fmt.Errorf("failed to create a temporary directory inside the directory %s . Error: %q", t.Env.TempPath, err)
	}
	dockerfileTemplatePath := filepath.Join(tempDir, common.DefaultDockerfileName+".build.template")
	if err := os.WriteFile(dockerfileTemplatePath, []byte(dockerfileTemplate), common.DefaultFilePermission); err != nil {
		return pathMappings, createdArtifacts, fmt.Errorf("failed to write the Dockerfile template to a temporary file at path %s . Error: %q", dockerfileTemplatePath, err)
	}

	// the build stage Dockefile should be placed in the root project directory

	relServiceRootDir, err := filepath.Rel(t.Env.GetEnvironmentSource(), serviceRootDir)
	if err != nil {
		return pathMappings, createdArtifacts, fmt.Errorf("failed to make the service root directory %s relative to the source code directory %s . Error: %q", serviceRootDir, t.Env.GetEnvironmentSource(), err)
	}
	dockerfilePath := filepath.Join(common.DefaultSourceDir, relServiceRootDir, common.DefaultDockerfileName+"."+buildStageC)

	// fill in the Dockerfile template for the build stage and write it out using a pathmapping

	buildStageDockerfilePathMapping := transformertypes.PathMapping{
		Type:     transformertypes.TemplatePathMappingType,
		SrcPath:  dockerfileTemplatePath,
		DestPath: dockerfilePath,
		TemplateConfig: GradleBuildDockerfileTemplate{
			GradlewPresent:     gradleConfig.IsGradlewPresent,
			JavaPackageName:    javaPackageName,
			GradleVersion:      t.GradleConfig.GradleVersion,
			BuildContainerName: imageToCopyFrom,
			GradleProperties:   map[string]string{}, // TODO: gather gradle properties maybe? analog for maven is info.MavenProfiles. https://www.credera.com/insights/gradle-profiles-for-multi-project-spring-boot-applications
			EnvVariables:       map[string]string{}, // TODO: Something about getting env vars from the IR config inside the artifact coming from the cloud foundry transformer?
		},
	}

	if selectedBuildOption == BUILD_IN_EVERY_IMAGE {
		dockerfilePath = filepath.Join(tempDir, common.DefaultDockerfileName+"."+buildStageC)
		buildStageDockerfilePathMapping.DestPath = dockerfilePath
		for _, createdArtifact := range createdArtifacts {
			createdArtifact.Paths[artifacts.BuildContainerFileType] = []string{dockerfilePath}
		}
	} else {

		// make sure the source code directory has been copied over first

		copySourceDirPathMapping := transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}
		pathMappings = append(pathMappings, copySourceDirPathMapping)

		// Tell the other transformers about the build stage Dockerfile we created.
		// That way, the image will get built by the builddockerimages.sh script.

		baseImageDockerfileArtifact := transformertypes.Artifact{
			Name:    imageToCopyFrom,
			Type:    artifacts.DockerfileArtifactType,
			Paths:   map[transformertypes.PathType][]string{artifacts.DockerfilePathType: {dockerfilePath}}, // TODO: should we add the context path as well?
			Configs: map[transformertypes.ConfigType]interface{}{artifacts.ImageNameConfigType: artifacts.ImageName{ImageName: imageToCopyFrom}},
		}
		createdArtifacts = append(createdArtifacts, baseImageDockerfileArtifact)
	}
	pathMappings = append(pathMappings, buildStageDockerfilePathMapping)

	return pathMappings, createdArtifacts, nil
}

// func getInfoFromSettingsGradle(gradleSettingsPtr *gradle.Gradle, gradleSettingsFilePath string, gradleConfig artifacts.GradleConfig) (gradleInfoT, error) {
// 	info := gradleInfoT{
// 		Name:             gradleConfig.RootProjectName,
// 		IsParentBuild:    len(gradleConfig.ChildModules) > 0,
// 		IsGradlewPresent: gradleConfig.IsGradlewPresent,
// 		ChildModules:     gradleConfig.ChildModules,
// 	}
// 	return info, nil
// }

func getInfoFromBuildGradle(gradleBuildPtr *gradle.Gradle, gradleBuildFilePath string, gradleConfig artifacts.GradleConfig, childModule artifacts.GradleChildModule) (gradleInfoT, error) {
	info := gradleInfoT{
		Name: childModule.Name,
	} // TODO: multiple levels of sub-projects
	gradleBuildFileDir := filepath.Dir(gradleBuildFilePath)
	if ps, err := common.GetFilesInCurrentDirectory(gradleBuildFileDir, []string{"gradlew"}, nil); err != nil {
		return info, fmt.Errorf("failed to look for gradle wrapper files in the child module service directory %s . Error: %q", gradleBuildFileDir, err)
	} else if len(ps) > 0 {
		info.IsGradlewPresent = true
	}
	if gradleBuildPtr != nil {
		gradleBuild := *gradleBuildPtr
		info.SpringBoot = getSpringBootConfigFromGradle(gradleBuildFilePath, gradleBuildPtr, nil)
		packType := getPackagingFromGradle(gradleBuildPtr)
		if packType == "" {
			packType = artifacts.JarPackaging
		}
		artType, err := packagingToArtifactType(packType)
		if err != nil {
			return info, fmt.Errorf("failed to convert the packaging type %s to a valid artifact type. Error: %q", gradleConfig.PackagingType, err)
		}
		info.Type = artType
		info.JavaVersion = getJavaVersionFromGradle(gradleBuildPtr)
		deploymentFilePath, err := getDeploymentFilePathFromGradle(gradleBuildPtr, gradleBuildFilePath, filepath.Dir(gradleBuildFilePath), gradleConfig, packType)
		if err != nil {
			return info, fmt.Errorf("failed to get the output path for the gradle build script %+v . Error: %q", gradleBuild, err)
		}
		info.DeploymentFilePath = deploymentFilePath
	}
	return info, nil
}

func getPackagingFromGradle(gradleBuild *gradle.Gradle) artifacts.JavaPackaging {
	if gradleBuild == nil {
		return ""
	}
	pluginIds := gradleBuild.GetPluginIDs()
	if common.IsStringPresent(pluginIds, string(artifacts.JarPackaging)) {
		return artifacts.JarPackaging
	} else if common.IsStringPresent(pluginIds, string(artifacts.EarPackaging)) {
		return artifacts.EarPackaging
	} else if common.IsStringPresent(pluginIds, string(artifacts.WarPackaging)) {
		return artifacts.WarPackaging
	}
	return ""
}

func getSpringBootConfigFromGradle(buildFilePath string, gradleBuild, parentGradleBuild *gradle.Gradle) *artifacts.SpringBootConfig {
	if gradleBuild == nil {
		logrus.Errorf("got a nil gradle build script")
		return nil
	}
	buildFileDir := filepath.Dir(buildFilePath)
	getSpringBootConfig := func(dependency gradle.GradleDependency) *artifacts.SpringBootConfig {
		if dependency.Group != springBootGroup {
			return nil
		}
		springAppName, springProfiles, profilePorts := getSpringBootAppNameProfilesAndPortsFromDir(buildFileDir)
		springConfig := &artifacts.SpringBootConfig{
			SpringBootVersion:      dependency.Version,
			SpringBootAppName:      springAppName,
			SpringBootProfilePorts: profilePorts,
		}
		if len(springProfiles) != 0 {
			springConfig.SpringBootProfiles = &springProfiles
		}
		return springConfig
	}
	// look for spring boot
	for _, dependency := range gradleBuild.Dependencies {
		if springConfig := getSpringBootConfig(dependency); springConfig != nil {
			return springConfig
		}
	}
	return nil
}

func (t *GradleAnalyser) getDockerfileTemplate() (string, error) {
	// TODO: see if we can cache gradle dependencies similar to https://stackoverflow.com/a/37442191
	licenseFilePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license")
	license, err := os.ReadFile(licenseFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read the Dockerfile license file at path %s . Error: %q", licenseFilePath, err)
	}
	gradleBuildTemplatePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.gradle-build")
	gradleBuildTemplate, err := os.ReadFile(gradleBuildTemplatePath)
	if err != nil {
		return string(license), fmt.Errorf("failed to read the Dockerfile Gradle build template file at path %s . Error: %q", gradleBuildTemplatePath, err)
	}
	return string(license) + "\n" + string(gradleBuildTemplate), nil
}

func (t *GradleAnalyser) getJavaPackage(javaVersion string) (string, error) {
	javaVersionToPackageMappingFilePath := filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath)
	return getJavaPackage(javaVersionToPackageMappingFilePath, javaVersion)
}

// getJavaVersionFromGradle finds the java version from a gradle build script (build.gradle).
func getJavaVersionFromGradle(build *gradle.Gradle) string {
	if build == nil {
		return ""
	}
	// https://docs.gradle.org/current/userguide/java_plugin.html#sec:java-extension
	if gb, ok := build.Blocks["java"]; ok {
		if gb, ok := gb.Blocks["toolchain"]; ok {
			if len(gb.Metadata[languageVersionC]) > 0 {
				ss := gradle.GetSingleArgumentFromFuntionCall(gb.Metadata[languageVersionC][0], "JavaLanguageVersion.of")
				gradleJavaVersion, err := strconv.Atoi(ss)
				if err != nil {
					logrus.Errorf("failed to parse the string '%s' as an integer. Error: %q", ss, err)
					return ""
				}
				if gradleJavaVersion < 10 {
					return "1." + fmt.Sprintf("%d", gradleJavaVersion)
				}
				return fmt.Sprintf("%d", gradleJavaVersion)
			}
		}
	}
	return ""
}

func getDeploymentFilePathFromGradle(gradleBuild *gradle.Gradle, buildScriptPath, serviceDir string, gradleConfig artifacts.GradleConfig, packagingType artifacts.JavaPackaging) (string, error) {
	if gradleBuild == nil {
		return "", fmt.Errorf("the given gradle build script is nil")
	}
	archivePath := ""
	archiveName := ""
	destinationPath := ""
	archiveBaseName := ""
	archiveAppendix := ""
	archiveVersion := ""
	archiveClassifier := ""
	archiveExtension := ""
	joinIntoName := func() string {
		ans := archiveBaseName
		if archiveAppendix != "" {
			ans += "-" + archiveAppendix
		}
		if archiveVersion != "" {
			ans += "-" + archiveVersion
		}
		if archiveClassifier != "" {
			ans += "-" + archiveClassifier
		}
		if archiveExtension != "" {
			ans += "." + archiveExtension
		}
		return ans
	}

	updateArchiveNameFromJarBlock := func(gb gradle.Gradle) {
		// https://docs.gradle.org/current/dsl/org.gradle.api.tasks.bundling.Jar.html#org.gradle.api.tasks.bundling.Jar:archiveFile
		if len(gb.Metadata[archiveFileC]) > 0 {
			archivePath = gb.Metadata[archiveFileC][0]
		}
		// https://docs.gradle.org/current/dsl/org.gradle.api.tasks.bundling.Jar.html#org.gradle.api.tasks.bundling.Jar:destinationDirectory
		if len(gb.Metadata[destinationDirectoryC]) > 0 {
			destinationPath = gradle.GetSingleArgumentFromFuntionCall(gb.Metadata[destinationDirectoryC][0], dirFnNameC)
		} else if len(gb.Metadata[destinationDirOldC]) > 0 {
			destinationPath = gradle.GetSingleArgumentFromFuntionCall(gb.Metadata[destinationDirOldC][0], dirFnNameC)
		}
		// https://docs.gradle.org/current/dsl/org.gradle.api.tasks.bundling.Jar.html#org.gradle.api.tasks.bundling.Jar:archiveFileName
		if len(gb.Metadata[archiveFileNameC]) > 0 {
			archiveName = gb.Metadata[archiveFileNameC][0]
		} else if len(gb.Metadata[archiveNameC]) > 0 {
			archiveName = gb.Metadata[archiveNameC][0]
		}
		if archiveName == "" {
			// look for ${archiveBaseName}-${archiveAppendix}-${archiveVersion}-${archiveClassifier}.${archiveExtension}
			// archiveBaseName
			if len(gb.Metadata[archiveBaseNameC]) > 0 {
				archiveBaseName = gb.Metadata[archiveBaseNameC][0]
			} else if len(gb.Metadata[archiveBaseNameOldC]) > 0 {
				archiveBaseName = gb.Metadata[archiveBaseNameOldC][0]
			} else if len(gradleBuild.Metadata[projectNameC]) > 0 {
				// TODO: project.name is a read-only property
				// https://docs.gradle.org/current/dsl/org.gradle.api.Project.html#org.gradle.api.Project:name
				// https://stackoverflow.com/a/55690608
				archiveBaseName = gradleBuild.Metadata[projectNameC][0]
			} else {
				archiveBaseName = filepath.Base(filepath.Dir(buildScriptPath))
			}
			// archiveAppendix
			if len(gb.Metadata[archiveAppendixC]) > 0 {
				archiveAppendix = gb.Metadata[archiveAppendixC][0]
			} else if len(gb.Metadata[archiveAppendixOldC]) > 0 {
				archiveAppendix = gb.Metadata[archiveAppendixOldC][0]
			} else if len(gradleBuild.Metadata[projectPrefixC+archiveAppendixOldC]) > 0 {
				archiveAppendix = gradleBuild.Metadata[projectPrefixC+archiveAppendixOldC][0]
			} else if len(gradleBuild.Metadata[archiveAppendixOldC]) > 0 {
				archiveAppendix = gradleBuild.Metadata[archiveAppendixOldC][0]
			}
			// archiveVersion
			if len(gb.Metadata[archiveVersionC]) > 0 {
				archiveVersion = gb.Metadata[archiveVersionC][0]
			} else if len(gb.Metadata[archiveVersionOldC]) > 0 {
				archiveVersion = gb.Metadata[archiveVersionOldC][0]
			} else if len(gradleBuild.Metadata[projectPrefixC+archiveVersionOldC]) > 0 {
				archiveVersion = gradleBuild.Metadata[projectPrefixC+archiveVersionOldC][0]
			} else if len(gradleBuild.Metadata[archiveVersionOldC]) > 0 {
				archiveVersion = gradleBuild.Metadata[archiveVersionOldC][0]
			}
			// archiveClassifier
			if len(gb.Metadata[archiveClassifierC]) > 0 {
				archiveClassifier = gb.Metadata[archiveClassifierC][0]
			} else if len(gb.Metadata[archiveClassifierOldC]) > 0 {
				archiveClassifier = gb.Metadata[archiveClassifierOldC][0]
			} else if len(gradleBuild.Metadata[projectPrefixC+archiveClassifierOldC]) > 0 {
				archiveClassifier = gradleBuild.Metadata[projectPrefixC+archiveClassifierOldC][0]
			} else if len(gradleBuild.Metadata[archiveClassifierOldC]) > 0 {
				archiveClassifier = gradleBuild.Metadata[archiveClassifierOldC][0]
			}
			// archiveExtension
			if len(gb.Metadata[archiveExtensionC]) > 0 {
				archiveExtension = gb.Metadata[archiveExtensionC][0]
			} else if len(gb.Metadata[archiveExtensionOldC]) > 0 {
				archiveExtension = gb.Metadata[archiveExtensionOldC][0]
			} else {
				archiveExtension = string(packagingType)
			}
		}
	}

	// first we look in the top level for the version

	// archiveBaseName
	// project.name is a read-only property
	// https://docs.gradle.org/current/dsl/org.gradle.api.Project.html#org.gradle.api.Project:name
	// https://stackoverflow.com/a/55690608
	archiveBaseName = filepath.Base(filepath.Dir(buildScriptPath))
	// archiveVersion
	if len(gradleBuild.Metadata[projectPrefixC+archiveVersionOldC]) > 0 {
		archiveVersion = gradleBuild.Metadata[projectPrefixC+archiveVersionOldC][0]
	} else if len(gradleBuild.Metadata[archiveVersionOldC]) > 0 {
		archiveVersion = gradleBuild.Metadata[archiveVersionOldC][0]
	}
	// archiveExtension
	archiveExtension = string(packagingType)

	// second we look in the shadowJar block to override the archive name

	if common.IsStringPresent(gradleBuild.GetPluginIDs(), gradleShadowJarPluginC) {
		archiveClassifier = gradleShadowJarPluginDefaultClassifierC
		if gb2, ok := gradleBuild.Blocks[gradleShadowJarPluginBlockC]; ok {
			updateArchiveNameFromJarBlock(gb2)
		}
	}
	if archivePath != "" {
		return filepath.Join(serviceDir, archivePath), nil
	}

	// third we look in the jar/war/ear block to override the archive name

	if gb1, ok := gradleBuild.Blocks[string(packagingType)]; ok {
		updateArchiveNameFromJarBlock(gb1)
	}
	if archivePath != "" {
		return filepath.Join(serviceDir, archivePath), nil
	}

	// get the archiveName by combining the different parts
	archiveName = joinIntoName()

	if destinationPath != "" {
		return filepath.Join(serviceDir, destinationPath, archiveName), nil
	}
	// libs directory where the archives are genearted by the jar/war/ear plugin
	if len(gradleBuild.Metadata[projectPrefixC+projectLibsDirNameC]) > 0 {
		destinationPath = gradleBuild.Metadata[projectPrefixC+projectLibsDirNameC][0]
	} else if len(gradleBuild.Metadata[projectLibsDirNameC]) > 0 {
		destinationPath = gradleBuild.Metadata[projectLibsDirNameC][0]
	} else {
		destinationPath = gradleDefaultLibsDirC
	}
	// find the build output directory
	// https://docs.gradle.org/current/dsl/org.gradle.api.Project.html#org.gradle.api.Project:buildDir
	buildDir := ""
	if len(gradleBuild.Metadata[projectPrefixC+buildDirC]) > 0 {
		buildDir = gradleBuild.Metadata[projectPrefixC+buildDirC][0]
	} else if len(gradleBuild.Metadata[buildDirC]) > 0 {
		buildDir = gradleBuild.Metadata[buildDirC][0]
	} else {
		buildDir = gradleDefaultBuildDirC
	}
	return filepath.Join(serviceDir, buildDir, destinationPath, archiveName), nil
}

func getChildModules(gradleSettingsFilePath string) ([]artifacts.GradleChildModule, error) {
	gradleSettingsFile, err := os.Open(gradleSettingsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open the settings.gradle file at path %s . Error: %q", gradleSettingsFilePath, err)
	}
	scanner := bufio.NewScanner(gradleSettingsFile)
	scanner.Split(bufio.ScanLines)
	childModuleRelPaths := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if gradleSettingsIncludeRegex.MatchString(line) {
			resultss := gradleIndividualProjectRegex.FindAllStringSubmatch(line, -1)
			for _, results := range resultss {
				if len(results) == 0 {
					continue
				}
				quotedChildModuleRelPath := results[0]
				if len(quotedChildModuleRelPath) < 3 {
					logrus.Debugf("invalid or empty child module path. Actual: %s\n", quotedChildModuleRelPath)
					continue
				}
				childModuleRelPath := quotedChildModuleRelPath[1 : len(quotedChildModuleRelPath)-1]
				childModuleRelPaths = append(childModuleRelPaths, childModuleRelPath)
			}
		}
	}
	childModules := []artifacts.GradleChildModule{}
	for _, childModuleRelPath := range childModuleRelPaths {
		childModuleRelPath = strings.Replace(childModuleRelPath, ":", string(os.PathSeparator), -1)
		if filepath.IsAbs(childModuleRelPath) {
			childModuleRelPath = childModuleRelPath[1:]
		}
		name := filepath.Base(childModuleRelPath)
		relPath := filepath.Join(childModuleRelPath, gradleBuildFileName)
		childModules = append(childModules, artifacts.GradleChildModule{Name: name, RelBuildScriptPath: relPath})
	}
	if err := scanner.Err(); err != nil {
		return childModules, fmt.Errorf("failed to read the gradle settings script at path %s line by line. Error: %q", gradleSettingsFilePath, err)
	}
	return childModules, nil
}
