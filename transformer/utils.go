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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/common/deepcopy"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func getTransformerConfig(path string) (transformertypes.Transformer, error) {
	tc := transformertypes.NewTransformer()
	tc.Spec.FilePath = path
	if err := common.ReadMove2KubeYaml(path, &tc); err != nil {
		logrus.Debugf("Failed to read the transformer metadata at path %q Error: %q", path, err)
		return tc, err
	}
	if tc.Kind != transformertypes.TransformerKind {
		err := fmt.Errorf("the file at path %q is not a valid cluster metadata. Expected kind: %s Actual kind: %s", path, transformertypes.TransformerKind, tc.Kind)
		logrus.Debug(err)
		return tc, err
	}
	tc.Labels[transformertypes.LabelName] = tc.Name
	tc.Spec.Override = nil
	if tc.Spec.OverrideAsObj != nil {
		s := metav1.LabelSelector{}
		err := common.GetObjFromInterface(tc.Spec.OverrideAsObj, &s)
		if err != nil {
			logrus.Errorf("Unable to parse Override configuration for %s, ignoring override : %s", tc.Name, err)
		} else {
			if len(s.MatchExpressions) != 0 || len(s.MatchLabels) != 0 {
				tc.Spec.Override, err = metav1.LabelSelectorAsSelector(&s)
				if err != nil {
					logrus.Errorf("Unable to convert label selector to selector : %s", err)
				}
			}
		}
	}
	return tc, nil
}

