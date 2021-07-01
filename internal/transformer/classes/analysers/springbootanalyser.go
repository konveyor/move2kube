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

package analysers

import (
	"encoding/xml"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/creekorful/mvnparser"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"

	//"github.com/konveyor/move2kube/internal/transformer/classes/analysers/compose"
	//collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	SpringbootServiceConfigType transformertypes.ConfigType = "SpringbootService"
)

const (
	// composeFilePathType defines the source artifact type of Docker compose
	mavenPomXML transformertypes.PathType = "MavenPomXML"
	// imageInfoPathType defines the source artifact type of image info
	//imageInfoPathType transformertypes.PathType = "ImageInfo"
)

// SpringbootAnalyser implements Transformer interface
type SpringbootAnalyser struct {
	Config transformertypes.Transformer
	Env    environment.Environment
}

type SpringbootConfig struct {
	ServiceName string `yaml:"serviceName,omitempty"`
}

type SpringbootTemplateConfig struct {
	Ports       []int
	JavaVersion string
}

type Application struct {
	Name string `yaml:"name"`
}
type SpringSpec struct {
	Application Application `yaml:"application"`
}

type SpringBootSpec struct {
	SpringSpec SpringSpec `yaml:"spring"`
	ServerSpec ServerSpec `yaml:"server"`
}

type ServerSpec struct {
	Port int `yaml:"port"`
}

func (t *SpringbootAnalyser) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

func (t *SpringbootAnalyser) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.Config, t.Env
}

func (t *SpringbootAnalyser) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {

	return nil, nil, nil
}

func (t *SpringbootAnalyser) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {

	destEntries, err := ioutil.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Unable to process directory %s : %s", dir, err)
	} else {
		for _, de := range destEntries {
			if de.Name() == "pom.xml" {

				// filled with previously declared xml
				pomStr, err := ioutil.ReadFile(filepath.Join(dir, de.Name()))
				if err != nil {
					return nil, nil, err
				}

				// Load project from string
				var project mvnparser.MavenProject
				if err := xml.Unmarshal([]byte(pomStr), &project); err != nil {
					logrus.Errorf("unable to unmarshal pom file. Reason: %s", err)
					return nil, nil, err
				}

				// Dont process if this is a root pom and there are submodules
				if len(project.Modules) != 0 {
					return nil, nil, nil
				}

				// Check if at least there is one springboot dependency
				isSpringboot := false
				for _, dependency := range project.Dependencies {
					if strings.Contains(dependency.GroupId, "org.springframework.boot") {
						isSpringboot = true
					}
				}
				if !isSpringboot {
					return nil, nil, nil
				}

				ct := plantypes.Transformer{
					Mode:                   plantypes.ModeContainer,
					ArtifactTypes:          []transformertypes.ArtifactType{irtypes.IRArtifactType, artifacts.ContainerBuildArtifactType},
					ExclusiveArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
					Configs: map[transformertypes.ConfigType]interface{}{
						SpringbootServiceConfigType: SpringbootConfig{
							ServiceName: filepath.Base(dir),
						}},
					Paths: map[transformertypes.PathType][]string{
						mavenPomXML: {
							filepath.Join(dir, "pom.xml"),
						},
						artifacts.ProjectPathPathType: {dir},
					},
				}
				return map[string]plantypes.Service{filepath.Base(dir): []plantypes.Transformer{ct}}, nil, nil
			}
		}
	}
	return nil, nil, nil
}

func (t *SpringbootAnalyser) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {

	pathMappings := []transformertypes.PathMapping{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}

		relSrcPath, err := filepath.Rel(t.Env.GetWorkspaceSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}

		var pConfig artifacts.PlanConfig
		err = a.GetConfig(artifacts.PlanConfigType, &pConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", pConfig, err)
			continue
		}
		var sConfig SpringbootConfig
		err = a.GetConfig(SpringbootServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}

		var seConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &seConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", seConfig, err)
			continue
		}

		// License
		strLicense, err := ioutil.ReadFile(filepath.Join(t.Env.Context, t.Env.RelTemplatesDir, "Dockerfile.license"))
		if err != nil {
			return nil, nil, err
		}

		// Build
		strBuild, err := ioutil.ReadFile(filepath.Join(t.Env.Context, t.Env.RelTemplatesDir, "Dockerfile.maven-build"))
		if err != nil {
			return nil, nil, err
		}

		// Runtime
		strEmbedded, err := ioutil.ReadFile(filepath.Join(t.Env.Context, t.Env.RelTemplatesDir, "Dockerfile.springboot-embedded"))
		if err != nil {
			return nil, nil, err
		}

		var outputPath = filepath.Join(t.Env.TempPath, "Dockerfile.template")

		template := string(strLicense) + "\n" + string(strBuild) + "\n" + string(strEmbedded)

		err = ioutil.WriteFile(outputPath, []byte(template), 0644)
		if err != nil {
			logrus.Errorf("error:", err)
		}

		dir := a.Paths[artifacts.ProjectPathPathType][0]

		// capture port and app name
		var stc SpringbootTemplateConfig
		yamlpaths, err := common.GetFilesByExt(dir, []string{".yaml", ".yml"})
		if err != nil {
			logrus.Errorf("Cannot get yaml files", err)

		}
		for _, path := range yamlpaths {
			sb := SpringBootSpec{}
			if err := common.ReadYaml(path, &sb); err != nil {
				continue
			}

			stc.Ports = []int{sb.ServerSpec.Port}
			break
		}

		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        outputPath,
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath, "Dockerfile."+seConfig.ServiceName),
			TemplateConfig: stc,
		}, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			SrcPath:  "",
			DestPath: common.DefaultSourceDir,
		})

	}
	return pathMappings, nil, nil
}
