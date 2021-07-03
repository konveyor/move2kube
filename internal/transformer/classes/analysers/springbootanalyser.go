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

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/konveyor/move2kube/types/source/springboot"
	"github.com/konveyor/move2kube/types/source/maven"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const pomXML string = "pom.xml"

const (
	springbootServiceConfigType transformertypes.ConfigType = "SpringbootService"
)

const (
	mavenPomXML         transformertypes.PathType = "MavenPomXML"
	applicationFilePath transformertypes.PathType = "SpringbootApplicationFile"
)

// SpringbootAnalyser implements Transformer interface
type SpringbootAnalyser struct {
	Config transformertypes.Transformer
	Env    environment.Environment
}

// SpringbootConfig defines SpringbootConfig properties
type SpringbootConfig struct {
	ServiceName string `yaml:"serviceName,omitempty"`
	Ports       []int  `yaml:"ports,omitempty"`
}

// SpringbootTemplateConfig defines SpringbootTemplateConfig properties
type SpringbootTemplateConfig struct {
	Port        int
	JavaVersion string
}

// Init Initializes the transformer
func (t *SpringbootAnalyser) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *SpringbootAnalyser) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *SpringbootAnalyser) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *SpringbootAnalyser) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	destEntries, err := ioutil.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Unable to process directory %s : %s", dir, err)
		return nil, nil, err
	}
	pomFound := false
	for _, de := range destEntries {
		if de.Name() == pomXML {
			pomFound = true
			break
		}
	}

	if !pomFound {
		return nil, nil, nil
	}

	// filled with previously declared xml
	pomStr, err := ioutil.ReadFile(filepath.Join(dir, pomXML))
	if err != nil {
		logrus.Errorf("Could not read the pom.xml file: %s", err)
		return nil, nil, err
	}

	// Load pom from string
	var pom maven.Pom
	if err := xml.Unmarshal([]byte(pomStr), &pom); err != nil {
		logrus.Errorf("unable to unmarshal pom file. Reason: %s", err)
		return nil, nil, err
	}

	// Dont process if this is a root pom and there are submodules
	if pom.Modules != nil && len(*(pom.Modules)) != 0 {
		logrus.Debugf("Ignoring pom at %s as it has modules", dir)
		return nil, nil, nil
	}

	// Check if at least there is one springboot dependency
	isSpringboot := false
	for _, dependency := range *pom.Dependencies {
		if strings.Contains(dependency.GroupID, "org.springframework.boot") {
			isSpringboot = true
		}
	}
	if !isSpringboot {
		logrus.Debugf("Ignoring pom at %s as does not contain Springboot dependencies: org.springframework.boot", dir)
		return nil, nil, nil
	}

	appfiles, err := common.GetFilesByName(dir, []string{"application.yaml", "application.yml"})
	if err != nil {
		logrus.Debugf("Cannot get application files: %s", err)
	}

	validSpringbootFiles := []string{}

	appName := filepath.Base(dir)
	ports := []int{}

	for _, appfile := range appfiles {
		var springApplicationYaml springboot.SpringApplicationYaml
		err = common.ReadYaml(appfile, &springApplicationYaml)
		if err != nil {
			logrus.Debugf("Could not load application file %s", appfile)
			continue
		}
		if (springApplicationYaml == springboot.SpringApplicationYaml{}) {
			logrus.Debugf("No information found in application file %s", appfile)
			continue
		}
		validSpringbootFiles = append(validSpringbootFiles, appfile)

		if springApplicationYaml.Spring.SpringApplication.Name != "" {
			appName = springApplicationYaml.Spring.SpringApplication.Name
		}

		if springApplicationYaml.Server.Port != 0 {
			ports = append(ports, springApplicationYaml.Server.Port)
		}
	}

	ct := plantypes.Transformer{
		Mode:                   transformertypes.ModeContainer,
		ArtifactTypes:          []transformertypes.ArtifactType{irtypes.IRArtifactType, artifacts.ContainerBuildArtifactType},
		ExclusiveArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
		Configs: map[transformertypes.ConfigType]interface{}{
			springbootServiceConfigType: SpringbootConfig{
				ServiceName: appName,
				Ports:       ports,
			}},
		Paths: map[transformertypes.PathType][]string{
			mavenPomXML:                   {filepath.Join(dir, pomXML)},
			artifacts.ProjectPathPathType: {dir},
			applicationFilePath:           validSpringbootFiles,
		},
	}
	return map[string]plantypes.Service{appName: []plantypes.Transformer{ct}}, nil, nil
}

// Transform transforms the artifacts
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
		err = a.GetConfig(springbootServiceConfigType, &sConfig)
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
			logrus.Errorf("Could not write the single generated Dockerfile template: %s", err)
		}

		port := 8080
		if len(sConfig.Ports) > 0 {
			port = sConfig.Ports[0]
		}

		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        outputPath,
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath, "Dockerfile."+seConfig.ServiceName),
			TemplateConfig: SpringbootTemplateConfig{Port: port},
		}, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			SrcPath:  "",
			DestPath: common.DefaultSourceDir,
		})

	}
	return pathMappings, nil, nil
}
