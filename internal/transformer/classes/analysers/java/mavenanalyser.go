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
	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/source/maven"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	//"github.com/magiconair/properties"
)

const (
	mavenPomXML transformertypes.PathType = "MavenPomXML"
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

type MavenYamlConfig struct {
	MavenVersion            string `yaml:"defaultMavenVersion"`
	JavaVersion             string `yaml:"defaultJavaVersion"`
	AppPathInBuildContainer string `yaml:"appPathInBuildContainer"`
}

type MavenBuildDockerfileTemplate struct {
	JavaPackageName string
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
	mavenFilePath, err := common.GetFileNameInCurrentDirectory(dir, maven.PomXMLFileName, false)
	if err != nil {
		logrus.Errorf("Error while parsing directory %s for maven file : %s", dir, err)
		return nil, nil, err
	}
	if mavenFilePath == "" {
		return nil, nil, nil
	}
	pom := &maven.Pom{}
	err = pom.Load(mavenFilePath)
	if err != nil {
		logrus.Errorf("Unable to unmarshal pom file (%s): %s", mavenFilePath, err)
		return nil, nil, err
	}
	if pom.Modules != nil && len(*(pom.Modules)) != 0 {
		logrus.Debugf("Parent pom detected (%s). Ignoring.", mavenFilePath)
		return nil, nil, nil
	}
	appName := pom.ArtifactID
	ct := transformertypes.TransformerPlan{
		Mode:              transformertypes.ModeContainer,
		ArtifactTypes:     []transformertypes.ArtifactType{irtypes.IRArtifactType, artifacts.ContainerBuildArtifactType},
		BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
		Configs:           map[transformertypes.ConfigType]interface{}{},
		Paths: map[transformertypes.PathType][]string{
			mavenPomXML:                   {filepath.Join(dir, maven.PomXMLFileName)},
			artifacts.ProjectPathPathType: {dir},
		},
	}
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
		if pom.Name != "" {
			namedServices[pom.Name] = append(namedServices[pom.Name], ct)
		} else {
			unnamedServices = append(unnamedServices, ct)
		}
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
		if len(a.Paths[mavenPomXML]) == 0 {
			err := fmt.Errorf("unable to find pom for %s", a.Name)
			logrus.Errorf("%s", err)
			return nil, nil, err
		}
		err := pom.Load(a.Paths[mavenPomXML][0])
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
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(a.Name)
		}

		javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), "mappings/javapackageversions.yaml"), javaVersion)
		if err != nil {
			logrus.Error("Unable to find mapping version for java version %s : %s", javaVersion, err)
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
		var dockerfileTemplate = filepath.Join(t.Env.TempPath, "Dockerfile.template")
		template := string(license) + "\n" + string(mavenBuild)
		err = ioutil.WriteFile(dockerfileTemplate, []byte(template), common.DefaultFilePermission)
		if err != nil {
			logrus.Errorf("Could not write the generated Build Dockerfile template: %s", err)
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.TemplatePathMappingType,
			SrcPath:  dockerfileTemplate,
			DestPath: filepath.Join(t.Env.TempPath, "Dockerfile.build"),
			TemplateConfig: MavenBuildDockerfileTemplate{
				JavaPackageName: javaPackage,
			},
		})
		deploymentFileName := pom.ArtifactID + "-" + pom.Version
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
						DeploymentFile:                    deploymentFileName + ".jar",
						JavaVersion:                       javaVersion,
						DeploymentFileDirInBuildContainer: filepath.Join(defaultAppPathInContainer, "target"),
					},
				},
				Paths: map[transformertypes.PathType][]string{
					artifacts.BuildContainerFileType: {dockerfileTemplate},
				},
			})
		}
	}
	return pathMappings, createdArtifacts, nil
}
