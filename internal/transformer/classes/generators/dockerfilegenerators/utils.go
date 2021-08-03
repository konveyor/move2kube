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

package dockerfilegenerators

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types/source/dotnet"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

func isSilverlight(configuration dotnet.CSProj) (bool, error) {
	if configuration.ItemGroups == nil || len(configuration.ItemGroups) == 0 {
		return false, fmt.Errorf("No item groups in project file to parse")
	}

	for _, ig := range configuration.ItemGroups {
		if ig.Contents == nil || len(ig.Contents) == 0 {
			continue
		}

		for _, r := range ig.Contents {
			if dotnet.WebSLLib.MatchString(r.Include) {
				return true, nil
			}
		}
	}

	return false, nil
}

func isWeb(configuration dotnet.CSProj) (bool, error) {
	if configuration.ItemGroups == nil || len(configuration.ItemGroups) == 0 {
		return false, fmt.Errorf("No item groups in project file to parse")
	}

	for _, ig := range configuration.ItemGroups {
		if ig.References == nil || len(ig.References) == 0 {
			continue
		}

		for _, r := range ig.References {
			if dotnet.WebLib.MatchString(r.Include) {
				return true, nil
			}
		}
	}

	return false, nil
}

func parseSolutionFile(inputPath string) []string {
	solFile, err := os.Open(inputPath)
	if err != nil {
		logrus.Errorf("Could not open the solution file: %s", err)
		return nil
	}
	defer solFile.Close()

	solFileScanner := bufio.NewScanner(solFile)

	r := regexp.MustCompile(dotnet.ProjBlock)

	csr := regexp.MustCompile(dotnet.CsProj)

	projectPaths := make([]string, 0)
	for solFileScanner.Scan() {
		s := solFileScanner.Text()
		if r.MatchString(s) {
			tokens := strings.Split(s, "=")
			if len(tokens[1]) > 0 {
				values := strings.Split(tokens[1], ",")
				for _, v := range values {
					if csr.MatchString(v) {
						projectPaths = append(projectPaths, v)
					}
				}
			}
		}
	}

	if err := solFileScanner.Err(); err != nil {
		logrus.Errorf("Could not parse the solution file: %s", err)
		return nil
	}

	for i, c := range projectPaths {
		c = strings.Replace(c, "\"", "", -1)
		c = strings.Replace(c, "\\", "/", -1)
		projectPaths[i] = c
	}

	return projectPaths
}

// transform transforms the artifacts
func transform(tc transformertypes.Transformer, env *environment.Environment, newArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}
		relSrcPath, err := filepath.Rel(env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
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
		templateConfig := a.Configs[artifacts.DockerfileTemplateConfigConfigType]
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(sConfig.ServiceName)
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(env.Context, tc.Spec.TemplatesDir),
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: templateConfig,
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
