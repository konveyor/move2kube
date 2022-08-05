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
	tomcatDefaultPort = 8080
)

// Tomcat implements Transformer interface
type Tomcat struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	TomcatConfig *TomcatYamlConfig
}

// TomcatYamlConfig stores jar related configuration information
type TomcatYamlConfig struct {
	JavaVersion string `yaml:"defaultJavaVersion"`
}

// TomcatDockerfileTemplate stores parameters for the dockerfile template
type TomcatDockerfileTemplate struct {
	JavaPackageName    string
	JavaVersion        string
	DeploymentFilePath string
	BuildContainerName string
	Port               int32
	EnvVariables       map[string]string
}

// Init Initializes the transformer
func (t *Tomcat) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.TomcatConfig = &TomcatYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.TomcatConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.TomcatConfig, err)
		return err
	}
	if t.TomcatConfig.JavaVersion == "" {
		t.TomcatConfig.JavaVersion = defaultJavaVersion
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Tomcat) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *Tomcat) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return
}

// Transform transforms the artifacts
func (t *Tomcat) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if newArtifact.Type != artifacts.WarArtifactType {
			continue
		}
		serviceConfig := artifacts.ServiceConfig{}
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &serviceConfig); err != nil {
			logrus.Debugf("failed to load service config from the artifact: %+v . Error: %q", newArtifact, err)
			continue
		}
		if serviceConfig.ServiceName == "" {
			serviceConfig.ServiceName = common.MakeStringK8sServiceNameCompliant(newArtifact.Name)
		}
		imageName := artifacts.ImageName{}
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Debugf("failed to load image name config from the artifact: %+v . Error: %q", imageName, err)
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
			logrus.Errorf("failed to make the source path %s relative to the source code directory %s . Error: %q", serviceDir, t.Env.GetEnvironmentSource(), err)
			continue
		}
		dockerfileTemplate, err := t.getDockerfileTemplate(newArtifact)
		if err != nil {
			logrus.Errorf("failed to get the tomcat Dockerfile template. Error: %q", err)
			continue
		}
		tempDir := filepath.Join(t.Env.TempPath, newArtifact.Name)
		if err := os.MkdirAll(tempDir, common.DefaultDirectoryPermission); err != nil {
			logrus.Errorf("failed to create the temporary directory %s . Error: %q", tempDir, err)
			continue
		}
		dockerfileTemplatePath := filepath.Join(tempDir, common.DefaultDockerfileName)
		if err := os.WriteFile(dockerfileTemplatePath, []byte(dockerfileTemplate), common.DefaultFilePermission); err != nil {
			logrus.Errorf("failed to write the tomcat Dockerfile template to a temporary file at path %s . Error: %q", dockerfileTemplatePath, err)
			continue
		}
		warConfig := artifacts.WarArtifactConfig{}
		if err := newArtifact.GetConfig(artifacts.WarConfigType, &warConfig); err != nil {
			logrus.Errorf("failed to load the war config from the artifact %+v . Error: %q", newArtifact, err)
			continue
		}
		if warConfig.JavaVersion == "" {
			warConfig.JavaVersion = t.TomcatConfig.JavaVersion
		}
		javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), warConfig.JavaVersion)
		if err != nil {
			logrus.Errorf("failed to find mapping version for java version %s . Error: %q", warConfig.JavaVersion, err)
			javaPackage = defaultJavaPackage
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		})
		templateData := TomcatDockerfileTemplate{
			JavaPackageName:    javaPackage,
			JavaVersion:        warConfig.JavaVersion,
			DeploymentFilePath: warConfig.DeploymentFilePath,
			Port:               tomcatDefaultPort,
			EnvVariables:       warConfig.EnvVariables,
			BuildContainerName: warConfig.BuildContainerName,
		}
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

func (t *Tomcat) getDockerfileTemplate(newArtifact transformertypes.Artifact) (string, error) {
	tomcatRunTemplatePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.tomcat")
	tomcatRunTemplate, err := os.ReadFile(tomcatRunTemplatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read the tomcat run stage Dockerfile template at path %s . Error: %q", tomcatRunTemplatePath, err)
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
	return dockerFileHead + "\n" + string(tomcatRunTemplate), nil
}