func getIgnorePaths(inputPath string) (ignoreDirectories []string, ignoreContents []string) {
	filePaths, err := common.GetFilesByName(inputPath, []string{common.IgnoreFilename}, nil)
	if err != nil {
		logrus.Warnf("Unable to fetch .m2kignore files at path %q Error: %q", inputPath, err)
		return ignoreDirectories, ignoreContents
	}
	for _, filePath := range filePaths {
		file, err := os.Open(filePath)
		if err != nil {
			logrus.Warnf("Failed to open the .m2kignore file at path %q Error: %q", filePath, err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasSuffix(line, "*") {
				line = strings.TrimSuffix(line, "*")
				path := filepath.Join(filepath.Dir(filePath), line)
				ignoreContents = append(ignoreContents, path)
			} else {
				path := filepath.Join(filepath.Dir(filePath), line)
				ignoreDirectories = append(ignoreDirectories, path)
			}
		}
	}
	return ignoreDirectories, ignoreContents
}

func updatedArtifacts(alreadySeenArtifacts []transformertypes.Artifact, newArtifacts ...transformertypes.Artifact) (updatedArtifacts []transformertypes.Artifact) {
	for ai, a := range newArtifacts {
		for _, oa := range alreadySeenArtifacts {
			if mergeda, merged := mergeArtifact(a, oa); merged {
				newArtifacts[ai] = mergeda
			}
		}
	}
	return mergeArtifacts(newArtifacts)
}

func mergeArtifacts(artifacts []transformertypes.Artifact) (newArtifacts []transformertypes.Artifact) {
	newArtifacts = []transformertypes.Artifact{}
	for _, a := range artifacts {
		added := false
		for nai, na := range newArtifacts {
			if mergeda, merged := mergeArtifact(a, na); merged {
				newArtifacts[nai] = mergeda
				added = true
				break
			}
		}
		if !added {
			newArtifacts = append(newArtifacts, a)
		}
	}
	return
}

func mergeArtifact(a transformertypes.Artifact, b transformertypes.Artifact) (c transformertypes.Artifact, merged bool) {
	if a.Type == b.Type && a.Name == b.Name {
		mergedConfig, merged := mergeConfigs(a.Configs, b.Configs)
		if !merged {
			return c, false
		}
		c = transformertypes.Artifact{
			Name:    a.Name,
			Type:    a.Type,
			Paths:   mergePathSliceMaps(a.Paths, b.Paths),
			Configs: mergedConfig,
		}
		return c, true
	}
	return c, false
}

func mergeConfigs(configs1 map[transformertypes.ConfigType]interface{}, configs2 map[transformertypes.ConfigType]interface{}) (mergedConfig map[transformertypes.ConfigType]interface{}, merged bool) {
	if configs1 == nil {
		return configs2, true
	}
	if configs2 == nil {
		return configs1, true
	}
	for cn2, cg2 := range configs2 {
		if configs1[cn2] == nil {
			configs1[cn2] = cg2
			continue
		}
		if ct, ok := artifacts.ConfigTypes[string(cn2)]; ok {
			c1 := reflect.New(ct).Interface().(transformertypes.Config)
			err := common.GetObjFromInterface(configs1[cn2], c1)
			if err != nil {
				logrus.Errorf("Unable to load config : %s", err)
				break
			}
			c2 := reflect.New(ct).Interface().(transformertypes.Config)
			err = common.GetObjFromInterface(configs2[cn2], c2)
			if err != nil {
				logrus.Errorf("Unable to load config : %s", err)
				break
			}
			if merged = c1.Merge(c2); !merged {
				return configs1, false
			}
			configs1[cn2] = c1
			continue
		}
		configs1[cn2] = deepcopy.Merge(configs1[cn2], configs2[cn2])
	}
	return configs1, true
}

// mergePathSliceMaps merges two string slice maps
func mergePathSliceMaps(map1 map[transformertypes.PathType][]string, map2 map[transformertypes.PathType][]string) map[transformertypes.PathType][]string {
	if map1 == nil {
		return map2
	}
	if map2 == nil {
		return map1
	}
	for k, v := range map2 {
		map1[k] = common.MergeStringSlices(map1[k], v...)
	}
	return map1
}

func setTransformerInfoForServices(services map[string][]transformertypes.Artifact, t transformertypes.Transformer) map[string][]transformertypes.Artifact {
	for sn, s := range services {
		for sti, st := range s {
			// To make the plan yaml look better we do this
			st.Type = transformertypes.ArtifactType(t.Name) //artifacts.ServiceArtifactType
			//st.ProcessWith = *metav1.AddLabelToSelector(&st.ProcessWith, transformertypes.LabelName, t.Name)
			services[sn][sti] = st
		}
	}
	return services
}

func getNamedAndUnNamedServicesLogMessage(services map[string][]transformertypes.Artifact) string {
	nnservices := len(services)
	nuntransformers := len(services[""])
	if _, ok := services[""]; ok {
		nuntransformers -= 1
	}
	return fmt.Sprintf("Identified %d named services and %d to-be-named services", nnservices, nuntransformers)
}

func getFilteredTransformers(transformerPaths map[string]string, selector labels.Selector, logError bool) (transformerConfigs map[string]transformertypes.Transformer) {
	filteredTransformerConfigs := map[string]transformertypes.Transformer{}
	overrideSelectors := []labels.Selector{}
	for tn, tfilepath := range transformerPaths {
		tc, err := getTransformerConfig(tfilepath)
		if err != nil {
			if logError {
				logrus.Errorf("Unable to load %s as Transformer config : %s", tfilepath, err)
			} else {
				logrus.Debugf("Unable to load %s as Transformer config : %s", tfilepath, err)
			}
			continue
		}
		if ot, ok := filteredTransformerConfigs[tc.Name]; ok {
			logrus.Errorf("Found two conflicting transformer Names %s : %s, %s. Ignoring %s.", tc.Name, ot.Spec.FilePath, tc.Spec.FilePath, ot.Spec.FilePath)
		}
		if !selector.Matches(labels.Set(tc.Labels)) {
			logrus.Debugf("Ignoring transformer %s because of filter", tn)
			continue
		}
		if tc.Spec.Override != nil {
			overrideSelectors = append(overrideSelectors, tc.Spec.Override)
		}
		if _, ok := transformerTypes[tc.Spec.Class]; ok {
			filteredTransformerConfigs[tc.Name] = tc
			continue
		}
		logrus.Errorf("Ignoring transformer %s since the class %s not found", tn, tc.Spec.Class)
	}
	transformerConfigs = map[string]transformertypes.Transformer{}
	for tn, tc := range filteredTransformerConfigs {
		if ot, ok := transformerConfigs[tc.Name]; ok {
			logrus.Errorf("Found two conflicting transformer Names %s : %s, %s. Ignoring %s.", tc.Name, ot.Spec.FilePath, tc.Spec.FilePath, ot.Spec.FilePath)
		}
		ignore := false
		for _, overrideSelector := range overrideSelectors {
			if overrideSelector.Matches(labels.Set(tc.Labels)) {
				ignore = true
				break
			}
		}
		if !ignore {
			transformerConfigs[tn] = tc
		}
	}
	return transformerConfigs
}

func postProcessArtifacts(artifacts []transformertypes.Artifact, t transformertypes.Transformer) []transformertypes.Artifact {
	newArtifacts := []transformertypes.Artifact{}
	for _, a := range artifacts {
		if p, ok := t.Spec.ProducedArtifacts[a.Type]; ok && p.ChangeTypeTo != "" {
			a.Type = p.ChangeTypeTo
		}
		newArtifacts = append(newArtifacts, a)
	}
	return newArtifacts
}

func selectTransformer(selector metav1.LabelSelector, t transformertypes.Transformer) (bool, error) {
	ls, err := metav1.LabelSelectorAsSelector(&selector)
	if err != nil {
		return false, err
	}
	if ls.Matches(labels.Set(t.Labels)) {
		return true, nil
	}
	return false, nil
}
