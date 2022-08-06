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

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	defaultJbossPort int32 = 8080
)

// Jboss implements Transformer interface
type Jboss struct {
	Config      transformertypes.Transformer
	Env         *environment.Environment
	JbossConfig *JbossYamlConfig
}

// JbossYamlConfig stores jar related configuration information
type JbossYamlConfig struct {
	JavaVersion string `yaml:"defaultJavaVersion"`
}

// JbossDockerfileTemplate stores parameters for the dockerfile template
type JbossDockerfileTemplate struct {
	JavaPackageName    string
	DeploymentFilePath string
	BuildContainerName string
	Port               int32
	EnvVariables       map[string]string
}

// Init Initializes the transformer
func (t *Jboss) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.JbossConfig = &JbossYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.JbossConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.JbossConfig, err)
		return err
	}
	if t.JbossConfig.JavaVersion == "" {
		t.JbossConfig.JavaVersion = defaultJavaVersion
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Jboss) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *Jboss) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return
}

// Transform transforms the artifacts
func (t *Jboss) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		serviceConfig := artifacts.ServiceConfig{}
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &serviceConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", serviceConfig, err)
			continue
		}
		if serviceConfig.ServiceName == "" {
			serviceConfig.ServiceName = common.MakeStringK8sServiceNameCompliant(newArtifact.Name)
		}
		imageName := artifacts.ImageName{}
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", imageName, err)
		}
		if imageName.ImageName == "" {
			imageName.ImageName = common.MakeStringContainerImageNameCompliant(newArtifact.Name)
		}
		if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 {
			logrus.Errorf("service directory missing from artifact: %+v", newArtifact)
			continue
		}
		serviceDir := newArtifact.Paths[artifacts.ServiceDirPathType][0]
		relServiceDir, err := filepath.Rel(t.Env.GetEnvironmentSource(), serviceDir)
		if err != nil {
			logrus.Errorf("failed to make the service directory %s relative to the source code directory %s . Error: %q", serviceDir, t.Env.GetEnvironmentSource(), err)
			continue
		}
		template, err := t.getDockerfileTemplate(newArtifact)
		if err != nil {
			logrus.Errorf("failed to get the jboss run stage Dockerfile template. Error: %q", err)
			continue
		}
		tempDir := filepath.Join(t.Env.TempPath, newArtifact.Name)
		if err := os.MkdirAll(tempDir, common.DefaultDirectoryPermission); err != nil {
			logrus.Errorf("failed to create the temporary directory %s . Error: %q", tempDir, err)
			continue
		}
		dockerfileTemplatePath := filepath.Join(tempDir, common.DefaultDockerfileName)
		if err := os.WriteFile(dockerfileTemplatePath, []byte(template), common.DefaultFilePermission); err != nil {
			logrus.Errorf("Could not write the generated Build Dockerfile template: %s", err)
		}
		templateData := JbossDockerfileTemplate{}
		warConfig := artifacts.WarArtifactConfig{}
		if err := newArtifact.GetConfig(artifacts.WarConfigType, &warConfig); err == nil {
			// WAR
			javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), warConfig.JavaVersion)
			if err != nil {
				logrus.Errorf("Unable to find mapping version for java version %s : %s", warConfig.JavaVersion, err)
				javaPackage = defaultJavaPackage
			}
			templateData.JavaPackageName = javaPackage
			templateData.DeploymentFilePath = warConfig.DeploymentFilePath
			templateData.Port = defaultJbossPort
			templateData.EnvVariables = warConfig.EnvVariables
			templateData.BuildContainerName = warConfig.BuildContainerName
		} else {
			// EAR
			logrus.Debugf("unable to load config for Transformer into %T : %s", warConfig, err)
			earConfig := artifacts.EarArtifactConfig{}
			if err := newArtifact.GetConfig(artifacts.EarConfigType, &earConfig); err != nil {
				logrus.Debugf("unable to load config for Transformer into %T : %s", earConfig, err)
			}
			javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), earConfig.JavaVersion)
			if err != nil {
				logrus.Errorf("Unable to find mapping version for java version %s : %s", earConfig.JavaVersion, err)
				javaPackage = defaultJavaPackage
			}
			templateData.JavaPackageName = javaPackage
			templateData.DeploymentFilePath = earConfig.DeploymentFilePath
			templateData.Port = defaultJbossPort
			templateData.EnvVariables = earConfig.EnvVariables
			templateData.BuildContainerName = earConfig.BuildContainerName
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		})
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        dockerfileTemplatePath,
			DestPath:       filepath.Join(common.DefaultSourceDir, relServiceDir),
			TemplateConfig: templateData,
		})
		paths := newArtifact.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relServiceDir, common.DefaultDockerfileName)}
		dockerfileArtifact := transformertypes.Artifact{
			Name:  imageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: imageName,
			},
		}
		dockerfileServiceArtifact := transformertypes.Artifact{
			Name:  serviceConfig.ServiceName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: newArtifact.Paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: imageName,
				artifacts.ServiceConfigType:   serviceConfig,
			},
		}
		ir := irtypes.IR{}
		if err = newArtifact.GetConfig(irtypes.IRConfigType, &ir); err == nil {
			dockerfileServiceArtifact.Configs[irtypes.IRConfigType] = ir
		}
		createdArtifacts = append(createdArtifacts, dockerfileArtifact, dockerfileServiceArtifact)
	}
	return pathMappings, createdArtifacts, nil
}

func (t *Jboss) getDockerfileTemplate(newArtifact transformertypes.Artifact) (string, error) {
	jbossRunTemplatePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.jboss")
	jbossRunTemplate, err := os.ReadFile(jbossRunTemplatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read the jboss run stage Dockerfile template at path %s . Error: %q", jbossRunTemplatePath, err)
	}
	dockerFileHead := ""
	if buildContainerPaths := newArtifact.Paths[artifacts.BuildContainerFileType]; len(buildContainerPaths) > 0 {
		dockerfileBuildPath := buildContainerPaths[0]
		dockerFileHeadBytes, err := os.ReadFile(dockerfileBuildPath)
		if err != nil {
			return "", fmt.Errorf("failed to read the build stage Dockerfile template at path %s . Error: %q", dockerfileBuildPath, err)
		}
		dockerFileHead = string(dockerFileHeadBytes)
	} else {
		licenseFilePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license")
		dockerFileHeadBytes, err := os.ReadFile(licenseFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read the Dockerfile license at path %s . Error: %q", licenseFilePath, err)
		}
		dockerFileHead = string(dockerFileHeadBytes)
	}
	return dockerFileHead + "\n" + string(jbossRunTemplate), nil
}
