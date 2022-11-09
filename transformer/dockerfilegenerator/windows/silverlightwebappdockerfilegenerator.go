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

package windows

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	dotnetutils "github.com/konveyor/move2kube/transformer/dockerfilegenerator/dotnet"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/konveyor/move2kube/types/source/dotnet"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// SilverLightTemplateConfig implements SilverLight config interface
type SilverLightTemplateConfig struct {
	Ports   []int32
	AppName string
}

// WinSilverLightWebAppDockerfileGenerator implements the Transformer interface
type WinSilverLightWebAppDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *WinSilverLightWebAppDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *WinSilverLightWebAppDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *WinSilverLightWebAppDockerfileGenerator) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	slnPaths, err := common.GetFilesByExtInCurrDir(dir, []string{dotnet.VISUAL_STUDIO_SOLUTION_FILE_EXT})
	if err != nil {
		return nil, fmt.Errorf("failed to list the dot net visual studio solution files in the directory %s . Error: %q", dir, err)
	}
	if len(slnPaths) == 0 {
		return nil, nil
	}
	if len(slnPaths) > 1 {
		logrus.Debugf("more than one visual studio solution file detected. Number of .sln files %d", len(slnPaths))
	}
	slnPath := slnPaths[0]
	appName := dotnetutils.GetParentProjectName(slnPath)
	normalizedAppName := common.MakeStringK8sServiceNameCompliant(appName)

	relCSProjPaths, err := dotnetutils.GetCSProjPathsFromSlnFile(slnPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get the .csproj paths from .sln file at path %s . Error: %q", slnPath, err)
	}

	if len(relCSProjPaths) == 0 {
		return nil, fmt.Errorf("no c sharp projects available found in the.sln path %s", slnPath)
	}

	found := false
	for _, relCSProjPath := range relCSProjPaths {
		csProjPath := filepath.Join(dir, strings.TrimSpace(relCSProjPath))
		csProjBytes, err := os.ReadFile(csProjPath)
		if err != nil {
			logrus.Errorf("failed to read the c sharp project file at path %s . Error: %q", csProjPath, err)
			continue
		}

		configuration := dotnet.CSProj{}
		if err := xml.Unmarshal(csProjBytes, &configuration); err != nil {
			logrus.Errorf("failed to parse the project file at path %s . Error: %q", csProjPath, err)
			continue
		}

		idx := common.FindIndex(configuration.PropertyGroups, func(x dotnet.PropertyGroup) bool { return x.TargetFrameworkVersion != "" })
		if idx == -1 {
			logrus.Debugf("failed to find the target framework in any of the property groups inside the c sharp project file at path %s", csProjPath)
			continue
		}
		targetFrameworkVersion := configuration.PropertyGroups[idx].TargetFrameworkVersion
		if !dotnet.Version4And3_5.MatchString(targetFrameworkVersion) {
			logrus.Errorf("silverlight dot net tranformer: the c sharp project file at path %s does not have a supported framework version. Actual version: %s", csProjPath, targetFrameworkVersion)
			continue
		}

		isWebProj, err := isWeb(configuration)
		if err != nil {
			logrus.Errorf("failed to detect if it's a web/asp net project. Error: %q", err)
			continue
		}
		if !isWebProj {
			continue
		}

		isSLProj, err := isSilverlight(configuration)
		if err != nil {
			logrus.Errorf("failed to detect if it's a silverlight project. Error: %q", err)
			continue
		}
		if !isSLProj {
			continue
		}

		found = true
	}

	if !found {
		return nil, nil
	}

	services := map[string][]transformertypes.Artifact{
		normalizedAppName: {{
			Paths: map[transformertypes.PathType][]string{
				artifacts.ServiceDirPathType: {dir},
			},
			Configs: map[transformertypes.ConfigType]interface{}{
				artifacts.OriginalNameConfigType: artifacts.OriginalNameConfig{OriginalName: appName},
			},
		}},
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *WinSilverLightWebAppDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
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
		detectedPorts := ir.GetAllServicePorts()
		if len(detectedPorts) == 0 {
			detectedPorts = append(detectedPorts, common.DefaultServicePort)
		}
		detectedPorts = commonqa.GetPortsForService(detectedPorts, `"`+a.Name+`"`)
		var silverLightConfig SilverLightTemplateConfig
		silverLightConfig.AppName = a.Name
		silverLightConfig.Ports = detectedPorts
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
			TemplateConfig: silverLightConfig,
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
