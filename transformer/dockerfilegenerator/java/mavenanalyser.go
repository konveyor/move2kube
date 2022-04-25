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
	// MAVEN_COMPILER_PLUGIN is the name of the maven plugin that compiles the java code.
	MAVEN_COMPILER_PLUGIN = "maven-compiler-plugin"
	// MAVEN_JAR_PLUGIN is the name of the maven plugin that packages the java code.
	MAVEN_JAR_PLUGIN = "maven-jar-plugin"
	// SPRING_BOOT_MAVEN_PLUGIN is the name of the maven plugin that Spring Boot uses.
	SPRING_BOOT_MAVEN_PLUGIN = "spring-boot-maven-plugin"
	// MAVEN_DEFAULT_BUILD_DIR is the name of the default build directory
	MAVEN_DEFAULT_BUILD_DIR = "target"
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
	MvnwPresent        bool
	IsParentPom        bool
	JavaPackageName    string
	MavenVersion       string
	BuildContainerName string
	MavenProfiles      []string
	EnvVariables       map[string]string
}

// Init initializes the transformer
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
	// There will be at most one file path because GetFilesInCurrentDirectory does not check subdirectories.
	mavenFilePaths, err := common.GetFilesInCurrentDirectory(dir, []string{maven.PomXMLFileName}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to look for maven %s files in the directory %s . Error: %q", maven.PomXMLFileName, dir, err)
	}
	if len(mavenFilePaths) == 0 {
		return nil, nil
	}
	logrus.Debugf("found %d pom.xml files in the directory %s . files: %+v", len(mavenFilePaths), dir, mavenFilePaths)
	pom := &maven.Pom{}
	pomFileDir := dir
	pomFilePath := mavenFilePaths[0]
	if err := pom.Load(pomFilePath); err != nil {
		return nil, fmt.Errorf("failed to load the pom.xml file at the path %s . Error: %q", pomFilePath, err)
	}
	isMvnwPresent := false
	if mvnwFilePaths, err := common.GetFilesInCurrentDirectory(dir, []string{"mvnw"}, nil); err == nil && len(mvnwFilePaths) > 0 {
		isMvnwPresent = true
	}
	packaging := artifacts.JavaPackaging(pom.Packaging)
	mavenProfiles := []string{}
	if pom.Profiles != nil {
		for _, profile := range *pom.Profiles {
			mavenProfiles = append(mavenProfiles, profile.ID)
		}
	}
	javaMavenServiceArtifact := transformertypes.Artifact{
		Name: pom.ArtifactID,
		Type: artifacts.ServiceArtifactType,
		Paths: map[transformertypes.PathType][]string{
			artifacts.ServiceDirPathType: {pomFileDir},
			artifacts.MavenPomPathType:   {pomFilePath},
		},
		Configs: map[transformertypes.ConfigType]interface{}{},
	}
	childModules := []artifacts.ChildModule{}
	if isParentPom(pom) {
		packaging = artifacts.PomPackaging
		javaMavenServiceArtifact.Paths[artifacts.ServiceDirPathType] = []string{}
		logrus.Debugf("parent pom.xml detected at the path %s", pomFilePath)
		// get the child/sub modules of the parent pom
		if pom.Modules == nil {
			return nil, fmt.Errorf("the list of child modules is empty for the parent pom.xml at path %s", pomFilePath)
		}
		for _, relChildModulePomPath := range *pom.Modules {
			relChildModulePomPath = filepath.Clean(relChildModulePomPath)
			if filepath.Ext(relChildModulePomPath) != ".xml" {
				relChildModulePomPath = filepath.Join(relChildModulePomPath, maven.PomXMLFileName)
			}
			childModulePomPath := filepath.Join(pomFileDir, relChildModulePomPath)
			childModulePom := &maven.Pom{}
			if err := childModulePom.Load(childModulePomPath); err != nil {
				logrus.Errorf("failed to load the child module pom.xml file at path %s Error: %q", childModulePomPath, err)
				continue
			}
			childModules = append(childModules, artifacts.ChildModule{Name: childModulePom.ArtifactID, RelPomPath: relChildModulePomPath})
			javaMavenServiceArtifact.Paths[artifacts.ServiceDirPathType] = append(javaMavenServiceArtifact.Paths[artifacts.ServiceDirPathType], filepath.Dir(childModulePomPath))
		}
	}
	if packaging == "" {
		packaging = artifacts.JarPackaging
	}
	artifactType, err := packagingToArtifactType(packaging)
	if err != nil {
		return nil, fmt.Errorf("failed to find an artifact type corresponding to the packaging type %s . Error: %q", packaging, err)
	}
	javaMavenServiceArtifact.Configs[artifacts.MavenConfigType] = artifacts.MavenConfig{
		MavenAppName:  pom.ArtifactID,
		ArtifactType:  artifactType,
		IsMvnwPresent: isMvnwPresent,
		MavenProfiles: mavenProfiles,
		ChildModules:  childModules,
	}
	services := map[string][]transformertypes.Artifact{javaMavenServiceArtifact.Name: {javaMavenServiceArtifact}}
	return services, nil
}

