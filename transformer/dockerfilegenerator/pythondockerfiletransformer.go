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

package dockerfilegenerator

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	pythonExt           = ".py"
	django              = "django"
	requirementsTxtFile = "requirements.txt"
	//RequirementsTxtPathType points to the requirements.txt file if it's present
	RequirementsTxtPathType transformertypes.PathType = "RequirementsTxtPathType"
	// PythonServiceConfigType points to python config
	PythonServiceConfigType transformertypes.ConfigType = "PythonConfig"
	// MainPythonFilesPathType points to the .py files path which contain main function
	MainPythonFilesPathType transformertypes.PathType = "MainPythonFilesPathType"
	// PythonFilesPathType points to the .py files path
	PythonFilesPathType transformertypes.PathType = "PythonFilesPathType"
)

// PythonDockerfileGenerator implements the Transformer interface
type PythonDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// PythonTemplateConfig implements python config interface
type PythonTemplateConfig struct {
	AppName               string
	Port                  int32
	StartingScriptRelPath string
	RequirementsTxt       string
	IsDjango              bool
}

// PythonConfig implements python config interface
type PythonConfig struct {
	IsDjango bool `json:"IsDjango" yaml:"IsDjango"`
}

var (
	pythonMainRegex = regexp.MustCompile(`^if\s+__name__\s*==\s*['"]__main__['"]\s*:\s+$`)
)

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

// findMainScripts returns the path of .py files having the main function
func findMainScripts(pythonFilesPath []string) ([]string, error) {
	if len(pythonFilesPath) == 0 {
		return []string{}, nil
	}
	var pythonMainFiles []string
	for _, pythonFilePath := range pythonFilesPath {
		pythonFile, err := os.Open(pythonFilePath)
		if err != nil {
			logrus.Debugf("Failed to open the file %s", pythonFilePath)
		}
		scanner := bufio.NewScanner(pythonFile)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			if pythonMainRegex.MatchString(scanner.Text()) {
				pythonMainFiles = append(pythonMainFiles, pythonFilePath)
			}
		}
	}
	if len(pythonMainFiles) == 0 {
		return []string{}, fmt.Errorf("could not find the main function in python files")
	}
	return pythonMainFiles, nil
}

// findDjangoDependency checks for django dependency in the requirements.txt file
func findDjangoDependency(reqTxtFilePath string) bool {
	reqTxtFile, err := ioutil.ReadFile(reqTxtFilePath)
	if err != nil {
		logrus.Warnf("Error in reading the file %s: %s.", reqTxtFilePath, err)
		return false
	}
	return strings.Contains(string(reqTxtFile), django)
}

// getMainPythonFileForService returns the main file used by a service
func getMainPythonFileForService(mainPythonFilesPath []string, baseDir string, serviceName string) string {
	var mainPythonFilesRelPath []string
	for _, mainPythonFilePath := range mainPythonFilesPath {
		if mainPythonFileRelPath, err := filepath.Rel(baseDir, mainPythonFilePath); err == nil {
			mainPythonFilesRelPath = append(mainPythonFilesRelPath, mainPythonFileRelPath)
		}
	}
	quesKey := common.JoinQASubKeys(common.ConfigServicesKey, `"`+serviceName+`"`, common.ConfigMainPythonFileForServiceKeySegment)
	desc := fmt.Sprintf("Select the main file to be used for the service %s :", serviceName)
	hints := []string{fmt.Sprintf("Selected main file will be used for the service %s", serviceName)}
	return qaengine.FetchSelectAnswer(quesKey, desc, hints, mainPythonFilesRelPath[0], mainPythonFilesRelPath)
}

// getStartingPythonFileForService returns the starting python file used by a service
func getStartingPythonFileForService(pythonFilesPath []string, baseDir string, serviceName string) string {
	var pythonFilesRelPath []string
	for _, pythonFilePath := range pythonFilesPath {
		if pythonFileRelPath, err := filepath.Rel(baseDir, pythonFilePath); err == nil {
			pythonFilesRelPath = append(pythonFilesRelPath, pythonFileRelPath)
		}
	}
	quesKey := common.JoinQASubKeys(common.ConfigServicesKey, `"`+serviceName+`"`, common.ConfigStartingPythonFileForServiceKeySegment)
	desc := fmt.Sprintf("Select the python file to be used for the service %s :", serviceName)
	hints := []string{fmt.Sprintf("Selected python file will be used for starting the service %s", serviceName)}
	return qaengine.FetchSelectAnswer(quesKey, desc, hints, pythonFilesRelPath[0], pythonFilesRelPath)
}

