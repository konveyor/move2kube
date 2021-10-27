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
	"io/ioutil"
	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
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

type JarYamlConfig struct {
	JavaVersion string `yaml:"defaultJavaVersion"`
}

type JarDockerfileTemplate struct {
	JavaVersion          string
	JavaPackageName      string
	DeploymentFile       string
	DeploymentFileDir    string
	Port                 string
	PortConfigureEnvName string
}

// Init Initializes the transformer
func (t *JarAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.JarConfig = &JarYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, &t.JarConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.JarConfig, err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *JarAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *JarAnalyser) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *JarAnalyser) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	unnamedServices = []transformertypes.TransformerPlan{}
	jarFilePaths, err := common.GetFilesInCurrentDirectory(dir, nil, []string{".*[.]jar"})
	if err != nil {
		logrus.Errorf("Error while parsing directory %s for jar file : %s", dir, err)
		return nil, nil, err
	}
	if len(jarFilePaths) == 0 {
		return nil, nil, nil
	}
	for _, path := range jarFilePaths {
		unnamedServices = append(unnamedServices, transformertypes.TransformerPlan{
			Mode:              transformertypes.ModeContainer,
			ArtifactTypes:     []transformertypes.ArtifactType{irtypes.IRArtifactType, artifacts.ContainerBuildArtifactType},
			BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			Configs:           map[transformertypes.ConfigType]interface{}{},
			Paths: map[transformertypes.PathType][]string{
				artifacts.JarPathType:         {path},
				artifacts.ProjectPathPathType: {path},
			},
		})
	}
	return
}

// Transform transforms the artifacts
func (t *JarAnalyser) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
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
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}
		jarRunDockerfile, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.embedded"))
		if err != nil {
			logrus.Errorf("Unable to read Dockerfile embedded template : %s", err)
		}
		dockerfileBuildDockerfile := a.Paths[artifacts.BuildContainerFileType][0]
		buildDockerfile, err := ioutil.ReadFile(dockerfileBuildDockerfile)
		if err != nil {
			logrus.Errorf("Unable to read build Dockerfile template : %s", err)
		}
		dockerfileTemplate := filepath.Join(t.Env.TempPath, "Dockerfile")
		template := string(buildDockerfile) + "\n" + string(jarRunDockerfile)
		err = ioutil.WriteFile(dockerfileTemplate, []byte(template), common.DefaultFilePermission)
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
		javaPackage, err := getJavaPackage(filepath.Join(t.Env.GetEnvironmentContext(), "mappings/javapackageversions.yaml"), jarArtifactConfig.JavaVersion)
		if err != nil {
			logrus.Error("Unable to find mapping version for java version %s : %s", jarArtifactConfig.JavaVersion, err)
			javaPackage = "java-1.8.0-openjdk-devel"
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:     transformertypes.TemplatePathMappingType,
			SrcPath:  dockerfileTemplate,
			DestPath: filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: JarDockerfileTemplate{
				JavaVersion:          jarArtifactConfig.JavaVersion,
				JavaPackageName:      javaPackage,
				DeploymentFile:       jarArtifactConfig.DeploymentFile,
				DeploymentFileDir:    jarArtifactConfig.DeploymentFileDir,
				Port:                 "8080",
				PortConfigureEnvName: "SERVER_PORT",
			},
		})
		paths := a.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, "Dockerfile")}
		p := transformertypes.Artifact{
			Name:     sImageName.ImageName,
			Artifact: artifacts.DockerfileArtifactType,
			Paths:    paths,
			Configs: map[string]interface{}{
				artifacts.ImageNameConfigType: sImageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:     sConfig.ServiceName,
			Artifact: artifacts.DockerfileForServiceArtifactType,
			Paths:    a.Paths,
			Configs: map[string]interface{}{
				artifacts.ImageNameConfigType: sImageName,
				artifacts.ServiceConfigType:   sConfig,
			},
		}
		createdArtifacts = append(createdArtifacts, p, dfs)
	}
	return pathMappings, createdArtifacts, nil
}
