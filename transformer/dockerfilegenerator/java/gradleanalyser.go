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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer/dockerfilegenerator/java/gradle"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	gradleBuildFileName = "gradle.build"
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
	JavaPackageName string
}

// Init Initializes the transformer
func (t *GradleAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.GradleConfig = &GradleYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, &t.GradleConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.GradleConfig, err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *GradleAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *GradleAnalyser) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	services = map[string][]transformertypes.Artifact{}
	gradleFilePaths, err := common.GetFilesInCurrentDirectory(dir, []string{gradleBuildFileName}, nil)
	if err != nil {
		logrus.Errorf("Error while parsing directory %s for Gradle file : %s", dir, err)
		return nil, err
	}
	if len(gradleFilePaths) == 0 {
		return nil, nil
	}
	gradleBuild, err := gradle.ParseGardleBuildFile(gradleFilePaths[0])
	if err != nil {
		logrus.Errorf("Error while parsing gradle build file : %s", err)
		return nil, err
	}
	ct := transformertypes.Artifact{
		Configs: map[transformertypes.ConfigType]interface{}{},
		Paths: map[transformertypes.PathType][]string{
			artifacts.GradleBuildFilePathType: {filepath.Join(dir, gradleBuildFileName)},
			artifacts.ProjectPathPathType:     {dir},
		},
	}
	gc := artifacts.GradleConfig{}
	gc.ArtifactType = JarPackaging
	if _, ok := gradleBuild.Metadata[EarPackaging]; ok {
		gc.ArtifactType = artifacts.EarArtifactType
	} else if _, ok := gradleBuild.Metadata[WarPackaging]; ok {
		gc.ArtifactType = artifacts.WarArtifactType
	}
	ct.Configs[artifacts.GradleConfigType] = gc
	appName := ""
	for _, dependency := range gradleBuild.Dependencies {
		if dependency.Group == springbootGroup {
			sbc := artifacts.SpringBootConfig{}
			sbps := []string{}
			appName, sbps = getSpringBootAppNameAndProfilesFromDir(dir)
			sbc.SpringBootAppName = appName
			if len(sbps) != 0 {
				sbc.SpringBootProfiles = &sbps
			}
			if dependency.Version != "" {
				sbc.SpringBootVersion = dependency.Version
			}
			ct.Configs[artifacts.SpringBootConfigType] = sbc
			break
		}
	}
	services[appName] = append(services[appName], ct)
	return
}

