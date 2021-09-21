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
	"io/ioutil"

	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// SpringbootRuntimeAnalyser implements Transformer interface
type SpringbootRuntimeAnalyser struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *SpringbootRuntimeAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *SpringbootRuntimeAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *SpringbootRuntimeAnalyser) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

func (t *SpringbootRuntimeAnalyser) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// Transform transforms the artifacts
func (t *SpringbootRuntimeAnalyser) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {

		if a.Artifact != artifacts.BuildType {
			continue
		}

		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}

		/*
			var sConfig SpringbootConfig
			err = a.GetConfig(springbootServiceConfigType, &sConfig)
			if err != nil {
				logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
				continue
			}
		*/
		/*
			var seConfig artifacts.ServiceConfig
			err = a.GetConfig(artifacts.ServiceConfigType, &seConfig)
			if err != nil {
				logrus.Errorf("unable to load config for Transformer into %T : %s", seConfig, err)
				continue
			}
		*/
		/*
			sImageName := artifacts.ImageName{}
			err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
			if err != nil {
				logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
			}
			if sImageName.ImageName == "" {
				sImageName.ImageName = common.MakeStringContainerImageNameCompliant(a.Name)
			}
		*/

		templateConfig := SpringbootTemplateConfig{}
		err = a.GetConfig("targetAppData", &templateConfig)
		if err != nil {
			logrus.Debugf("unable to load config %s : %s", "targetAppData", err)
		}

		/*
			fromBuildConfig := map[string]string{}
			err = a.GetConfig("fromBuild", &fromBuildConfig)
			if err != nil {
				logrus.Debugf("unable to load config %s : %s", "fromBuild", err)
			}
		*/
		// We need to read the info from the build phase:

		strBuild, err := ioutil.ReadFile(templateConfig.BuildOutputPath)
		if err != nil {
			logrus.Debugf("build location %s", templateConfig.BuildOutputPath)
			return nil, nil, err
		}

		// Runtime

		runtimeSegment := "Dockerfile.springboot-embedded" // default
		if templateConfig.AppServer == "jboss/wildfly" {
			runtimeSegment = "Dockerfile.springboot-wildfly-jboss-runtime"
		} else if templateConfig.AppServer == "openliberty/open-liberty" {
			runtimeSegment = "Dockerfile.springboot-open-liberty-runtime"
		}

		strRuntime, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, runtimeSegment))
		if err != nil {
			return nil, nil, err
		}

		var outputPath = filepath.Join(t.Env.TempPath, "Dockerfile.template")
		template := string(strBuild) + "\n" + string(strRuntime)

		err = ioutil.WriteFile(outputPath, []byte(template), 0644)
		if err != nil {
			logrus.Errorf("Could not write the single generated Dockerfile template: %s", err)
		}

		port := 8080
		if templateConfig.Port != 0 {
			port = templateConfig.Port
		}

		dfp := filepath.Join(common.DefaultSourceDir, relSrcPath, "Dockerfile")

		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.TemplatePathMappingType,
			SrcPath:  outputPath,
			DestPath: dfp,
			TemplateConfig: SpringbootTemplateConfig{
				JavaPackageName: templateConfig.JavaPackageName,
				AppServerImage:  templateConfig.AppServerImage,
				Port:            port,
				AppFile:         templateConfig.AppFile,
				DeploymentFile:  templateConfig.DeploymentFile,
			},
		}, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			SrcPath:  "",
			DestPath: common.DefaultSourceDir,
		})

		/*
			// not using it
			p := transformertypes.Artifact{
				Name:     sImageName.ImageName,
				Artifact: artifacts.DockerfileArtifactType,
				Paths: map[string][]string{
					artifacts.ProjectPathPathType: {filepath.Dir(dfp)},
					artifacts.DockerfilePathType:  {dfp},
				},
				Configs: map[string]interface{}{
					artifacts.ImageNameConfigType: sImageName,
				},
			}


			// not using it
			dfs := transformertypes.Artifact{
				Name:     sConfig.ServiceName,
				Artifact: artifacts.DockerfileForServiceArtifactType,
				Paths: map[string][]string{
					artifacts.ProjectPathPathType: {filepath.Dir(dfp)},
					artifacts.DockerfilePathType:  {dfp},
				},
				Configs: map[string]interface{}{
					artifacts.ImageNameConfigType: sImageName,
					artifacts.ServiceConfigType:   sConfig,
				},
			}
		*/

		//new
		/*
			buildArtifact := transformertypes.Artifact{
				Name:     sImageName.ImageName + "-build",
				Artifact: artifacts.DockerfileArtifactType,
				//In here we store the current path of the current Dockerfile template
				Paths: map[string][]string{
					artifacts.ProjectPathPathType: {filepath.Dir(dfp)},
					artifacts.DockerfilePathType:  {dfp},
				},
				Configs: map[string]interface{}{
					"targetAppData": SpringbootTemplateConfig{
						JavaPackageName: sConfig.JavaPackageName,
						AppServerImage:  sConfig.ApplicationServerImage,
						Port:            port,
						AppFile:         sConfig.AppFile,
						DeploymentFile:  sConfig.DeploymentFile,
					},
				},
			}
		*/

		/*
			runtimeDataArtifact := transformertypes.Artifact{
				Name:     "test-runtime",
				Artifact: artifacts.DockerfileArtifactType,
				//In here we store the current path of the current Dockerfile template

				Configs: map[string]interface{}{
					"targetAppData": SpringbootTemplateConfig{
						JavaPackageName: sConfig.JavaPackageName,
						AppServerImage:  sConfig.ApplicationServerImage,
						Port:            port,
						AppFile:         sConfig.AppFile,
						DeploymentFile:  sConfig.DeploymentFile,
					},
				},
			}

			//createdArtifacts = append(createdArtifacts, p, dfs) // original
			createdArtifacts = append(createdArtifacts, runtimeDataArtifact)
		*/

	}

	return pathMappings, createdArtifacts, nil

	// changes:
	// - just artifacts: jar/war/ + params to run it
}
