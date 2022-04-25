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
	"github.com/konveyor/move2kube/types"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	defaultLibertyPort int32 = 9080
)

// Liberty implements Transformer interface
type Liberty struct {
	Config        transformertypes.Transformer
	Env           *environment.Environment
	LibertyConfig *LibertyYamlConfig
}

// LibertyYamlConfig stores jar related configuration information
type LibertyYamlConfig struct {
	JavaVersion string `yaml:"defaultJavaVersion"`
}

// LibertyDockerfileTemplate stores parameters for the dockerfile template
type LibertyDockerfileTemplate struct {
	JavaPackageName    string
	JavaVersion        string
	DeploymentFilePath string
	BuildContainerName string
	Port               int32
	EnvVariables       map[string]string
}

// JavaLibertyImageMapping stores the java version to liberty image version mappings
type JavaLibertyImageMapping struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             JavaLibertyImageMappingSpec `yaml:"spec,omitempty"`
}

// JavaLibertyImageMappingSpec stores the java to liberty image version spec
type JavaLibertyImageMappingSpec struct {
	Mapping map[string]string `yaml:"mapping"`
}

// Init Initializes the transformer
func (t *Liberty) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.LibertyConfig = &LibertyYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.LibertyConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.LibertyConfig, err)
		return err
	}
	// defaults
	if t.LibertyConfig.JavaVersion == "" {
		t.LibertyConfig.JavaVersion = defaultJavaVersion
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Liberty) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *Liberty) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return
}

// Transform transforms the artifacts
func (t *Liberty) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if newArtifact.Type != artifacts.WarArtifactType {
			continue
		}
		serviceConfig := artifacts.ServiceConfig{}
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &serviceConfig); err != nil {
			logrus.Errorf("failed to load service config from the artifact: %+v . Error: %q", serviceConfig, err)
			continue
		}
		imageName := artifacts.ImageName{}
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", imageName, err)
		}
		if imageName.ImageName == "" {
			imageName.ImageName = common.MakeStringContainerImageNameCompliant(serviceConfig.ServiceName)
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
		template, err := t.getDockerfileTemplate(newArtifact)
		if err != nil {
			logrus.Errorf("failed to get the liberty run stage Dockerfile template. Error: %q", err)
			continue
		}
		tempDir := filepath.Join(t.Env.TempPath, newArtifact.Name)
		if err := os.MkdirAll(tempDir, common.DefaultDirectoryPermission); err != nil {
			logrus.Errorf("failed to make the temporary directory %s . Error: %q", tempDir, err)
			continue
		}
		dockerfileTemplatePath := filepath.Join(tempDir, common.DefaultDockerfileName)
		if err := os.WriteFile(dockerfileTemplatePath, []byte(template), common.DefaultFilePermission); err != nil {
			logrus.Errorf("failed to write the liberty Dockerfile template to the temporary file at path %s . Error: %q", dockerfileTemplatePath, err)
			continue
		}
		templateData := LibertyDockerfileTemplate{}
		warConfig := artifacts.WarArtifactConfig{}
		if err := newArtifact.GetConfig(artifacts.WarConfigType, &warConfig); err == nil {
			// WAR
			if warConfig.JavaVersion == "" {
				warConfig.JavaVersion = t.LibertyConfig.JavaVersion
			}
			javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), warConfig.JavaVersion)
			if err != nil {
				logrus.Errorf("Unable to find mapping version for java version %s . Error: %q", warConfig.JavaVersion, err)
				javaPackage = defaultJavaPackage
			}
			templateData.JavaPackageName = javaPackage
			templateData.JavaVersion = warConfig.JavaVersion
			templateData.Port = defaultLibertyPort
			templateData.EnvVariables = warConfig.EnvVariables
			templateData.DeploymentFilePath = warConfig.DeploymentFilePath
			templateData.BuildContainerName = warConfig.BuildContainerName
		} else {
			// EAR
			logrus.Debugf("failed to load war config from the artifact: %+v . Error: %q", newArtifact, err)
			earConfig := artifacts.EarArtifactConfig{}
			if err := newArtifact.GetConfig(artifacts.EarConfigType, &earConfig); err != nil {
				logrus.Debugf("failed to load ear config from the artifact: %+v . Error: %q", newArtifact, err)
			}
			if earConfig.JavaVersion == "" {
				earConfig.JavaVersion = t.LibertyConfig.JavaVersion
			}
			javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), earConfig.JavaVersion)
			if err != nil {
				logrus.Errorf("Unable to find mapping version for java version %s : %s", earConfig.JavaVersion, err)
				javaPackage = defaultJavaPackage
			}
			templateData.JavaPackageName = javaPackage
			templateData.JavaVersion = earConfig.JavaVersion
			templateData.Port = defaultLibertyPort
			templateData.EnvVariables = earConfig.EnvVariables
			templateData.DeploymentFilePath = earConfig.DeploymentFilePath
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
		dockerfileServiceAritfact := transformertypes.Artifact{
			Name:  serviceConfig.ServiceName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: newArtifact.Paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: imageName,
				artifacts.ServiceConfigType:   serviceConfig,
			},
		}
		ir := irtypes.IR{}
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err == nil {
			dockerfileServiceAritfact.Configs[irtypes.IRConfigType] = ir
		}
		createdArtifacts = append(createdArtifacts, dockerfileArtifact, dockerfileServiceAritfact)
	}
	return pathMappings, createdArtifacts, nil
}

func (t *Liberty) getDockerfileTemplate(newArtifact transformertypes.Artifact) (string, error) {
	libertyRunTemplatePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.liberty")
	libertyRunTemplate, err := os.ReadFile(libertyRunTemplatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read the liberty run stage Dockerfile template at path %s . Error: %q", libertyRunTemplatePath, err)
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
	return dockerFileHead + "\n" + string(libertyRunTemplate), nil
}
