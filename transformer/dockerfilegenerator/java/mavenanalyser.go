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
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/konveyor/move2kube/types/source/maven"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	defaultMavenVersion = "3.8.4"
)

// MavenAnalyser implements Transformer interface
type MavenAnalyser struct {
	Config      transformertypes.Transformer
	Env         *environment.Environment
	MavenConfig *MavenYamlConfig
}

// MavenYamlConfig stores the maven related information
type MavenYamlConfig struct {
	MavenVersion            string `yaml:"defaultMavenVersion"`
	JavaVersion             string `yaml:"defaultJavaVersion"`
	AppPathInBuildContainer string `yaml:"appPathInBuildContainer"`
}

// MavenBuildDockerfileTemplate defines the information for the build dockerfile template
type MavenBuildDockerfileTemplate struct {
	JavaPackageName string
	MavenVersion    string
	EnvVariables    map[string]string
	MavenProfiles   []string
	MvnwPresent     bool
}

// Init Initializes the transformer
func (t *MavenAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.MavenConfig = &MavenYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.MavenConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.MavenConfig, err)
		return err
	}
	if t.MavenConfig.MavenVersion == "" {
		t.MavenConfig.MavenVersion = defaultMavenVersion
	}
	if t.MavenConfig.JavaVersion == "" {
		t.MavenConfig.JavaVersion = defaultJavaVersion
	}
	if t.MavenConfig.AppPathInBuildContainer == "" {
		t.MavenConfig.AppPathInBuildContainer = defaultAppPathInContainer
	}
	return nil
}

