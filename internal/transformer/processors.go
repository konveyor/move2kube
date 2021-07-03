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
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/filesystem"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

type pair struct {
	A string
	B string
}

func getpair(a, b string) pair {
	return pair{A: a, B: b}
}

func processPathMappings(pms []transformertypes.PathMapping, sourcePath, outputPath string) error {
	copiedSourceDests := map[pair]bool{}
	for _, pm := range pms {
		if !strings.EqualFold(pm.Type, transformertypes.SourcePathMappingType) || copiedSourceDests[getpair(pm.SrcPath, pm.DestPath)] {
			continue
		}
		srcPath := pm.SrcPath
		if !filepath.IsAbs(pm.SrcPath) {
			srcPath = filepath.Join(sourcePath, pm.SrcPath)
		}
		destPath := filepath.Join(outputPath, pm.DestPath)
		if err := filesystem.Merge(srcPath, destPath, true); err != nil {
			logrus.Errorf("Error while copying sourcepath for %+v . Error: %q", pm, err)
		}
		copiedSourceDests[getpair(pm.SrcPath, pm.DestPath)] = true
	}
	copiedDefaultDests := map[pair]bool{}
	for _, pm := range pms {
		destPath := filepath.Join(outputPath, pm.DestPath)
		switch strings.ToLower(pm.Type) {
		case strings.ToLower(transformertypes.SourcePathMappingType): // skip sources
		case strings.ToLower(transformertypes.ModifiedSourcePathMappingType):
			if err := filesystem.Merge(pm.SrcPath, destPath, false); err != nil {
				logrus.Errorf("Error while copying sourcepath for %+v", pm)
			}
		case strings.ToLower(transformertypes.TemplatePathMappingType):
			if err := filesystem.TemplateCopy(pm.SrcPath, destPath, pm.TemplateConfig); err != nil {
				logrus.Errorf("Error while copying sourcepath for %+v", pm)
			}
		default:
			if !copiedDefaultDests[getpair(pm.SrcPath, pm.DestPath)] {
				if err := filesystem.Merge(pm.SrcPath, destPath, true); err != nil {
					logrus.Errorf("Error while copying sourcepath for %+v", pm)
				}
				copiedDefaultDests[getpair(pm.SrcPath, pm.DestPath)] = true
			}
		}
	}
	return nil
}