// DirectoryDetect runs detect in each sub directory
func (t *PythonDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	pythonFilesPath, err := common.GetFilesByExtInCurrDir(dir, []string{pythonExt})
	if err != nil {
		logrus.Errorf("Error while finding python files %s", err)
		return nil, nil
	}
	if len(pythonFilesPath) == 0 {
		return nil, nil
	}
	mainPythonFilesPath, err := findMainScripts(pythonFilesPath)
	if err == nil || len(pythonFilesPath) > 0 {
		var serviceName, requirementsTxtPath string
		var isDjango bool
		requirementsTxtFiles, err := common.GetFilesInCurrentDirectory(dir, []string{requirementsTxtFile}, nil)
		if err != nil {
			logrus.Debugf("Cannot get the requirements.txt file: %s", err)
		}
		if len(requirementsTxtFiles) == 1 {
			requirementsTxtPath = requirementsTxtFiles[0]
			isDjango = findDjangoDependency(requirementsTxtFiles[0])
		}
		pythonService := transformertypes.Artifact{
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceDirPathType: {dir},
				MainPythonFilesPathType:      mainPythonFilesPath,
				PythonFilesPathType:          pythonFilesPath,
			}, Configs: map[transformertypes.ConfigType]interface{}{
				PythonServiceConfigType: PythonConfig{
					IsDjango: isDjango,
				},
			},
		}
		if requirementsTxtPath != "" {
			pythonService.Paths[RequirementsTxtPathType] = []string{requirementsTxtPath}
		}
		services = map[string][]transformertypes.Artifact{
			serviceName: {pythonService},
		}
		return services, nil
	}
	return nil, nil
}

// Transform transforms the artifacts
func (t *PythonDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ServiceDirPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ServiceDirPathType][0], err)
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
		ir := irtypes.IR{}
		irPresent := true
		err = a.GetConfig(irtypes.IRConfigType, &ir)
		if err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T : %s", ir, err)
		}
		var pythonTemplateConfig PythonTemplateConfig
		if len(a.Paths[MainPythonFilesPathType]) > 0 {
			pythonTemplateConfig.StartingScriptRelPath = getMainPythonFileForService(a.Paths[MainPythonFilesPathType], a.Paths[artifacts.ServiceDirPathType][0], a.Name)
		} else {
			pythonTemplateConfig.StartingScriptRelPath = getStartingPythonFileForService(a.Paths[PythonFilesPathType], a.Paths[artifacts.ServiceDirPathType][0], a.Name)
		}
		pythonTemplateConfig.AppName = a.Name
		var pythonConfig PythonConfig
		err = a.GetConfig(PythonServiceConfigType, &pythonConfig)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		pythonTemplateConfig.IsDjango = pythonConfig.IsDjango
		ports := ir.GetAllServicePorts()
		if len(ports) == 0 {
			ports = []int32{common.DefaultServicePort}
		}
		pythonTemplateConfig.Port = commonqa.GetPortForService(ports, `"`+a.Name+`"`)
		if len(a.Paths[artifacts.ServiceDirPathType]) == 0 {
			logrus.Errorf("The service directory path is missing for the artifact: %+v", a)
			continue
		}
		if len(a.Paths[RequirementsTxtPathType]) != 0 {
			if requirementsTxt, err := filepath.Rel(a.Paths[artifacts.ServiceDirPathType][0], a.Paths[RequirementsTxtPathType][0]); err == nil {
				pythonTemplateConfig.RequirementsTxt = requirementsTxt
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
			TemplateConfig: pythonTemplateConfig,
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
		if irPresent {
			dfs.Configs[irtypes.IRConfigType] = ir
		}
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}