// GetConfig returns the transformer config
func (t *MavenAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *MavenAnalyser) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	mavenFilePaths, err := common.GetFilesInCurrentDirectory(dir, []string{maven.PomXMLFileName}, nil)
	if err != nil {
		logrus.Errorf("failed to look for maven pom.xml files in the directory %s . Error: %q", dir, err)
		return nil, err
	}
	if len(mavenFilePaths) == 0 {
		return nil, nil
	}
	logrus.Debugf("found %d pom.xml files in the directory %s . files: %+v", len(mavenFilePaths), dir, mavenFilePaths)
	pom := &maven.Pom{}
	// TODO: what about the other mavenFilePaths?
	if err := pom.Load(mavenFilePaths[0]); err != nil {
		logrus.Errorf("Unable to unmarshal pom file (%s): %s", mavenFilePaths[0], err)
		return nil, err
	}
	if pom.Packaging == string(artifacts.PomPackaging) {
		logrus.Debugf("Parent pom detected (%s). Ignoring.", mavenFilePaths[0])
		return nil, nil
	}
	appName := pom.ArtifactID
	mavenProfiles := []string{}
	if pom.Profiles != nil {
		for _, profile := range *pom.Profiles {
			mavenProfiles = append(mavenProfiles, profile.ID)
		}
	}
	mavenArtifact := transformertypes.Artifact{
		Configs: map[transformertypes.ConfigType]interface{}{},
		Paths: map[transformertypes.PathType][]string{
			artifacts.MavenPomPathType:   {filepath.Join(dir, maven.PomXMLFileName)},
			artifacts.ServiceDirPathType: {dir},
		},
	}
	mavenConfig := artifacts.MavenConfig{}
	mavenConfig.MavenAppName = pom.ArtifactID
	mavenConfig.ArtifactType = artifacts.JavaPackaging(pom.Packaging)
	if mavenConfig.ArtifactType == "" {
		mavenConfig.ArtifactType = artifacts.JarPackaging
	}
	if len(mavenProfiles) != 0 {
		mavenConfig.MavenProfiles = mavenProfiles
	}
	// look for maven wrapper
	if mavenWrapperFilePaths, err := common.GetFilesInCurrentDirectory(dir, []string{"mvnw"}, nil); err != nil {
		logrus.Errorf("Error while parsing directory %s for mvnw file : %s", dir, err)
		return nil, err
	} else if len(mavenWrapperFilePaths) > 0 {
		mavenConfig.MvnwPresent = true
	}
	mavenArtifact.Configs[artifacts.MavenConfigType] = mavenConfig
	getSpringBootConfig := func(dependency maven.Dependency, pomDir string) *artifacts.SpringBootConfig {
		if dependency.GroupID != springBootGroup {
			return nil
		}
		springConfig := &artifacts.SpringBootConfig{}
		springConfig.SpringBootVersion = dependency.Version
		springAppName, springProfiles := getSpringBootAppNameAndProfilesFromDir(pomDir)
		springConfig.SpringBootAppName = springAppName
		if len(springProfiles) != 0 {
			springConfig.SpringBootProfiles = &springProfiles
		}
		return springConfig
	}
	// look for spring boot
	isSpringBoot := false
	if pom.Dependencies != nil {
		for _, dependency := range *pom.Dependencies {
			if springConfig := getSpringBootConfig(dependency, dir); springConfig != nil {
				isSpringBoot = true
				mavenArtifact.Configs[artifacts.SpringBootConfigType] = springConfig
				break
			}
		}
	}
	if !isSpringBoot {
		if pom.DependencyManagement != nil && pom.DependencyManagement.Dependencies != nil {
			for _, dependency := range *pom.DependencyManagement.Dependencies {
				if springConfig := getSpringBootConfig(dependency, dir); springConfig != nil {
					isSpringBoot = true
					mavenArtifact.Configs[artifacts.SpringBootConfigType] = springConfig
					break
				}
			}
		}
	}
	if !isSpringBoot {
		if pom.Parent != nil {
			// parse parent pom and look for spring boot
			childPomDir := dir
			parentPomDir := filepath.Join(childPomDir, "..")
			parentPomPath := filepath.Join(parentPomDir, "pom.xml")
			if pom.Parent.RelativePath != "" {
				parentPomDir = filepath.Join(childPomDir, pom.Parent.RelativePath)
				parentPomPath = filepath.Join(parentPomDir, "pom.xml")
				if filepath.Ext(parentPomDir) == ".xml" {
					parentPomPath = parentPomDir
					parentPomDir = filepath.Dir(parentPomDir)
				}
			}
			parentPom := &maven.Pom{}
			// TODO: handle more than one level of parent poms. Recurse up?
			if err := parentPom.Load(parentPomPath); err != nil {
				logrus.Errorf("Unable to unmarshal pom file (%s): %s", mavenFilePaths[0], err)
				return nil, err
			}
			if parentPom.Dependencies != nil {
				for _, dependency := range *parentPom.Dependencies {
					if springConfig := getSpringBootConfig(dependency, parentPomDir); springConfig != nil {
						isSpringBoot = true
						mavenArtifact.Configs[artifacts.SpringBootConfigType] = springConfig
						break
					}
				}
			}
			if !isSpringBoot {
				if parentPom.DependencyManagement != nil && parentPom.DependencyManagement.Dependencies != nil {
					for _, dependency := range *parentPom.DependencyManagement.Dependencies {
						if springConfig := getSpringBootConfig(dependency, parentPomDir); springConfig != nil {
							isSpringBoot = true
							mavenArtifact.Configs[artifacts.SpringBootConfigType] = springConfig
							break
						}
					}
				}
			}
		}
	}
	services := map[string][]transformertypes.Artifact{appName: {mavenArtifact}}
	return services, err
}

