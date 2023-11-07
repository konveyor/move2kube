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
	irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	//irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/konveyor/move2kube-wasm/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

// JarAnalyser implements Transformer interface
type JarAnalyser struct {
	Config    transformertypes.Transformer
	Env       *environment.Environment
	JarConfig *JarYamlConfig
}

// JarYamlConfig stores jar related configuration information
type JarYamlConfig struct {
	JavaVersion string `yaml:"defaultJavaVersion"`
	DefaultPort int32  `yaml:"defaultPort"`
}

// JarDockerfileTemplate stores parameters for the dockerfile template
type JarDockerfileTemplate struct {
	Port               int32
	JavaPackageName    string
	BuildContainerName string
	DeploymentFilePath string
	DeploymentFilename string
	EnvVariables       map[string]string
}

// Init Initializes the transformer
func (t *JarAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.JarConfig = &JarYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.JarConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.JarConfig, err)
		return err
	}
	if t.JarConfig.JavaVersion == "" {
		t.JarConfig.JavaVersion = defaultJavaVersion
	}
	if t.JarConfig.DefaultPort == 0 {
		t.JarConfig.DefaultPort = common.DefaultServicePort
	}
	return nil
}

// GetConfig returns the transformer config
func (t *JarAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *JarAnalyser) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	services := map[string][]transformertypes.Artifact{}
	paths, err := common.GetFilesInCurrentDirectory(dir, nil, []string{`\.jar$`})
	if err != nil {
		return nil, fmt.Errorf("failed to get the jar files in the directory %s . Error: %q", dir, err)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	for _, path := range paths {
		relPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), path)
		if err != nil {
			logrus.Errorf("failed to make the path %s relative to the sourc code directory %s . Error: %q", path, t.Env.GetEnvironmentSource(), err)
			continue
		}
		serviceName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		normalizedServiceName := common.MakeStringK8sServiceNameCompliant(serviceName)
		newArtifact := transformertypes.Artifact{
			Paths: map[transformertypes.PathType][]string{
				artifacts.JarPathType:        {path},
				artifacts.ServiceDirPathType: {dir},
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.JarConfigType: artifacts.JarArtifactConfig{
					DeploymentFilePath: relPath,
					EnvVariables:       map[string]string{"PORT": cast.ToString(t.JarConfig.DefaultPort)},
					Port:               t.JarConfig.DefaultPort,
					JavaVersion:        t.JarConfig.JavaVersion,
				},
			},
		}
		services[normalizedServiceName] = append(services[normalizedServiceName], newArtifact)
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *JarAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if newArtifact.Type != artifacts.ServiceArtifactType && newArtifact.Type != artifacts.JarArtifactType {
			continue
		}
		jarArtifactConfig := artifacts.JarArtifactConfig{}
		if err := newArtifact.GetConfig(artifacts.JarConfigType, &jarArtifactConfig); err != nil {
			logrus.Debugf("failed to load the JAR config from the artifact %+v . Error: %q", jarArtifactConfig, err)
			continue
		}
		serviceConfig := artifacts.ServiceConfig{}
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &serviceConfig); err != nil {
			logrus.Debugf("failed to load the service config from the artifact %+v . Error: %q", newArtifact, err)
			continue
		}
		imageName := artifacts.ImageName{}
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Debugf("failed to load the image name config from the artifact %+v . Error: %q", newArtifact, err)
		}
		if imageName.ImageName == "" {
			imageName.ImageName = common.MakeStringContainerImageNameCompliant(serviceConfig.ServiceName)
		}
		// get the Dockerfile template
		dockerfileTemplate, _, err := t.getDockerfileTemplate(newArtifact)
		if err != nil {
			logrus.Errorf("failed to get the Dockerfile template for the jar artifact %+v . Error: %q", newArtifact, err)
			continue
		}
		// write the Dockerfile template to a temporary file for a pathmapping to pick it up
		tempDir := filepath.Join(t.Env.TempPath, newArtifact.Name)
		if err := os.MkdirAll(tempDir, common.DefaultDirectoryPermission); err != nil {
			logrus.Errorf("failed to create the temporary directory %s . Error: %q", tempDir, err)
			continue
		}
		dockerfileTemplatePath := filepath.Join(tempDir, common.DefaultDockerfileName)
		if err := os.WriteFile(dockerfileTemplatePath, []byte(dockerfileTemplate), common.DefaultFilePermission); err != nil {
			logrus.Errorf("failed to write the Dockerfile template at path %s . Error: %q", dockerfileTemplatePath, err)
			continue
		}
		// get the data to fill the Dockerfile template
		if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 {
			logrus.Errorf("the service directory path is missing for the artifact: %+v", newArtifact)
			continue
		}
		serviceDir := newArtifact.Paths[artifacts.ServiceDirPathType][0]
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), serviceDir)
		if err != nil {
			logrus.Errorf("failed to make the service directory %s relative to the source code directory %s . Error: %q", serviceDir, t.Env.GetEnvironmentSource(), err)
			continue
		}
		dockerfilePath := filepath.Join(common.DefaultSourceDir, relSrcPath, common.DefaultDockerfileName)
		if jarArtifactConfig.JavaVersion == "" {
			jarArtifactConfig.JavaVersion = t.JarConfig.JavaVersion
		}
		javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), jarArtifactConfig.JavaVersion)
		if err != nil {
			logrus.Errorf("failed to find the java package for the java version %s . Going with java package %s instead. Error: %q", jarArtifactConfig.JavaVersion, defaultJavaPackage, err)
			javaPackage = defaultJavaPackage
		}
		buildContainerName := jarArtifactConfig.BuildContainerName
		pathMappingTemplateConfig := JarDockerfileTemplate{
			Port:               jarArtifactConfig.Port,
			JavaPackageName:    javaPackage,
			BuildContainerName: buildContainerName,
			DeploymentFilePath: jarArtifactConfig.DeploymentFilePath,
			DeploymentFilename: filepath.Base(jarArtifactConfig.DeploymentFilePath),
			EnvVariables:       jarArtifactConfig.EnvVariables,
		}
		// Fill the Dockerfile template using a pathmapping.
		writeDockerfilePathMapping := transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        dockerfileTemplatePath,
			DestPath:       dockerfilePath,
			TemplateConfig: pathMappingTemplateConfig,
		}
		// Make sure the source code directory has been copied over first.
		copySourceDirPathMapping := transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}
		pathMappings = append(pathMappings, copySourceDirPathMapping, writeDockerfilePathMapping)
		// Reference the Dockerfile we created in an artifact for other transformers that consume Dockerfiles.
		paths := newArtifact.Paths
		paths[artifacts.DockerfilePathType] = []string{dockerfilePath}
		dockerfileArtifact := transformertypes.Artifact{
			Name:  imageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: imageName,
			},
		}
		dockerfileForServiceArtifact := transformertypes.Artifact{
			Name:  serviceConfig.ServiceName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: imageName,
				artifacts.ServiceConfigType:   serviceConfig,
			},
		}

		// preserve the ir config and inject cloud foundry vcap properties if it is present

		ir := irtypes.IR{}
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err == nil {
			dockerfileForServiceArtifact.Configs[irtypes.IRConfigType] = ir
		}

		createdArtifacts = append(createdArtifacts, dockerfileArtifact, dockerfileForServiceArtifact)
	}
	return pathMappings, createdArtifacts, nil
}

