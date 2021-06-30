/*
 *  Copyright IBM Corporation 2020, 2021
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
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/konveyor/move2kube/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	dockerparser "github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/sirupsen/logrus"
)

// DockerfileDetector implements the Transformer interface
type DockerfileDetector struct {
	Config transformertypes.Transformer
	Env    environment.Environment
}

func (t *DockerfileDetector) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

func (t *DockerfileDetector) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.Config, t.Env
}

func (t *DockerfileDetector) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	ts := []plantypes.Transformer{}
	if info, err := os.Stat(dir); os.IsNotExist(err) {
		logrus.Warnf("Error in walking through files due to : %s", err)
		return nil, nil, err
	} else if !info.IsDir() {
		logrus.Warnf("The path %q is not a directory.", dir)
	}
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("Skipping path %s due to error: %s", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		if isdf, _ := isDockerFile(path); isdf {
			trans := plantypes.Transformer{
				Mode:                   string(t.Config.Spec.Mode),
				ArtifactTypes:          t.Config.Spec.Artifacts,
				ExclusiveArtifactTypes: t.Config.Spec.ExclusiveArtifacts,
				Paths: map[string][]string{
					artifacts.ProjectPathPathType: {filepath.Dir(path)},
					artifacts.DockerfilePathType:  {path},
				},
			}
			ts = append(ts, trans)
		}
		return nil
	})
	if err != nil {
		logrus.Warnf("Error in walking through files due to : %s", err)
	}
	return nil, ts, nil
}

func (t *DockerfileDetector) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

func (t *DockerfileDetector) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}
		var pConfig artifacts.PlanConfig
		err := a.GetConfig(artifacts.PlanConfigType, &pConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", pConfig, err)
			continue
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		p := transformertypes.Artifact{
			Name:     sConfig.ServiceName,
			Artifact: artifacts.DockerfileArtifactType,
			Paths:    a.Paths,
		}
		artifactsCreated = append(artifactsCreated, p)
	}
	return nil, artifactsCreated, nil
}

func isDockerFile(path string) (isDockerfile bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		logrus.Debugf("Unable to open file %s : %s", path, err)
		return false, err
	}
	defer f.Close()
	res, err := dockerparser.Parse(f)
	if err != nil {
		logrus.Debugf("Unable to parse file %s as Docker files : %s", path, err)
		return false, err
	}
	for _, dfchild := range res.AST.Children {
		if dfchild.Value == "from" {
			r := regexp.MustCompile(`(?i)FROM\s+(--platform=[^\s]+)?[^\s]+(\s+AS\s+[^\s]+)?\s*(#.+)?$`)
			if r.MatchString(dfchild.Original) {
				logrus.Debugf("Identified a docker file : " + path)
				return true, nil
			}
			return false, nil
		}
		if dfchild.Value == "arg" {
			continue
		}
		return false, fmt.Errorf("%s is not a valid Dockerfile", path)
	}
	return false, fmt.Errorf("%s is not a valid Dockerfile", path)
}