// Transform transforms the artifacts
func (t *GradleAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		javaVersion := ""
		var pom Gradle.Pom
		if len(a.Paths[artifacts.GradlePomPathType]) == 0 {
			err := fmt.Errorf("unable to find pom for %s", a.Name)
			logrus.Errorf("%s", err)
			return nil, nil, err
		}
		err := pom.Load(a.Paths[artifacts.GradlePomPathType][0])
		if err != nil {
			logrus.Errorf("Unable to load pom for %s : %s", a.Name, err)
		}
		if _, ok := a.Configs[artifacts.SpringBootConfigType]; ok {
			jv, err := pom.GetProperty("java.version")
			if err == nil {
				javaVersion = jv
			}
		}
		if javaVersion == "" {
			jv, err := pom.GetProperty("Gradle.compiler.target")
			if err == nil {
				javaVersion = jv
			}
		}
		if javaVersion == "" && pom.Build != nil && pom.Build.Plugins != nil {
			for _, dep := range *pom.Build.Plugins {
				if dep.ArtifactID == "Gradle-compiler-plugin" {
					javaVersion = dep.Configuration.Target
				}
			}
		}
		if javaVersion == "" {
			javaVersion = t.GradleConfig.JavaVersion
		}
		GradleConfig := artifacts.GradleConfig{}
		err = a.GetConfig(artifacts.GradleConfigType, &GradleConfig)
		if err != nil {
			logrus.Debugf("Unable to load Gradle config object: %s", err)
		}
		ir := irtypes.IR{}
		irPresent := true
		err = a.GetConfig(irtypes.IRConfigType, &ir)
		if err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T : %s", ir, err)
		}
		defaultProfiles := []string{}
		if pom.Profiles != nil {
			for _, profile := range *pom.Profiles {
				if profile.Activation != nil && profile.Activation.ActiveByDefault && common.IsStringPresent(GradleConfig.GradleProfiles, profile.ID) {
					defaultProfiles = append(defaultProfiles, profile.ID)
				}
			}
		}
		if len(defaultProfiles) == 0 {
			defaultProfiles = GradleConfig.GradleProfiles
		}
		selectedGradleProfiles := qaengine.FetchMultiSelectAnswer(
			common.ConfigServicesKey+common.Delim+a.Name+common.Delim+common.ConfigActiveGradleProfilesForServiceKeySegment,
			fmt.Sprintf("Choose the Gradle profile to be used for the service %s", a.Name),
			[]string{fmt.Sprintf("Selected Gradle profiles will be used for setting configuration for the service %s", a.Name)},
			defaultProfiles,
			GradleConfig.GradleProfiles,
		)
		if len(selectedGradleProfiles) == 0 {
			logrus.Debugf("No Gradle profiles selected")
		}
		classifier := ""
		if pom.Build != nil && pom.Build.Plugins != nil {
			// Iterate over existing plugins
			for _, GradlePlugin := range *pom.Build.Plugins {
				// Check if spring-boot-Gradle-plugin is present
				if GradlePlugin.ArtifactID != "spring-boot-Gradle-plugin" || GradlePlugin.Executions == nil {
					continue
				}
				isRepackageEnabled := false
				for _, GradlePluginExecution := range *GradlePlugin.Executions {
					if GradlePluginExecution.Goals == nil {
						continue
					}
					if common.IsStringPresent(*GradlePluginExecution.Goals, "repackage") {
						isRepackageEnabled = true
					}
				}
				if !isRepackageEnabled {
					continue
				}
				if GradlePlugin.Configuration.ConfigurationProfiles == nil || len(*GradlePlugin.Configuration.ConfigurationProfiles) == 0 {
					classifier = GradlePlugin.Configuration.Classifier
					break
				}
				for _, configProfile := range *GradlePlugin.Configuration.ConfigurationProfiles {
					// we check if any of these profiles is contained in the list of profiles
					// selected by the user
					// if yes, we look for the classifier property of this plugin and
					// assign it to the classifier variable
					if common.IsStringPresent(selectedGradleProfiles, configProfile) {
						classifier = GradlePlugin.Configuration.Classifier
						break
					}
				}
				break
			}
			logrus.Debugf("classifier: %s", classifier)
		}
		// Springboot profiles handling
		// We collect the springboot profiles from the current service
		springbootConfig := artifacts.SpringBootConfig{}
		err = a.GetConfig(artifacts.SpringBootConfigType, &springbootConfig)
		if err != nil {
			logrus.Debugf("Unable to load springboot config object: %s", err)
		}
		// if there are profiles, we ask the user to select
		springBootProfilesFlattened := ""
		if springbootConfig.SpringBootProfiles != nil && len(*springbootConfig.SpringBootProfiles) > 0 {
			selectedSpringBootProfiles := qaengine.FetchMultiSelectAnswer(
				common.ConfigServicesKey+common.Delim+a.Name+common.Delim+common.ConfigActiveSpringBootProfilesForServiceKeySegment,
				fmt.Sprintf("Choose Springboot profiles to be used for the service %s", a.Name),
				[]string{fmt.Sprintf("Selected Springboot profiles will be used for setting configuration for the service %s", a.Name)},
				*springbootConfig.SpringBootProfiles,
				*springbootConfig.SpringBootProfiles,
			)
			if len(selectedSpringBootProfiles) != 0 {
				// we flatten the list of springboot profiles for passing it as env var
				springBootProfilesFlattened = strings.Join(selectedSpringBootProfiles, ",")
			} else {
				logrus.Debugf("No springboot profiles selected")
			}
		}
		// Dockerfile Env variables storage
		envVariablesMap := map[string]string{}
		if springBootProfilesFlattened != "" {
			// we add to the map of env vars
			envVariablesMap["SPRING_PROFILES_ACTIVE"] = springBootProfilesFlattened
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(a.Name)
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), javaVersion)
		if err != nil {
			logrus.Errorf("Unable to find mapping version for java version %s : %s", javaVersion, err)
			javaPackage = "java-1.8.0-openjdk-devel"
		}
		license, err := os.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license"))
		if err != nil {
			logrus.Errorf("Unable to read Dockerfile license template : %s", err)
		}
		GradleBuild, err := os.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.Gradle-build"))
		if err != nil {
			logrus.Errorf("Unable to read Dockerfile license template : %s", err)
		}
		tempDir := filepath.Join(t.Env.TempPath, a.Name)
		os.MkdirAll(tempDir, common.DefaultDirectoryPermission)
		dockerfileTemplate := filepath.Join(tempDir, "Dockerfile.template")
		template := string(license) + "\n" + string(GradleBuild)
		err = os.WriteFile(dockerfileTemplate, []byte(template), common.DefaultFilePermission)
		if err != nil {
			logrus.Errorf("Could not write the generated Build Dockerfile template: %s", err)
		}
		buildDockerfile := filepath.Join(tempDir, "Dockerfile.build")
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.TemplatePathMappingType,
			SrcPath:  dockerfileTemplate,
			DestPath: buildDockerfile,
			TemplateConfig: GradleBuildDockerfileTemplate{
				JavaPackageName: javaPackage,
			},
		})
		deploymentFileName := pom.ArtifactID + "-" + pom.Version
		if classifier != "" {
			deploymentFileName = deploymentFileName + "-" + classifier
		}
		var newArtifact transformertypes.Artifact
		switch pom.Packaging {
		case WarPackaging:
			newArtifact = transformertypes.Artifact{
				Name:     a.Name,
				Artifact: artifacts.WarArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.WarConfigType: artifacts.WarArtifactConfig{
						DeploymentFile:                    deploymentFileName + ".war",
						JavaVersion:                       javaVersion,
						BuildContainerName:                common.DefaultBuildContainerName,
						DeploymentFileDirInBuildContainer: filepath.Join(defaultAppPathInContainer, "target"),
						EnvVariables:                      envVariablesMap,
					},
				},
			}
		case EarPackaging:
			newArtifact = transformertypes.Artifact{
				Name:     a.Name,
				Artifact: artifacts.EarArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.EarConfigType: artifacts.EarArtifactConfig{
						DeploymentFile:                    deploymentFileName + ".ear",
						JavaVersion:                       javaVersion,
						BuildContainerName:                common.DefaultBuildContainerName,
						DeploymentFileDirInBuildContainer: filepath.Join(defaultAppPathInContainer, "target"),
						EnvVariables:                      envVariablesMap,
					},
				},
			}
		default:
			ports := ir.GetAllServicePorts()
			if len(ports) == 0 {
				ports = append(ports, common.DefaultServicePort)
			}
			port := commonqa.GetPortForService(ports, a.Name)
			if springBootProfilesFlattened != "" {
				envVariablesMap["SERVER_PORT"] = fmt.Sprintf("%d", port)
			} else {
				envVariablesMap["PORT"] = fmt.Sprintf("%d", port)
			}
			newArtifact = transformertypes.Artifact{
				Name:     a.Name,
				Artifact: artifacts.JarArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.JarConfigType: artifacts.JarArtifactConfig{
						DeploymentFile:                    deploymentFileName + ".jar",
						JavaVersion:                       javaVersion,
						BuildContainerName:                common.DefaultBuildContainerName,
						DeploymentFileDirInBuildContainer: filepath.Join(defaultAppPathInContainer, "target"),
						EnvVariables:                      envVariablesMap,
						Port:                              common.DefaultServicePort,
					},
				},
			}
		}
		if irPresent {
			newArtifact.Configs[irtypes.IRConfigType] = ir
		}
		if newArtifact.Configs == nil {
			newArtifact.Configs = map[string]interface{}{}
		}
		newArtifact.Configs[artifacts.ImageNameConfigType] = sImageName
		newArtifact.Configs[artifacts.ServiceConfigType] = sConfig
		if newArtifact.Paths == nil {
			newArtifact.Paths = map[string][]string{}
		}
		newArtifact.Paths[artifacts.BuildContainerFileType] = []string{buildDockerfile}
		newArtifact.Paths[artifacts.ProjectPathPathType] = a.Paths[artifacts.ProjectPathPathType]
		createdArtifacts = append(createdArtifacts, newArtifact)
	}
	return pathMappings, createdArtifacts, nil
}