// Transform transforms the input artifacts mostly handling artifacts created during the plan phase.
func (t *MavenAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if newArtifact.Type != artifacts.ServiceArtifactType {
			continue
		}
		mavenConfig := artifacts.MavenConfig{}
		if err := newArtifact.GetConfig(artifacts.MavenConfigType, &mavenConfig); err != nil {
			continue
		}
		mavenPomPaths := newArtifact.Paths[artifacts.MavenPomPathType]
		if len(mavenPomPaths) == 0 {
			logrus.Errorf("the artifact doesn't contain any maven pom.xml paths. Artifact: %+v", newArtifact)
			continue
		}
		pom := &maven.Pom{}
		pomFilePath := mavenPomPaths[0] // In a multi-module project this is just the parent pom.xml
		if err := pom.Load(pomFilePath); err != nil {
			logrus.Errorf("failed to load the pom.xml file at path %s . Error: %q", pomFilePath, err)
			continue
		}
		currPathMappings, currArtifacts, err := t.TransformArtifact(newArtifact, alreadySeenArtifacts, pom, pomFilePath, mavenConfig)
		if err != nil {
			logrus.Errorf("failed to transform the artifact: %+v . Error: %q", newArtifact, err)
			continue
		}
		pathMappings = append(pathMappings, currPathMappings...)
		createdArtifacts = append(createdArtifacts, currArtifacts...)
	}
	return pathMappings, createdArtifacts, nil
}

type infoT struct {
	Name               string
	Type               transformertypes.ArtifactType
	IsParentPom        bool
	IsMvnwPresent      bool
	JavaVersion        string
	DeploymentFilePath string
	MavenProfiles      []string
	ChildModules       []artifacts.ChildModule
	SpringBoot         *artifacts.SpringBootConfig
}

