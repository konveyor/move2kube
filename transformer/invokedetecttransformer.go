/*
 *  Copyright IBM Corporation 2023
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

package transformer

import (
	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/konveyor/move2kube-wasm/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// InvokeDetect implements the Transformer interface
type InvokeDetect struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *InvokeDetect) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *InvokeDetect) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect does nothing
func (t *InvokeDetect) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *InvokeDetect) Transform(
	inputArtifacts []transformertypes.Artifact,
	inputOldArtifacts []transformertypes.Artifact,
) (
	[]transformertypes.PathMapping,
	[]transformertypes.Artifact,
	error,
) {
	logrus.Trace("InvokeDetect.Transform start")
	defer logrus.Trace("InvokeDetect.Transform end")
	outputArtifacts := []transformertypes.Artifact{}
	for _, inputArtifact := range inputArtifacts {
		if len(inputArtifact.Paths[artifacts.InvokeDetectPathType]) == 0 {
			logrus.Errorf("the path to run the detect function is missing from the InvokeDetect artifact. Skipping")
			continue
		}
		logrus.Debugf("inputArtifact.Paths[artifacts.InvokeDetectPathType] %+v", inputArtifact.Paths[artifacts.InvokeDetectPathType])
		detectDir := inputArtifact.Paths[artifacts.InvokeDetectPathType][0]
		invokeDetectConfig := artifacts.InvokeDetectConfig{}
		if err := inputArtifact.GetConfig(artifacts.InvokeDetectConfigType, &invokeDetectConfig); err != nil {
			logrus.Errorf("failed to load the InvokeDetect type config into struct of type %T . Error: %q", invokeDetectConfig, err)
			continue
		}
		detectedServices, err := GetServices(common.DefaultProjectName, detectDir, &invokeDetectConfig.TransformerSelector)
		if err != nil {
			logrus.Errorf("failed to invoke the directory detect. Error: %q", err)
			continue
		}
		for serviceName, planArtifacts := range detectedServices {
			if len(planArtifacts) > 1 {
				logrus.Infof("Found multiple transformation options for the service '%s'. Selecting the first valid option.", serviceName)
			}
			found := false
			for _, planArtifact := range planArtifacts {
				if _, err := GetTransformerByName(planArtifact.TransformerName); err != nil {
					logrus.Errorf(
						"failed to get the transformer named '%s' for the service '%s' . Error: %q",
						planArtifact.TransformerName, serviceName, err,
					)
					continue
				}
				planArtifact.ServiceName = serviceName
				planArtifact = preprocessArtifact(planArtifact)
				outputArtifacts = append(outputArtifacts, planArtifact.Artifact)
				logrus.Infof("Using the transformation option '%s' for the service '%s'.", planArtifact.TransformerName, serviceName)
				found = true
				break
			}
			if !found {
				logrus.Warnf("No valid transformers were found for the service '%s'. Skipping.", serviceName)
			}
		}
	}
	return nil, outputArtifacts, nil
}
