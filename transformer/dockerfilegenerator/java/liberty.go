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
	JavaPackageName                   string
	JavaVersion                       string
	DeploymentFile                    string
	BuildContainerName                string
	DeploymentFileDirInBuildContainer string
	Port                              int32
	EnvVariables                      map[string]string
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
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(sConfig.ServiceName)
		}
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ServiceDirPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ServiceDirPathType][0], err)
		}
		libertyRunDockerfile, err := os.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.liberty"))
		if err != nil {
			logrus.Errorf("Unable to read Dockerfile liberty template : %s", err)
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
		template := string(dockerFileHead) + "\n" + string(libertyRunDockerfile)
		err = os.WriteFile(dockerfileTemplate, []byte(template), common.DefaultFilePermission)
		if err != nil {
			logrus.Errorf("Could not write the generated Build Dockerfile template: %s", err)
		}
		dft := LibertyDockerfileTemplate{}
		libertyArtifactConfig := artifacts.WarArtifactConfig{}
		err = a.GetConfig(artifacts.WarConfigType, &libertyArtifactConfig)
		if err != nil {
			// EAR
			logrus.Debugf("unable to load config for Transformer into %T : %s", libertyArtifactConfig, err)
			libertyEarArtifactConfig := artifacts.EarArtifactConfig{}
			err = a.GetConfig(artifacts.EarConfigType, &libertyEarArtifactConfig)
			if err != nil {
				logrus.Debugf("unable to load config for Transformer into %T : %s", libertyEarArtifactConfig, err)
			}
			if libertyEarArtifactConfig.JavaVersion == "" {
				libertyEarArtifactConfig.JavaVersion = t.LibertyConfig.JavaVersion
			}
			javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), libertyEarArtifactConfig.JavaVersion)
			if err != nil {
				logrus.Errorf("Unable to find mapping version for java version %s : %s", libertyEarArtifactConfig.JavaVersion, err)
				javaPackage = defaultJavaPackage
			}
			dft.JavaPackageName = javaPackage
			dft.JavaVersion = libertyEarArtifactConfig.JavaVersion
			dft.DeploymentFile = libertyEarArtifactConfig.DeploymentFile
			dft.Port = defaultLibertyPort
			dft.EnvVariables = libertyEarArtifactConfig.EnvVariables
			if isBuildContainerPresent {
				dft.BuildContainerName = libertyEarArtifactConfig.BuildContainerName
				dft.DeploymentFileDirInBuildContainer = libertyEarArtifactConfig.DeploymentFileDirInBuildContainer
			}
		} else {
			// WAR
			if libertyArtifactConfig.JavaVersion == "" {
				libertyArtifactConfig.JavaVersion = t.LibertyConfig.JavaVersion
			}
			javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), versionMappingFilePath), libertyArtifactConfig.JavaVersion)
			if err != nil {
				logrus.Errorf("Unable to find mapping version for java version %s : %s", libertyArtifactConfig.JavaVersion, err)
				javaPackage = defaultJavaPackage
			}
			dft.JavaPackageName = javaPackage
			dft.JavaVersion = libertyArtifactConfig.JavaVersion
			dft.Port = defaultLibertyPort
			dft.EnvVariables = libertyArtifactConfig.EnvVariables
			dft.DeploymentFile = libertyArtifactConfig.DeploymentFile
			if isBuildContainerPresent {
				dft.DeploymentFile = libertyArtifactConfig.DeploymentFile
				dft.BuildContainerName = libertyArtifactConfig.BuildContainerName
				dft.DeploymentFileDirInBuildContainer = libertyArtifactConfig.DeploymentFileDirInBuildContainer
			}
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		})
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        dockerfileTemplate,
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: dft,
		})
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