// TransformArtifact is the same as Transform but operating on a single artifact and its pom.xml at a time.
func (t *MavenAnalyser) TransformArtifact(newArtifact transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact, pom *maven.Pom, pomFilePath string, mavenConfig artifacts.MavenConfig) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	// Collect data to fill the template.
	info, err := getInfoFromPom(pom, nil, pomFilePath, &mavenConfig)
	if err != nil {
		return pathMappings, createdArtifacts, fmt.Errorf("failed to get the info from the pom.xml at path %s . Error: %q", pomFilePath, err)
	}
	if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 {
		return pathMappings, createdArtifacts, fmt.Errorf("the service directory is missing for the artifact: %+v", newArtifact)
	}
	serviceDir := newArtifact.Paths[artifacts.ServiceDirPathType][0]
	imageToCopyFrom := common.DefaultBuildContainerName
	if info.IsParentPom {
		imageToCopyFrom = info.Name
	}
	// Write the Dockerfile template to a temporary file for the pathmapping to pick it up.
	dockerfileTemplate, err := t.getDockerfileTemplate()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the Dockerfile template. Error: %q", err)
	}
	tempDir, err := os.MkdirTemp(t.Env.TempPath, "maven-transformer-build-*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to make the temporary directory %s . Error: %q", tempDir, err)
	}
	dockerfileTemplatePath := filepath.Join(tempDir, "Dockerfile.template")
	if err := os.WriteFile(dockerfileTemplatePath, []byte(dockerfileTemplate), common.DefaultFilePermission); err != nil {
		return nil, nil, fmt.Errorf("failed to write the Dockerfile template to a temporary file at path %s .  Error: %q", dockerfileTemplatePath, err)
	}
	// Fill in the Dockerfile template and write it to a temporary file using a pathmapping.
	dockerfilePath := filepath.Join(tempDir, "Dockerfile.build")
	if info.IsParentPom {
		pomFileDir := filepath.Dir(pomFilePath)
		relPomFileDir, err := filepath.Rel(t.Env.GetEnvironmentSource(), pomFileDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to make the service directory %s relative to the source code directory %s . Error: %q", pomFileDir, t.Env.GetEnvironmentSource(), err)
		}
		dockerfilePath = filepath.Join(common.DefaultSourceDir, relPomFileDir, common.DefaultDockerfileName)
		// Make sure the source code directory has been copied over first.
		copySourceDirPathMapping := transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}
		pathMappings = append(pathMappings, copySourceDirPathMapping)
	}
	lowestJavaVersion := ""
	if info.IsParentPom {
		// have jar/war/ear analyzer transformer generate a Dockerfile with only the run stage for each of the child modules
		selectedChildModuleNames := []string{}
		for _, childModule := range info.ChildModules {
			selectedChildModuleNames = append(selectedChildModuleNames, childModule.Name)
		}
		quesKey := fmt.Sprintf(common.ConfigServicesChildModulesNamesKey, info.Name)
		desc := fmt.Sprintf("For the multi-module Maven project '%s', please select all the child modules that should be run as services in the cluster:", info.Name)
		hints := []string{"deselect child modules that should not be run (like libraries)"}
		selectedChildModuleNames = qaengine.FetchMultiSelectAnswer(quesKey, desc, hints, selectedChildModuleNames, selectedChildModuleNames)
		if len(selectedChildModuleNames) == 0 {
			return pathMappings, createdArtifacts, fmt.Errorf("user deselected all the child modules of the maven multi-module project '%s'", info.Name)
		}
		pomFileDir := filepath.Dir(pomFilePath)
		baseImageDockerfileArtifact := transformertypes.Artifact{
			Name:    info.Name,
			Type:    artifacts.DockerfileArtifactType,
			Paths:   map[transformertypes.PathType][]string{artifacts.DockerfilePathType: {dockerfilePath}}, // TODO: add the context path as well?
			Configs: map[transformertypes.ConfigType]interface{}{artifacts.ImageNameConfigType: artifacts.ImageName{ImageName: info.Name}},
		}
		createdArtifacts = append(createdArtifacts, baseImageDockerfileArtifact)
		for _, childModule := range info.ChildModules {
			if !common.IsStringPresent(selectedChildModuleNames, childModule.Name) {
				continue
			}
			// load the child module pom
			childPomFilePath := filepath.Join(pomFileDir, childModule.RelPomPath)
			childPom := &maven.Pom{}
			if err := childPom.Load(childPomFilePath); err != nil {
				logrus.Errorf("failed to load the child pom.xml at path %s . Error: %q", childPomFilePath, err)
				continue
			}
			// get info about the child module
			childModuleInfo, err := getInfoFromPom(childPom, pom, childPomFilePath, nil)
			if err != nil {
				logrus.Errorf("failed to get information from the child pom %+v . Error: %q", childPom, err)
				continue
			}
			if lowestJavaVersion == "" {
				lowestJavaVersion = childModuleInfo.JavaVersion
			}
			// find the port for the child module
			desc := fmt.Sprintf("Select the spring boot profiles for the service '%s' :", childModule.Name)
			hints := []string{"select all the profiles that are applicable"}
			detectedPorts := []int32{}
			envVarsMap := map[string]string{}
			if childModuleInfo.SpringBoot != nil {
				if childModuleInfo.SpringBoot.SpringBootProfiles != nil && len(*childModuleInfo.SpringBoot.SpringBootProfiles) != 0 {
					quesKey := common.ConfigServicesKey + common.Delim + info.Name + common.Delim + "childModules" + common.Delim + childModule.Name + common.Delim + "springBootProfiles"
					selectedSpringProfiles := qaengine.FetchMultiSelectAnswer(quesKey, desc, hints, *childModuleInfo.SpringBoot.SpringBootProfiles, *childModuleInfo.SpringBoot.SpringBootProfiles)
					for _, selectedSpringProfile := range selectedSpringProfiles {
						detectedPorts = append(detectedPorts, childModuleInfo.SpringBoot.SpringBootProfilePorts[selectedSpringProfile]...)
					}
					envVarsMap["SPRING_PROFILES_ACTIVE"] = strings.Join(selectedSpringProfiles, ",")
				} else {
					detectedPorts = childModuleInfo.SpringBoot.SpringBootProfilePorts[defaultSpringProfile]
				}
			}
			selectedPort := commonqa.GetPortForService(detectedPorts, info.Name+common.Delim+"childModules"+common.Delim+childModule.Name)
			if childModuleInfo.SpringBoot != nil {
				envVarsMap["SERVER_PORT"] = fmt.Sprintf("%d", selectedPort)
			} else {
				envVarsMap["PORT"] = fmt.Sprintf("%d", selectedPort)
			}
			// create a JAR/WAR/EAR artifact for the child module
			relDeploymentFilePath, err := filepath.Rel(serviceDir, childModuleInfo.DeploymentFilePath)
			if err != nil {
				logrus.Errorf("failed to make the path %s relative to the service directory %s . Error: %q", childModuleInfo.DeploymentFilePath, serviceDir, err)
				continue
			}
			insideContainerDepFilePath := filepath.Join(t.MavenConfig.AppPathInBuildContainer, relDeploymentFilePath)
			createdArtifact := transformertypes.Artifact{
				Name:  childModule.Name,
				Type:  childModuleInfo.Type,
				Paths: map[transformertypes.PathType][]string{artifacts.ServiceDirPathType: {filepath.Dir(childPomFilePath)}},
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
			createdArtifacts = append(createdArtifacts, createdArtifact)
		}
	} else {
		lowestJavaVersion = info.JavaVersion
		// find the port for the child module
		desc := fmt.Sprintf("Select the spring boot profiles for the service '%s' :", info.Name)
		hints := []string{"select all the profiles that are applicable"}
		detectedPorts := []int32{}
		envVarsMap := map[string]string{}
		if info.SpringBoot != nil {
			if info.SpringBoot.SpringBootProfiles != nil && len(*info.SpringBoot.SpringBootProfiles) != 0 {
				quesKey := common.ConfigServicesKey + common.Delim + info.Name + common.Delim + "springBootProfiles"
				selectedSpringProfiles := qaengine.FetchMultiSelectAnswer(quesKey, desc, hints, *info.SpringBoot.SpringBootProfiles, *info.SpringBoot.SpringBootProfiles)
				for _, selectedSpringProfile := range selectedSpringProfiles {
					detectedPorts = append(detectedPorts, info.SpringBoot.SpringBootProfilePorts[selectedSpringProfile]...)
				}
				envVarsMap["SPRING_PROFILES_ACTIVE"] = strings.Join(selectedSpringProfiles, ",")
			} else {
				detectedPorts = info.SpringBoot.SpringBootProfilePorts[defaultSpringProfile]
			}
		}
		selectedPort := commonqa.GetPortForService(detectedPorts, info.Name)
		if info.SpringBoot != nil {
			envVarsMap["SERVER_PORT"] = fmt.Sprintf("%d", selectedPort)
		} else {
			envVarsMap["PORT"] = fmt.Sprintf("%d", selectedPort)
		}
		// Reference the temporary file in an artifact for the jar/war/earanalyzer transformer to pick it up
		relDeploymentFilePath, err := filepath.Rel(serviceDir, info.DeploymentFilePath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to make the path %s relative to the service directory %s . Error: %q", info.DeploymentFilePath, serviceDir, err)
		}
		insideContainerDepFilePath := filepath.Join(t.MavenConfig.AppPathInBuildContainer, relDeploymentFilePath)
		createdArtifact := transformertypes.Artifact{
			Name: info.Name,
			Type: info.Type,
			Paths: map[transformertypes.PathType][]string{
				artifacts.BuildContainerFileType: {dockerfilePath},
				artifacts.ServiceDirPathType:     {filepath.Dir(pomFilePath)},
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				transformertypes.ConfigType(info.Type): artifacts.JarArtifactConfig{
					Port:               selectedPort,
					JavaVersion:        info.JavaVersion,
					BuildContainerName: imageToCopyFrom,
					DeploymentFilePath: insideContainerDepFilePath,
					EnvVariables:       envVarsMap,
				},
				artifacts.ImageNameConfigType: artifacts.ImageName{ImageName: info.Name},
				artifacts.ServiceConfigType:   artifacts.ServiceConfig{ServiceName: info.Name},
			},
		}
		{
			ir := irtypes.IR{}
			if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err == nil {
				ir = injectProperties(ir, info.Name)
				createdArtifact.Configs[irtypes.IRConfigType] = ir
			}
		}
		createdArtifacts = append(createdArtifacts, createdArtifact)
	}
	if lowestJavaVersion == "" {
		lowestJavaVersion = defaultJavaVersion
	}
	javaPackageName, err := t.getJavaPackage(lowestJavaVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the java package for the java version %s . Error: %q", lowestJavaVersion, err)
	}
	selectedMavenProfiles := qaengine.FetchMultiSelectAnswer(
		common.ConfigServicesKey+common.Delim+info.Name+common.Delim+"mavenProfiles",
		fmt.Sprintf("select the maven profiles to use for the service '%s'", info.Name),
		[]string{"the selected maven profiles will be used during the build"},
		info.MavenProfiles,
		info.MavenProfiles,
	)
	pathMapping := transformertypes.PathMapping{
		Type:     transformertypes.TemplatePathMappingType,
		SrcPath:  dockerfileTemplatePath,
		DestPath: dockerfilePath,
		TemplateConfig: MavenBuildDockerfileTemplate{
			MvnwPresent:        info.IsMvnwPresent,
			IsParentPom:        info.IsParentPom,
			JavaPackageName:    javaPackageName,
			MavenVersion:       t.MavenConfig.MavenVersion,
			BuildContainerName: imageToCopyFrom,
			MavenProfiles:      selectedMavenProfiles,
			EnvVariables:       map[string]string{}, // TODO: Something about getting env vars from the IR config inside the artifact coming from the cloud foundry transformer?
		},
	}
	pathMappings = append(pathMappings, pathMapping)
	return pathMappings, createdArtifacts, nil
}

