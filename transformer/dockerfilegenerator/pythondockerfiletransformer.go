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
	"os"
	"path/filepath"
	"regexp"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
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

const (
	pythonExt           = ".py"
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

var (
	djangoRegex     = regexp.MustCompile(`(?m)^[Dd]jango`)
	pythonMainRegex = regexp.MustCompile(`^if\s+__name__\s*==\s*['"]__main__['"]\s*:\s*$`)
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
		return nil, nil
	}
	pythonMainFiles := []string{}
	for _, pythonFilePath := range pythonFilesPath {
		pythonFile, err := os.Open(pythonFilePath)
		if err != nil {
			logrus.Debugf("failed to open the file at path %s . Error: %q", pythonFilePath, err)
			continue
		}
		defer pythonFile.Close()
		scanner := bufio.NewScanner(pythonFile)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			if pythonMainRegex.MatchString(scanner.Text()) {
				pythonMainFiles = append(pythonMainFiles, pythonFilePath)
				break
			}
		}
		pythonFile.Close()
	}
	return pythonMainFiles, nil
}

// findDjangoDependency checks for django dependency in the requirements.txt file
func findDjangoDependency(reqTxtFilePath string) bool {
	reqTxtFile, err := os.ReadFile(reqTxtFilePath)
	if err != nil {
		logrus.Warnf("failed to read the file at path %s . Error: %q", reqTxtFilePath, err)
		return false
	}
	return djangoRegex.MatchString(string(reqTxtFile))
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
func (t *PythonDockerfileGenerator) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	pythonFilesPath, err := common.GetFilesByExtInCurrDir(dir, []string{pythonExt})
	if err != nil {
		return nil, fmt.Errorf("failed to look for python files in the directory %s . Error: %q", dir, err)
	}
	if len(pythonFilesPath) == 0 {
		return nil, nil
	}
	mainPythonFilesPath, err := findMainScripts(pythonFilesPath)
	if err != nil {
		logrus.Debugf("failed to find the python files in the project at directory %s that have a main function. Error: %q", dir, err)
	}
	serviceName := filepath.Base(dir)
	normalizedServiceName := common.MakeStringK8sServiceNameCompliant(serviceName)
	pythonService := transformertypes.Artifact{
		Paths: map[transformertypes.PathType][]string{
			artifacts.ServiceDirPathType: {dir},
			MainPythonFilesPathType:      mainPythonFilesPath,
			PythonFilesPathType:          pythonFilesPath,
		}, Configs: map[transformertypes.ConfigType]interface{}{
			artifacts.OriginalNameConfigType: artifacts.OriginalNameConfig{OriginalName: serviceName},
		},
	}

	// check if it is a Django project
	pythonService.Configs[PythonServiceConfigType] = PythonConfig{IsDjango: false}
	requirementsTxtFiles, err := common.GetFilesInCurrentDirectory(dir, []string{requirementsTxtFile}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to look for the requirements.txt file in the directory %s . Error: %q", dir, err)
	}
	if len(requirementsTxtFiles) > 0 {
		requirementsTxtPath := requirementsTxtFiles[0]
		pythonService.Paths[RequirementsTxtPathType] = []string{requirementsTxtPath}
		pythonService.Configs[PythonServiceConfigType] = PythonConfig{IsDjango: findDjangoDependency(requirementsTxtPath)}
	}
	return map[string][]transformertypes.Artifact{normalizedServiceName: {pythonService}}, nil
}

// Transform transforms the artifacts
func (t *PythonDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 {
			logrus.Errorf("the service directory is missing from the artifact: %+v", newArtifact)
			continue
		}
		serviceDir := newArtifact.Paths[artifacts.ServiceDirPathType][0]
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), serviceDir)
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", serviceDir, err)
			continue
		}
		serviceConfig := artifacts.ServiceConfig{}
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &serviceConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", serviceConfig, err)
			continue
		}
		imageName := artifacts.ImageName{}
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", imageName, err)
		}
		if imageName.ImageName == "" {
			imageName.ImageName = common.MakeStringContainerImageNameCompliant(serviceConfig.ServiceName)
		}
		ir := irtypes.IR{}
		irPresent := true
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			irPresent = false
			logrus.Debugf("unable to load config for Transformer into %T : %s", ir, err)
		}
		var pythonTemplateConfig PythonTemplateConfig
		if len(newArtifact.Paths[MainPythonFilesPathType]) > 0 {
			pythonTemplateConfig.StartingScriptRelPath = getMainPythonFileForService(newArtifact.Paths[MainPythonFilesPathType], serviceDir, newArtifact.Name)
		} else {
			pythonTemplateConfig.StartingScriptRelPath = getStartingPythonFileForService(newArtifact.Paths[PythonFilesPathType], serviceDir, newArtifact.Name)
		}
		pythonTemplateConfig.AppName = newArtifact.Name
		var pythonConfig PythonConfig
		err = newArtifact.GetConfig(PythonServiceConfigType, &pythonConfig)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", imageName, err)
		}
		pythonTemplateConfig.IsDjango = pythonConfig.IsDjango
		ports := ir.GetAllServicePorts()
		if len(ports) == 0 {
			ports = []int32{common.DefaultServicePort}
		}
		pythonTemplateConfig.Port = commonqa.GetPortForService(ports, `"`+newArtifact.Name+`"`)
		if len(newArtifact.Paths[artifacts.ServiceDirPathType]) == 0 {
			logrus.Errorf("The service directory path is missing for the artifact: %+v", newArtifact)
			continue
		}
		if len(newArtifact.Paths[RequirementsTxtPathType]) != 0 {
			if requirementsTxt, err := filepath.Rel(serviceDir, newArtifact.Paths[RequirementsTxtPathType][0]); err == nil {
				pythonTemplateConfig.RequirementsTxt = requirementsTxt
			}
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
		paths := newArtifact.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, common.DefaultDockerfileName)}
		p := transformertypes.Artifact{
			Name:  imageName.ImageName,
			Type:  artifacts.DockerfileArtifactType,
			Paths: paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ServiceConfigType:   serviceConfig,
				artifacts.ImageNameConfigType: imageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:  serviceConfig.ServiceName,
			Type:  artifacts.DockerfileForServiceArtifactType,
			Paths: newArtifact.Paths,
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.ServiceConfigType:   serviceConfig,
				artifacts.ImageNameConfigType: imageName,
			},
		}
		if irPresent {
			dfs.Configs[irtypes.IRConfigType] = ir
		}
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}
