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
	"github.com/sirupsen/logrus"

	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

func processPathMappings(pms []transformertypes.PathMapping, sourcePath, outputPath string) error {
	copiedSourceDests := map[string]bool{}
	for _, pm := range pms {
		if strings.EqualFold(pm.Type, transformertypes.SourcePathMappingType) && !copiedSourceDests[pm.SrcPath+":"+pm.DestPath] {
			srcPath := pm.SrcPath
			if !filepath.IsAbs(pm.SrcPath) {
				srcPath = filepath.Join(sourcePath, pm.SrcPath)
			}
			if err := filesystem.Merge(srcPath, filepath.Join(outputPath, pm.DestPath), true); err != nil {
				logrus.Errorf("Error while copying sourcepath for %+v", pm)
			}
			copiedSourceDests[pm.SrcPath+":"+pm.DestPath] = true
		}
	}
	copiedDefaultDests := map[string]bool{}
	for _, pm := range pms {
		switch strings.ToLower(pm.Type) {
		case strings.ToLower(transformertypes.SourcePathMappingType):
		case strings.ToLower(transformertypes.ModifiedSourcePathMappingType):
			if err := filesystem.Merge(pm.SrcPath, filepath.Join(outputPath, pm.DestPath), false); err != nil {
				logrus.Errorf("Error while copying sourcepath for %+v", pm)
			}
		case strings.ToLower(transformertypes.TemplatePathMappingType):
			if err := filesystem.TemplateCopy(pm.SrcPath, filepath.Join(outputPath, pm.DestPath), pm.TemplateConfig); err != nil {
				logrus.Errorf("Error while copying sourcepath for %+v", pm)
			}
		default:
			if !copiedDefaultDests[pm.SrcPath+":"+pm.DestPath] {
				if err := filesystem.Merge(pm.SrcPath, filepath.Join(outputPath, pm.DestPath), true); err != nil {
					logrus.Errorf("Error while copying sourcepath for %+v", pm)
				}
				copiedDefaultDests[pm.SrcPath+":"+pm.DestPath] = true
			}
		}
	}
	return nil
}