func getInfoFromPom(pom, parentPom *maven.Pom, pomFilePath string, mavenConfig *artifacts.MavenConfig) (infoT, error) {
	name := pom.ArtifactID
	if mavenConfig != nil {
		name = mavenConfig.MavenAppName
	}
	artifactType, err := packagingToArtifactType(artifacts.JavaPackaging(pom.Packaging))
	if err != nil && string(artifactType) != pom.Packaging {
		logrus.Warnf("failed to convert the maven pom.xml packaging field '%s' to an artifact type. Error: %q", pom.Packaging, err)
	}
	if artifactType == "" {
		artifactType = artifacts.JarArtifactType
	}
	if mavenConfig != nil {
		artifactType = mavenConfig.ArtifactType
	}
	isParentPom := isParentPom(pom)
	pomFileDir := filepath.Dir(pomFilePath)
	isMvnwPresent := false
	if mavenConfig != nil {
		isMvnwPresent = mavenConfig.IsMvnwPresent
	} else {
		if mvnwFilePaths, err := common.GetFilesInCurrentDirectory(pomFileDir, []string{"mvnw"}, nil); err == nil && len(mvnwFilePaths) > 0 {
			isMvnwPresent = true
		}
	}
	javaVersion := getJavaVersionFromPom(pom)
	mavenProfiles := []string{}
	if pom.Profiles != nil {
		for _, profile := range *pom.Profiles {
			mavenProfiles = append(mavenProfiles, profile.ID)
		}
	}
	if mavenConfig != nil {
		mavenProfiles = mavenConfig.MavenProfiles
	}
	childModules := []artifacts.ChildModule{}
	if mavenConfig != nil {
		childModules = mavenConfig.ChildModules
	} else {
		if pom.Modules != nil {
			for _, module := range *pom.Modules {
				relChildPomPath := module
				if filepath.Ext(relChildPomPath) != ".xml" {
					relChildPomPath = filepath.Join(relChildPomPath, maven.PomXMLFileName)
				}
				childPom := maven.Pom{}
				childPomPath := filepath.Join(pomFileDir, relChildPomPath)
				if err := childPom.Load(childPomPath); err != nil {
					logrus.Errorf("failed to load the child pom.xml file at path %s . Error: %q", childPomPath, err)
					continue
				}
				childModule := artifacts.ChildModule{Name: pom.ArtifactID, RelPomPath: relChildPomPath}
				childModules = append(childModules, childModule)
			}
		}
	}
	deploymentFilePath, err := getDeploymentFilePathFromPom(pom, pomFileDir)
	if err != nil {
		logrus.Errorf("failed to get the deployment (jar/war/ear) file path for the pom.xml %+v . Error: %q", pom, err)
	}
	return infoT{
		Name:               name,
		Type:               artifactType,
		IsParentPom:        isParentPom,
		IsMvnwPresent:      isMvnwPresent,
		JavaVersion:        javaVersion,
		DeploymentFilePath: deploymentFilePath,
		MavenProfiles:      mavenProfiles,
		ChildModules:       childModules,
		SpringBoot:         getSpringBootConfigFromPom(pomFileDir, pom, parentPom),
	}, nil
}

