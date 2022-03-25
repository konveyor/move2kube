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
	JavaPackageName                   string
	DeploymentFile                    string
	BuildContainerName                string
	DeploymentFileDirInBuildContainer string
	Port                              int32
	EnvVariables                      map[string]string
	ServiceName                       string
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
func (t *JarAnalyser) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	services = map[string][]transformertypes.Artifact{}
	jarFilePaths, err := common.GetFilesInCurrentDirectory(dir, nil, []string{".*[.]jar"})
	if err != nil {
		logrus.Errorf("Error while parsing directory %s for jar file : %s", dir, err)
		return nil, err
	}
	if len(jarFilePaths) == 0 {
		return nil, nil
	}
	for _, path := range jarFilePaths {
		envVariablesMap := map[string]string{}
		envVariablesMap["PORT"] = fmt.Sprintf("%d", t.JarConfig.DefaultPort)
		newArtifact := transformertypes.Artifact{
			Paths: map[transformertypes.PathType][]string{
				artifacts.JarPathType:        {path},
				artifacts.ServiceDirPathType: {filepath.Dir(path)},
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.JarConfigType: artifacts.JarArtifactConfig{
					DeploymentFile: filepath.Base(path),
					EnvVariables:   envVariablesMap,
					Port:           t.JarConfig.DefaultPort,
					JavaVersion:    t.JarConfig.JavaVersion,
				},
			},
		}
		services[""] = append(services[""], newArtifact)
	}
	return
}

// Transform transforms the artifacts
func (t *JarAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		var sConfig artifacts.ServiceConfig
		err := a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ServiceDirPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ServiceDirPathType][0], err)
		}
		jarRunDockerfile, err := os.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.embedded"))
		if err != nil {
			logrus.Errorf("Unable to read Dockerfile embedded template : %s", err)
		}
		dockerFileHead := ""
		isBuildContainerPresent := false
		if buildContainerPaths, ok := a.Paths[artifacts.BuildContainerFileType]; ok && len(buildContainerPaths) > 0 {
			isBuildContainerPresent = true
			dockerfileBuildDockerfile := buildContainerPaths[0]
			dockerFileHeadBytes, err := os.ReadFile(dockerfileBuildDockerfile)
			if err != nil {
				logrus.Errorf("Unable to read build Dockerfile template : %s", err)
				continue
			}
			dockerFileHead = string(dockerFileHeadBytes)
		} else {
			dockerFileHeadBytes, err := os.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license"))
			if err != nil {
				logrus.Errorf("Unable to read Dockerfile license template : %s", err)
			}
			dockerFileHead = string(dockerFileHeadBytes)
		}
		tempDir := filepath.Join(t.Env.TempPath, a.Name)
		os.MkdirAll(tempDir, common.DefaultDirectoryPermission)
		dockerfileTemplate := filepath.Join(tempDir, common.DefaultDockerfileName)
		template := string(dockerFileHead) + "\n" + string(jarRunDockerfile)
		err = os.WriteFile(dockerfileTemplate, []byte(template), common.DefaultFilePermission)
		if err != nil {
			logrus.Errorf("Could not write the generated Build Dockerfile template: %s", err)
		}
		jarArtifactConfig := artifacts.JarArtifactConfig{}
		err = a.GetConfig(artifacts.JarConfigType, &jarArtifactConfig)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", jarArtifactConfig, err)
		}
		if jarArtifactConfig.JavaVersion == "" {
			jarArtifactConfig.JavaVersion = t.JarConfig.JavaVersion
		}
		javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), jarArtifactConfig.JavaVersion)
		if err != nil {
			logrus.Errorf("Unable to find mapping version for java version %s : %s", jarArtifactConfig.JavaVersion, err)
			javaPackage = defaultJavaPackage
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		})
		if isBuildContainerPresent {
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.TemplatePathMappingType,
				SrcPath:  dockerfileTemplate,
				DestPath: filepath.Join(common.DefaultSourceDir, relSrcPath),
				TemplateConfig: JarDockerfileTemplate{
					JavaPackageName:                   javaPackage,
					DeploymentFile:                    jarArtifactConfig.DeploymentFile,
					BuildContainerName:                jarArtifactConfig.BuildContainerName,
					DeploymentFileDirInBuildContainer: jarArtifactConfig.DeploymentFileDirInBuildContainer,
					Port:                              jarArtifactConfig.Port,
					EnvVariables:                      jarArtifactConfig.EnvVariables,
					ServiceName:                       sConfig.ServiceName,
				},
			})
		} else {
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.TemplatePathMappingType,
				SrcPath:  dockerfileTemplate,
				DestPath: filepath.Join(common.DefaultSourceDir, relSrcPath),
				TemplateConfig: JarDockerfileTemplate{
					JavaPackageName: javaPackage,
					DeploymentFile:  jarArtifactConfig.DeploymentFile,
					Port:            jarArtifactConfig.Port,
					EnvVariables:    jarArtifactConfig.EnvVariables,
					ServiceName:     sConfig.ServiceName,
				},
			})
		}
		paths := a.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, common.DefaultDockerfileName)}
		p := transformertypes.Artifact{
			Name:  sImageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: sImageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:  sConfig.ServiceName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: a.Paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ImageNameConfigType: sImageName,
				artifacts.ServiceConfigType:   sConfig,
			},
		}
		ir := irtypes.IR{}
		if err = a.GetConfig(irtypes.IRConfigType, &ir); err == nil {
			dfs.Configs[irtypes.IRConfigType] = ir
		}
		createdArtifacts = append(createdArtifacts, p, dfs)
	}
	return pathMappings, createdArtifacts, nil
}
