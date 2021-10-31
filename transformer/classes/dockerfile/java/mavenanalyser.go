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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/source/maven"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	defaultAppPathInContainer = "/app/"
)

var (
	defaultResourcesPath = filepath.Join("src", "main", "resources")
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
	MavenProfiles   []string
}

// Init Initializes the transformer
func (t *MavenAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.MavenConfig = &MavenYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, &t.MavenConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.MavenConfig, err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *MavenAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *MavenAnalyser) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *MavenAnalyser) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	namedServices = map[string]transformertypes.ServicePlan{}
	mavenFilePaths, err := common.GetFilesInCurrentDirectory(dir, []string{maven.PomXMLFileName}, nil)
	if err != nil {
		logrus.Errorf("Error while parsing directory %s for maven file : %s", dir, err)
		return nil, nil, err
	}
	if len(mavenFilePaths) == 0 {
		return nil, nil, nil
	}
	pom := &maven.Pom{}
	err = pom.Load(mavenFilePaths[0])
	if err != nil {
		logrus.Errorf("Unable to unmarshal pom file (%s): %s", mavenFilePaths[0], err)
		return nil, nil, err
	}
	if pom.Modules != nil && len(*(pom.Modules)) != 0 {
		logrus.Debugf("Parent pom detected (%s). Ignoring.", mavenFilePaths[0])
		return nil, nil, nil
	}
	appName := pom.ArtifactID
	profiles := []string{}
	if pom.Profiles != nil {
		for _, profile := range *pom.Profiles {
			profiles = append(profiles, profile.ID)
		}
	}
	ct := transformertypes.TransformerPlan{
		Mode:              transformertypes.ModeContainer,
		ArtifactTypes:     []transformertypes.ArtifactType{irtypes.IRArtifactType, artifacts.ContainerBuildArtifactType},
		BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
		Configs:           map[transformertypes.ConfigType]interface{}{},
		Paths: map[transformertypes.PathType][]string{
			artifacts.MavenPomPathType:    {filepath.Join(dir, maven.PomXMLFileName)},
			artifacts.ProjectPathPathType: {dir},
		},
	}
	mc := artifacts.MavenConfig{}
	mc.ArtifactType = pom.Packaging
	if len(profiles) != 0 {
		mc.MavenProfiles = profiles
	}
	if mc.ArtifactType == "" {
		mc.ArtifactType = "jar"
	}
	if pom.ArtifactID != "" {
		mc.MavenAppName = pom.ArtifactID
	}
	ct.Configs[artifacts.MavenConfigType] = mc
	for _, dependency := range *pom.Dependencies {
		if dependency.GroupID == "org.springframework.boot" {
			sbc := artifacts.SpringBootConfig{}
			sbc.SpringBootAppName, sbc.SpringBootProfiles = getSpringBootAppNameAndProfilesFromDir(dir)
			if dependency.Version != "" {
				sbc.SpringBootVersion = dependency.Version
			}
			ct.Configs[artifacts.SpringBootConfigType] = sbc
			break
		}
	}
	if appName == "" {
		unnamedServices = append(unnamedServices, ct)
	} else {
		namedServices[appName] = append(namedServices[appName], ct)
	}
	return
}