func (t *MavenAnalyser) getDockerfileTemplate() (string, error) {
	// multi stage build similar to https://nieldw.medium.com/caching-maven-dependencies-in-a-docker-build-dca6ca7ad612
	licenseFilePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license")
	license, err := os.ReadFile(licenseFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read the Dockerfile license file at path %s . Error: %q", licenseFilePath, err)
	}
	mavenBuildTemplatePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.maven-build")
	mavenBuildTemplate, err := os.ReadFile(mavenBuildTemplatePath)
	if err != nil {
		return string(license), fmt.Errorf("failed to read the Dockerfile Maven build template file at path %s . Error: %q", mavenBuildTemplatePath, err)
	}
	return string(license) + "\n" + string(mavenBuildTemplate), nil
}

func (t *MavenAnalyser) getJavaPackage(javaVersion string) (string, error) {
	javaVersionToPackageMappingFilePath := filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath)
	return getJavaPackage(javaVersionToPackageMappingFilePath, javaVersion)
}

// helper functions

func getDeploymentFilePathFromPom(pom *maven.Pom, pomFileDir string) (string, error) {
	packaging := pom.Packaging
	if packaging == "" {
		packaging = string(artifacts.JarPackaging)
	}
	if pom.Build != nil {
		if pom.Build.FinalName != "" {
			return filepath.Join(pomFileDir, MAVEN_DEFAULT_BUILD_DIR, pom.Build.FinalName+"."+packaging), nil
		}
		for _, plugin := range *pom.Build.Plugins {
			if plugin.ArtifactID != MAVEN_COMPILER_PLUGIN {
				continue
			}
			if plugin.Configuration.FinalName != "" {
				return filepath.Join(pomFileDir, MAVEN_DEFAULT_BUILD_DIR, plugin.Configuration.FinalName+"."+packaging), nil
			}
		}
	}
	return filepath.Join(pomFileDir, MAVEN_DEFAULT_BUILD_DIR, pom.ArtifactID+"-"+pom.Version+"."+packaging), nil
}