func (t *JarAnalyser) getDockerfileTemplate(newArtifact transformertypes.Artifact) (string, bool, error) {
	jarRunDockerfileTemplatePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.jar-run")
	jarRunDockerfileTemplate, err := os.ReadFile(jarRunDockerfileTemplatePath)
	if err != nil {
		return "", false, fmt.Errorf("failed to read the JAR run Dockerfile template at path %s . Error: %q", jarRunDockerfileTemplatePath, err)
	}
	dockerFileHead := ""
	isBuildContainerPresent := false
	if buildContainerPaths, ok := newArtifact.Paths[artifacts.BuildContainerFileType]; ok && len(buildContainerPaths) > 0 {
		isBuildContainerPresent = true
		buildStageDockerfilePath := buildContainerPaths[0]
		dockerFileHeadBytes, err := os.ReadFile(buildStageDockerfilePath)
		if err != nil {
			return "", isBuildContainerPresent, fmt.Errorf("failed to read the build stage Dockerfile at path %s . Error: %q", buildStageDockerfilePath, err)
		}
		dockerFileHead = string(dockerFileHeadBytes)
	} else {
		licensePath := filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license")
		dockerFileHeadBytes, err := os.ReadFile(licensePath)
		if err != nil {
			return "", isBuildContainerPresent, fmt.Errorf("failed to read the Dockerfile license at path %s . Error: %q", licensePath, err)
		}
		dockerFileHead = string(dockerFileHeadBytes)
	}
	return string(dockerFileHead) + "\n" + string(jarRunDockerfileTemplate), isBuildContainerPresent, nil
}
