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
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/common/deepcopy"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
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

func getArtifactForTransformerPlan(serviceName string, t transformertypes.TransformerPlan, p plantypes.Plan) transformertypes.Artifact {
	serviceConfig := artifacts.ServiceConfig{
		ServiceName: serviceName,
	}
	if t.Configs == nil {
		t.Configs = map[string]interface{}{}
	}
	t.Configs[artifacts.ServiceConfigType] = serviceConfig
	artifact := transformertypes.Artifact{
		Name:     serviceName,
		Artifact: artifacts.ServiceArtifactType,
		Paths:    t.Paths,
		Configs:  t.Configs,
	}
	return artifact
}

func updatedArtifacts(oldArtifacts, newArtifacts []transformertypes.Artifact) (updatedArtifacts []transformertypes.Artifact) {
	for ai, a := range newArtifacts {
		for _, oa := range oldArtifacts {
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
	if a.Artifact == b.Artifact && a.Name == b.Name {
		c = transformertypes.Artifact{
			Name:     a.Name,
			Artifact: a.Artifact,
			Paths:    mergePathSliceMaps(a.Paths, b.Paths),
			Configs:  mergeConfigs(a.Configs, b.Configs),
		}
		return c, true
	}
	return c, false
}

func mergeConfigs(configs1 map[transformertypes.ConfigType]interface{}, configs2 map[transformertypes.ConfigType]interface{}) map[transformertypes.ConfigType]interface{} {
	if configs1 == nil {
		return configs2
	}
	if configs2 == nil {
		return configs1
	}
	for cn2, cg2 := range configs2 {
		if configs1[cn2] == nil {
			configs1[cn2] = cg2
			continue
		}
		configs1[cn2] = deepcopy.Merge(configs1[cn2], configs2[cn2])
	}
	return configs1
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

func setTransformerInfoForServices(services map[string][]transformertypes.TransformerPlan, t transformertypes.Transformer) map[string][]transformertypes.TransformerPlan {
	for sn, s := range services {
		for sti, st := range s {
			st.TransformerName = t.Name
			st.Mode = t.Spec.Mode
			services[sn][sti] = st
		}
	}
	return services
}

func getNamedAndUnNamedServicesLogMessage(services map[string][]transformertypes.TransformerPlan) string {
	nnservices := len(services)
	nuntransformers := len(services[""])
	if _, ok := services[""]; ok {
		nuntransformers -= 1
	}
	return fmt.Sprintf("Identified %d namedservices and %d unnamed transformer plans", nnservices, nuntransformers)
}