func isParentPom(pom *maven.Pom) bool {
	return pom.Packaging == string(artifacts.PomPackaging) || (pom.Modules != nil && len(*pom.Modules) > 0)
}

func getJavaVersionFromPom(pom *maven.Pom) string {
	if pom == nil {
		return ""
	}
	if pom.Properties != nil {
		jv, ok := pom.Properties.Entries["java.version"]
		if ok && jv != "" {
			return jv
		}
		jv, ok = pom.Properties.Entries["maven.compiler.target"]
		if ok && jv != "" {
			return jv
		}
		jv, ok = pom.Properties.Entries["maven.compiler.source"]
		if ok && jv != "" {
			return jv
		}
	}
	if pom.Build.Plugins != nil {
		for _, plugin := range *pom.Build.Plugins {
			if plugin.ArtifactID == MAVEN_COMPILER_PLUGIN {
				if plugin.Configuration.Target != "" {
					return plugin.Configuration.Target
				}
				if plugin.Configuration.Source != "" {
					return plugin.Configuration.Source
				}
			}
		}
	}
	return ""
}

func packagingToArtifactType(packaging artifacts.JavaPackaging) (transformertypes.ArtifactType, error) {
	switch strings.ToLower(string(packaging)) {
	case string(artifacts.JarPackaging):
		return artifacts.JarArtifactType, nil
	case string(artifacts.WarPackaging):
		return artifacts.WarArtifactType, nil
	case string(artifacts.EarPackaging):
		return artifacts.EarArtifactType, nil
	default:
		return transformertypes.ArtifactType(packaging), fmt.Errorf("the packaging type '%s' does not have a corresponding artifcat type", packaging)
	}
}

