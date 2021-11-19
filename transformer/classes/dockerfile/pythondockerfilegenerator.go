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

package dockerfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	pythonExt           = ".py"
	pythonMain          = "main"
	pythonManage        = "manage"
	pythonService       = "python"
	djangoService       = "django"
	requirementsTxtFile = "requirements.txt"
	// MainPythonFilePathType points to the .py file path which contains main function
	MainPythonFilePathType transformertypes.PathType = "MainPythonFileRelPath"
)

var (
	pythonMainRegex = regexp.MustCompile(`__main__`)
)

// PythonDockerfileGenerator implements the Transformer interface
type PythonDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// PythonTemplateConfig implements python config interface
type PythonTemplateConfig struct {
	AppName           string
	Port              int32
	MainScriptRelPath string
	RequirementsTxt   string
	IsDjangoProject   bool
}

// Init Initializes the transformer
func (t *PythonDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *PythonDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// findMainScript returns the path of .py file having the main function
func findMainScript(pythonFilesPath []string) (string, error) {
	if len(pythonFilesPath) == 0 {
		return "", nil
	}
	for _, pythonFilePath := range pythonFilesPath {
		pythonFile, err := os.Open(pythonFilePath)
		if err != nil {
			logrus.Debugf("Failed to open the file %s", pythonFilePath)
		}
		scanner := bufio.NewScanner(pythonFile)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			if pythonMainRegex.MatchString(scanner.Text()) {
				return pythonFilePath, nil
			}
		}
	}
	return "", fmt.Errorf("could not find the main function in python files")
}

// DirectoryDetect runs detect in each sub directory
func (t *PythonDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	pythonFiles, err := common.GetFilesByExt(dir, []string{pythonExt})
	if err != nil {
		logrus.Errorf("Error while finding python files %s", err)
		return nil, nil
	}
	if len(pythonFiles) == 0 {
		return nil, nil
	}
	mainPythonFilePath, err := findMainScript(pythonFiles)
	if err != nil {
		return nil, err
	}
	// Search requirements.txt
	if mainPythonFilePath != "" {
		serviceName := strings.TrimSuffix(filepath.Base(mainPythonFilePath), pythonExt)
		if serviceName == pythonMain {
			serviceName = pythonService
		} else if serviceName == pythonManage {
			serviceName = djangoService
		}
		services = map[string][]transformertypes.Artifact{
			serviceName: {{
				Paths: map[string][]string{
					artifacts.ProjectPathPathType: {dir},
					MainPythonFilePathType:        {mainPythonFilePath},
				},
			}},
		}
		return services, nil
	}
	return nil, nil
}

// Transform transforms the artifacts
func (t *PythonDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		var pythonConfig PythonTemplateConfig
		pythonConfig.AppName = a.Name
		if a.Name == djangoService {
			pythonConfig.IsDjangoProject = true
		}
		mainPythonFileRelPath, err := filepath.Rel(a.Paths[artifacts.ProjectPathPathType][0], a.Paths[MainPythonFilePathType][0])
		if err != nil {
			logrus.Errorf("Error in getting the relative path %s", err)
		} else {
			pythonConfig.MainScriptRelPath = mainPythonFileRelPath
		}
		var detectedPorts []int32
		detectedPorts = append(detectedPorts, 8080) //TODO: Write parser to parse and identify port
		pythonConfig.Port = commonqa.GetPortForService(detectedPorts, a.Name)
		requirementsTxtFiles, err := common.GetFilesByName(a.Paths[artifacts.ProjectPathPathType][0], []string{requirementsTxtFile}, nil)
		if err != nil {
			logrus.Debugf("Cannot get the requirements.txt file: %s", err)
		}
		if len(requirementsTxtFiles) == 1 {
			requirementsTxt, err := filepath.Rel(a.Paths[artifacts.ProjectPathPathType][0], requirementsTxtFiles[0])
			if err != nil {
				logrus.Errorf("Error in getting the relative path %s", err)
			} else {
				pythonConfig.RequirementsTxt = requirementsTxt
			}
		}
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(sConfig.ServiceName)
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: pythonConfig,
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
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}
