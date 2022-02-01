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

package transformer

import (
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// getContainerizationOptions returns possible containerization options for a directory
func getContainerizationOptions(directory string) []string {
	options := []string{}
	filters := labels.NewSelector()
	req, err := labels.NewRequirement("move2kube.konveyor.io/task", selection.Equals, []string{"containerization"})
	if err != nil {
		logrus.Errorf("Unable to parse requirement : %s", err)
	}
	filters = filters.Add(*req)
	for _, t := range GetInitializedTransformersF(filters) {
		config, env := t.GetConfig()
		env.Reset()
		newoptions, err := t.DirectoryDetect(directory)
		if err != nil {
			logrus.Warnf("[%s] Failed during containerization option fetch : %s", config.Name, err)
			continue
		}
		if len(newoptions) > 0 {
			options = append(options, config.Name)
		}
	}
	return options
}

// getContainerizationConfig returns possible containerization config for a directory while using a transformer
func getContainerizationConfig(serviceName string, projectDirectory, buildArtifacts []string, transformerName string) transformertypes.Artifact {
	t, err := GetTransformerByName(transformerName)
	if err != nil {
		logrus.Errorf("Unable to find the transformer %s", transformerName)
		return transformertypes.Artifact{}
	}
	tc, env := t.GetConfig()
	env.Reset()
	newoptions, err := t.DirectoryDetect(projectDirectory[0])
	if err != nil {
		logrus.Warnf("[%s] Failed during containerization option fetch : %s", tc.Name, err)
	}
	if len(newoptions) > 1 {
		logrus.Infof("More than one containerization option found for a directory. Choosing one for %s", projectDirectory)
		if option, ok := newoptions[serviceName]; ok && len(option) > 0 {
			newoptions = map[string][]transformertypes.Artifact{
				serviceName: option,
			}
		}
	}
	for _, nos := range newoptions {
		if len(nos) > 1 {
			logrus.Infof("More than one containerization option found for a directory. Choosing one for %s", projectDirectory)
		}
		if len(nos) == 0 {
			continue
		}
		if buildArtifacts != nil {
			nos[0].Paths[artifacts.BuildArtifactPathType] = buildArtifacts
		}
		nos[0].Name = serviceName
		nos[0].ProcessWith = *metav1.AddLabelToSelector(&nos[0].ProcessWith, transformertypes.LabelName, string(transformerName))
		nos[0].Type = artifacts.ServiceArtifactType
		return nos[0]
	}
	return transformertypes.Artifact{}
}