func getSpringBootConfigFromPom(pomFileDir string, pom *maven.Pom, parentPom *maven.Pom) *artifacts.SpringBootConfig {
	getSpringBootConfig := func(dependency maven.Dependency) *artifacts.SpringBootConfig {
		if dependency.GroupID != springBootGroup {
			return nil
		}
		springAppName, springProfiles, profilePorts := getSpringBootAppNameProfilesAndPortsFromDir(pomFileDir)
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
	if pom.Dependencies != nil {
		for _, dependency := range *pom.Dependencies {
			if springConfig := getSpringBootConfig(dependency); springConfig != nil {
				return springConfig
			}
		}
	}
	if pom.DependencyManagement != nil && pom.DependencyManagement.Dependencies != nil {
		for _, dependency := range *pom.DependencyManagement.Dependencies {
			if springConfig := getSpringBootConfig(dependency); springConfig != nil {
				return springConfig
			}
		}
	}
	// look for spring boot in parent pom.xml
	if parentPom != nil {
		if parentPom.Dependencies != nil {
			for _, dependency := range *parentPom.Dependencies {
				if springConfig := getSpringBootConfig(dependency); springConfig != nil {
					return springConfig
				}
			}
		}
		if parentPom.DependencyManagement != nil && parentPom.DependencyManagement.Dependencies != nil {
			for _, dependency := range *parentPom.DependencyManagement.Dependencies {
				if springConfig := getSpringBootConfig(dependency); springConfig != nil {
					return springConfig
				}
			}
		}
	}
	return nil
}
