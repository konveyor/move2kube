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

package api

import (
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/parameterizer"
	parameterizertypes "github.com/konveyor/move2kube/types/parameterizer"
	"github.com/sirupsen/logrus"
)

// Parameterize does the parameterization
func Parameterize(srcDir string, packDir string, outDir string) ([]string, error) {
	cleanPackDir, err := filepath.Abs(packDir)
	if err != nil {
		return nil, err
	}
	packs, err := collectPacksFromPath(cleanPackDir)
	if err != nil {
		return nil, err
	}
	namedPs, err := parameterizer.CollectParamsFromPath(cleanPackDir)
	if err != nil {
		return nil, err
	}
	filesWritten := []string{}
	for _, pack := range packs {
		ps := []parameterizertypes.ParameterizerT{}
		for _, name := range pack.Spec.ParameterizerRefs {
			if currPs, ok := namedPs[name]; ok {
				ps = append(ps, currPs...)
				continue
			}
			logrus.Errorf("failed to find the paramterizers with the name %s referred to by the packaging with the name %s , in the folder %s", name, pack.ObjectMeta.Name, cleanPackDir)
		}
		ps = append(ps, pack.Spec.Parameterizers...)
		for _, path := range pack.Spec.Paths {
			fw, err := parameterizer.Parameterize(srcDir, outDir, path, ps)
			if err != nil {
				logrus.Errorf("Unable to process path %s : %s", path.Src, err)
				continue
			}
			filesWritten = append(filesWritten, fw...)
		}
	}
	return filesWritten, nil
}

func collectPacksFromPath(packDir string) ([]parameterizertypes.PackagingFileT, error) {
	yamlPaths, err := common.GetFilesByExt(packDir, []string{".yaml", ".yml"})
	if err != nil {
		return nil, err
	}
	packs := []parameterizertypes.PackagingFileT{}
	for _, yamlPath := range yamlPaths {
		pack := parameterizertypes.PackagingFileT{
			Spec: parameterizertypes.PackagingSpecT{
				FilePath: yamlPath,
			},
		}
		if err := common.ReadMove2KubeYamlStrict(yamlPath, &pack, parameterizertypes.PackagingKind); err == nil {
			logrus.Debugf("found packing yaml at path %s", yamlPath)
			packs = append(packs, pack)
			continue
		}
	}
	return packs, nil
}