// Transform transforms the artifacts
func (t *MavenAnalyser) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}
		javaVersion := ""
		var pom maven.Pom
		if len(a.Paths[artifacts.MavenPomPathType]) == 0 {
			err := fmt.Errorf("unable to find pom for %s", a.Name)
			logrus.Errorf("%s", err)
			return nil, nil, err
		}
		err := pom.Load(a.Paths[artifacts.MavenPomPathType][0])
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
			jv, err := pom.GetProperty("maven.compiler.target")
			if err == nil {
				javaVersion = jv
			}
		}
		if javaVersion == "" {
			if pom.Build != nil {
				for _, dep := range *pom.Build.Plugins {
					if dep.ArtifactID == "maven-compiler-plugin" {
						javaVersion = dep.Configuration.Target
					}
				}
			}
		}
		if javaVersion == "" {
			javaVersion = t.MavenConfig.JavaVersion
		}
		mavenConfig := artifacts.MavenConfig{}
		err = a.GetConfig(artifacts.MavenConfigType, &mavenConfig)
		if err != nil {
			logrus.Debugf("Unable to load maven config object: %s", err)
		}
		selectedMavenProfiles := qaengine.FetchMultiSelectAnswer(
			common.ConfigServicesKey+common.Delim+a.Name+common.Delim+common.ConfigActiveMavenProfilesForServiceKeySegment,
			fmt.Sprintf("Choose the Maven profile to be used for the service %s", a.Name),
			[]string{fmt.Sprintf("Selected Maven profiles will be used for setting configuration for the service %s", a.Name)},
			mavenConfig.MavenProfiles,
			mavenConfig.MavenProfiles,
		)
		if len(selectedMavenProfiles) == 0 {
			logrus.Debugf("No maven profiles selected")
		}
		classifier := ""
		if pom.Build != nil && pom.Build.Plugins != nil {
			// Iterate over existing plugins
			for _, mavenPlugin := range *pom.Build.Plugins {
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
		javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), "mappings/javapackageversions.yaml"), javaVersion)
		if err != nil {
			logrus.Errorf("Unable to find mapping version for java version %s : %s", javaVersion, err)
			javaPackage = "java-1.8.0-openjdk-devel"
		}
		license, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license"))
		if err != nil {
			logrus.Errorf("Unable to read Dockerfile license template : %s", err)
		}
		mavenBuild, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.maven-build"))
		if err != nil {
			logrus.Errorf("Unable to read Dockerfile license template : %s", err)
		}
		tempDir := filepath.Join(t.Env.TempPath, a.Name)
		os.MkdirAll(tempDir, common.DefaultDirectoryPermission)
		dockerfileTemplate := filepath.Join(tempDir, "Dockerfile.template")
		template := string(license) + "\n" + string(mavenBuild)
		err = ioutil.WriteFile(dockerfileTemplate, []byte(template), common.DefaultFilePermission)
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
				MavenProfiles:   selectedMavenProfiles,
			},
		})

		deploymentFileName := pom.ArtifactID + "-" + pom.Version
		if classifier != "" {
			deploymentFileName = deploymentFileName + "-" + classifier
		}

		switch pom.Packaging {
		case WarPackaging:
			createdArtifacts = append(createdArtifacts, transformertypes.Artifact{
				Name:     a.Name,
				Artifact: artifacts.WarArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.WarConfigType: artifacts.WarArtifactConfig{
						DeploymentFile:                    deploymentFileName + ".war",
						JavaVersion:                       javaVersion,
						DeploymentFileDirInBuildContainer: filepath.Join(defaultAppPathInContainer, "target"),
					},
				},
				Paths: map[transformertypes.PathType][]string{
					artifacts.BuildContainerFileType: {dockerfileTemplate},
				},
			})
		case EarPackaging:
			createdArtifacts = append(createdArtifacts, transformertypes.Artifact{
				Name:     a.Name,
				Artifact: artifacts.EarArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.EarConfigType: artifacts.EarArtifactConfig{
						DeploymentFile:                    deploymentFileName + ".ear",
						JavaVersion:                       javaVersion,
						DeploymentFileDirInBuildContainer: filepath.Join(defaultAppPathInContainer, "target"),
					},
				},
				Paths: map[transformertypes.PathType][]string{
					artifacts.BuildContainerFileType: {dockerfileTemplate},
				},
			})
		default:
			createdArtifacts = append(createdArtifacts, transformertypes.Artifact{
				Name:     a.Name,
				Artifact: artifacts.JarArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.JarConfigType: artifacts.JarArtifactConfig{
						DeploymentFile:              deploymentFileName + ".jar",
						JavaVersion:                 javaVersion,
						DeploymentFileDir:           filepath.Join(defaultAppPathInContainer, "target"),
						IsDeploymentFileInContainer: true,
					},
					artifacts.ImageNameConfigType: sImageName,
					artifacts.ServiceConfigType:   sConfig,
				},
				Paths: map[transformertypes.PathType][]string{
					artifacts.BuildContainerFileType: {buildDockerfile},
					artifacts.ProjectPathPathType:    a.Paths[artifacts.ProjectPathPathType],
				},
			})
		}
	}
	return pathMappings, createdArtifacts, nil
}