// Transform transforms the artifacts
func (t *MavenAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		javaVersion := ""
		var pom maven.Pom
		if len(newArtifact.Paths[artifacts.MavenPomPathType]) == 0 {
			logrus.Errorf("the artifact doesn't contain a maven pom.xml path. Artifact: %+v", newArtifact)
			continue
		}
		if err := pom.Load(newArtifact.Paths[artifacts.MavenPomPathType][0]); err != nil {
			logrus.Errorf("Unable to load pom for %s : %s", newArtifact.Name, err)
		}
		if _, ok := newArtifact.Configs[artifacts.SpringBootConfigType]; ok {
			jv, err := pom.GetProperty("java.version")
			if err == nil {
				javaVersion = jv
			}
		}
		if javaVersion == "" {
			jv, err := pom.GetProperty("maven.compiler.target")
			if err == nil {
				javaVersion = jv
			}
		}
		mavenConfig := artifacts.MavenConfig{}
		if err := newArtifact.GetConfig(artifacts.MavenConfigType, &mavenConfig); err != nil {
			logrus.Debugf("Unable to load maven config object: %s", err)
		}
		ir := irtypes.IR{}
		irPresent := true
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T : %s", ir, err)
		}

		classifier := ""
		deploymentDir := ""
		deploymentFileName := ""

		// Processing Maven profiles
		defaultProfiles := []string{}
		if pom.Profiles != nil {
			for _, profile := range *pom.Profiles {
				if profile.Activation != nil && profile.Activation.ActiveByDefault && common.IsStringPresent(mavenConfig.MavenProfiles, profile.ID) {
					defaultProfiles = append(defaultProfiles, profile.ID)
				}
			}
		}
		if len(defaultProfiles) == 0 {
			defaultProfiles = mavenConfig.MavenProfiles
		}
		selectedMavenProfiles := qaengine.FetchMultiSelectAnswer(
			common.ConfigServicesKey+common.Delim+newArtifact.Name+common.Delim+common.ConfigActiveMavenProfilesForServiceKeySegment,
			fmt.Sprintf("Choose the Maven profile to be used for the service %s", newArtifact.Name),
			[]string{fmt.Sprintf("Selected Maven profiles will be used for setting configuration for the service %s", newArtifact.Name)},
			defaultProfiles,
			mavenConfig.MavenProfiles,
		)
		if len(selectedMavenProfiles) == 0 {
			logrus.Debugf("No maven profiles selected")
		}
		builds := []maven.Build{}
		if pom.Build != nil {
			builds = append(builds, *pom.Build)
		}
		if pom.Profiles != nil {
			for _, profile := range *pom.Profiles {
				if common.IsStringPresent(selectedMavenProfiles, profile.ID) {
					if profile.Build != nil {
						builds = append(builds, maven.Build{BuildBase: *profile.Build})
					}
				}
			}
		}
		for _, build := range builds {
			if build.Plugins != nil {
				// Iterate over existing plugins
				for _, mavenPlugin := range *build.Plugins {
					// Check if spring-boot-maven-plugin is present
					if mavenPlugin.ArtifactID != "spring-boot-maven-plugin" || mavenPlugin.Executions == nil {
						continue
					}
					isRepackageEnabled := false
					for _, mavenPluginExecution := range *mavenPlugin.Executions {
						if mavenPluginExecution.Goals == nil {
							continue
						}
						if common.IsStringPresent(*mavenPluginExecution.Goals, "repackage") {
							isRepackageEnabled = true
						}
					}
					if !isRepackageEnabled {
						continue
					}
					if mavenPlugin.Configuration.ConfigurationProfiles == nil || len(*mavenPlugin.Configuration.ConfigurationProfiles) == 0 {
						classifier = mavenPlugin.Configuration.Classifier
						break
					}
					for _, configProfile := range *mavenPlugin.Configuration.ConfigurationProfiles {
						// we check if any of these profiles is contained in the list of profiles
						// selected by the user
						// if yes, we look for the classifier property of this plugin and
						// assign it to the classifier variable
						if common.IsStringPresent(selectedMavenProfiles, configProfile) {
							classifier = mavenPlugin.Configuration.Classifier
							break
						}
					}
					break
				}
				logrus.Debugf("classifier: %s", classifier)
			}
			if javaVersion == "" && build.Plugins != nil {
				for _, dep := range *build.Plugins {
					if dep.ArtifactID == "maven-compiler-plugin" {
						javaVersion = dep.Configuration.Target
					}
				}
			}
			if build.FinalName != "" {
				deploymentFileName = build.FinalName
			}
			if build.Directory != "" {
				deploymentDir = build.Directory
			}
		}
		if javaVersion == "" {
			javaVersion = t.MavenConfig.JavaVersion
			if javaVersion == "" {
				javaVersion = defaultJavaVersion
			}
		}
		if deploymentFileName == "" {
			if pom.ArtifactID != "" {
				deploymentFileName = pom.ArtifactID
				if pom.Version != "" {
					deploymentFileName += "-" + pom.Version
				}
				if classifier != "" {
					deploymentFileName += "-" + classifier
				}
			} else {
				deploymentFileName = newArtifact.Name
			}
		}
		if deploymentDir == "" {
			deploymentDir = "target"
		}
		// Springboot profiles handling
		// We collect the Spring Boot profiles from the current service
		springBootConfig := artifacts.SpringBootConfig{}
		isSpringBootApp := true
		if err := newArtifact.GetConfig(artifacts.SpringBootConfigType, &springBootConfig); err != nil {
			logrus.Debugf("Unable to load springboot config object: %s", err)
			isSpringBootApp = false
		}
		// if there are profiles, we ask the user to select
		springBootProfilesFlattened := ""
		var selectedSpringBootProfiles []string
		if springBootConfig.SpringBootProfiles != nil && len(*springBootConfig.SpringBootProfiles) > 0 {
			selectedSpringBootProfiles = qaengine.FetchMultiSelectAnswer(
				common.ConfigServicesKey+common.Delim+newArtifact.Name+common.Delim+common.ConfigActiveSpringBootProfilesForServiceKeySegment,
				fmt.Sprintf("Choose Springboot profiles to be used for the service %s", newArtifact.Name),
				[]string{fmt.Sprintf("Selected Springboot profiles will be used for setting configuration for the service %s", newArtifact.Name)},
				*springBootConfig.SpringBootProfiles,
				*springBootConfig.SpringBootProfiles,
			)
			if len(selectedSpringBootProfiles) != 0 {
				// we flatten the list of Spring Boot profiles for passing it as env var
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
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &sImageName); err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(newArtifact.Name)
		}
		var sConfig artifacts.ServiceConfig
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &sConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), javaVersion)
		if err != nil {
			logrus.Errorf("Unable to find mapping version for java version %s : %s", javaVersion, err)
			javaPackage = defaultJavaPackage
		}
		license, err := os.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license"))
		if err != nil {
			logrus.Errorf("Unable to read Dockerfile license template : %s", err)
		}
		mavenBuild, err := os.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.maven-build"))
		if err != nil {
			logrus.Errorf("Unable to read Dockerfile license template : %s", err)
		}
		tempDir := filepath.Join(t.Env.TempPath, newArtifact.Name)
		os.MkdirAll(tempDir, common.DefaultDirectoryPermission)
		dockerfileTemplate := filepath.Join(tempDir, "Dockerfile.template")
		template := string(license) + "\n" + string(mavenBuild)
		err = os.WriteFile(dockerfileTemplate, []byte(template), common.DefaultFilePermission)
		if err != nil {
			logrus.Errorf("Could not write the generated Build Dockerfile template: %s", err)
		}
		buildDockerfile := filepath.Join(tempDir, "Dockerfile.build")
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.TemplatePathMappingType,
			SrcPath:  dockerfileTemplate,
			DestPath: buildDockerfile,
			TemplateConfig: MavenBuildDockerfileTemplate{
				JavaPackageName: javaPackage,
				EnvVariables:    envVariablesMap,
				MavenVersion:    t.MavenConfig.MavenVersion,
				MavenProfiles:   selectedMavenProfiles,
				MvnwPresent:     mavenConfig.MvnwPresent,
			},
		})
		var mavenArtifact transformertypes.Artifact
		switch artifacts.JavaPackaging(pom.Packaging) {
		case artifacts.WarPackaging:
			mavenArtifact = transformertypes.Artifact{
				Name: newArtifact.Name,
				Type: artifacts.WarArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.WarConfigType: artifacts.WarArtifactConfig{
						DeploymentFile:                    deploymentFileName + ".war",
						JavaVersion:                       javaVersion,
						BuildContainerName:                common.DefaultBuildContainerName,
						DeploymentFileDirInBuildContainer: filepath.Join(t.MavenConfig.AppPathInBuildContainer, deploymentDir),
						EnvVariables:                      envVariablesMap,
					},
				},
			}
		case artifacts.EarPackaging:
			mavenArtifact = transformertypes.Artifact{
				Name: newArtifact.Name,
				Type: artifacts.EarArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.EarConfigType: artifacts.EarArtifactConfig{
						DeploymentFile:                    deploymentFileName + ".ear",
						JavaVersion:                       javaVersion,
						BuildContainerName:                common.DefaultBuildContainerName,
						DeploymentFileDirInBuildContainer: filepath.Join(t.MavenConfig.AppPathInBuildContainer, deploymentDir),
						EnvVariables:                      envVariablesMap,
					},
				},
			}
		default:
			ports := ir.GetAllServicePorts()
			if isSpringBootApp {
				if len(newArtifact.Paths[artifacts.ServiceDirPathType]) != 0 {
					dir := newArtifact.Paths[artifacts.ServiceDirPathType][0]
					_, _, profilePorts := getSpringBootAppNameProfilesAndPorts(getSpringBootMetadataFiles(dir))
					if len(selectedSpringBootProfiles) > 0 {
						for _, selectedSpringBootProfile := range selectedSpringBootProfiles {
							ports = append(ports, profilePorts[selectedSpringBootProfile]...)
						}
					} else if _, ok := profilePorts[defaultSpringProfile]; ok {
						ports = append(ports, profilePorts[defaultSpringProfile]...)
					}
				} else {
					logrus.Warnf("there are no service directory paths for the artifact: %+v", newArtifact)
				}
			}
			if len(ports) == 0 {
				ports = append(ports, common.DefaultServicePort)
			}
			port := commonqa.GetPortForService(ports, newArtifact.Name)
			if isSpringBootApp {
				envVariablesMap["SERVER_PORT"] = fmt.Sprintf("%d", port)
			} else {
				envVariablesMap["PORT"] = fmt.Sprintf("%d", port)
			}
			mavenArtifact = transformertypes.Artifact{
				Name: newArtifact.Name,
				Type: artifacts.JarArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.JarConfigType: artifacts.JarArtifactConfig{
						DeploymentFile:                    deploymentFileName + ".jar",
						JavaVersion:                       javaVersion,
						BuildContainerName:                common.DefaultBuildContainerName,
						DeploymentFileDirInBuildContainer: filepath.Join(t.MavenConfig.AppPathInBuildContainer, deploymentDir),
						EnvVariables:                      envVariablesMap,
						Port:                              port,
					},
				},
			}
		}
		if irPresent {
			mavenArtifact.Configs[irtypes.IRConfigType] = injectProperties(ir, newArtifact.Name)
		}
		if mavenArtifact.Configs == nil {
			mavenArtifact.Configs = map[transformertypes.ConfigType]interface{}{}
		}
		mavenArtifact.Configs[artifacts.ImageNameConfigType] = sImageName
		mavenArtifact.Configs[artifacts.ServiceConfigType] = sConfig
		if mavenArtifact.Paths == nil {
			mavenArtifact.Paths = map[transformertypes.PathType][]string{}
		}
		mavenArtifact.Paths[artifacts.BuildContainerFileType] = []string{buildDockerfile}
		mavenArtifact.Paths[artifacts.ServiceDirPathType] = newArtifact.Paths[artifacts.ServiceDirPathType]
		createdArtifacts = append(createdArtifacts, mavenArtifact)
	}
	return pathMappings, createdArtifacts, nil
}
