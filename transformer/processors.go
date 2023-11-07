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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube-wasm/filesystem"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/sirupsen/logrus"
)

type pair struct {
	A string
	B string
}

func getpair(a, b string) pair {
	return pair{A: a, B: b}
}

func processPathMappings(pms []transformertypes.PathMapping, sourcePath, outputPath string, failOnFirstError bool) error {
	copiedSourceDests := map[pair]bool{}
	for _, pm := range pms {
		if !strings.EqualFold(string(pm.Type), string(transformertypes.SourcePathMappingType)) || copiedSourceDests[getpair(pm.SrcPath, pm.DestPath)] {
			continue
		}
		srcPath := pm.SrcPath
		if !filepath.IsAbs(pm.SrcPath) {
			srcPath = filepath.Join(sourcePath, pm.SrcPath)
		}
		destPath := filepath.Join(outputPath, pm.DestPath)
		if err := filesystem.Merge(srcPath, destPath, true); err != nil {
			if failOnFirstError {
				return fmt.Errorf("failed to copy the source path '%s' to the destination path '%s' for the path mapping %+v . Error: %w", srcPath, destPath, pm, err)
			}
			logrus.Errorf("failed to copy the source path '%s' to the destination path '%s' for the path mapping %+v . Error: %q", srcPath, destPath, pm, err)
			continue
		}
		copiedSourceDests[getpair(pm.SrcPath, pm.DestPath)] = true
	}
	copiedDefaultDests := map[pair]bool{}
	for _, pm := range pms {
		destPath := pm.DestPath
		if !filepath.IsAbs(pm.DestPath) {
			destPath = filepath.Join(outputPath, pm.DestPath)
		}
		switch strings.ToLower(string(pm.Type)) {
		case strings.ToLower(string(transformertypes.SourcePathMappingType)): // skip sources
		case strings.ToLower(string(transformertypes.DeletePathMappingType)): // skip deletes
		case strings.ToLower(string(transformertypes.ModifiedSourcePathMappingType)):
			if err := filesystem.Merge(pm.SrcPath, destPath, failOnFirstError); err != nil {
				if failOnFirstError {
					return fmt.Errorf("failed to merge for the path mapping %+v . Error: %w", pm, err)
				}
				logrus.Errorf("Error while copying sourcepath for %+v . Error: %q", pm, err)
			}
		case strings.ToLower(string(transformertypes.TemplatePathMappingType)):
			if err := filesystem.TemplateCopy(pm.SrcPath, destPath, filesystem.AddOnConfig{Config: pm.TemplateConfig}); err != nil {
				if failOnFirstError {
					return fmt.Errorf("failed to copy the template for the path mapping %+v . Error: %w", pm, err)
				}
				logrus.Errorf("Error while copying sourcepath for %+v . Error: %q", pm, err)
			}
		case strings.ToLower(string(transformertypes.SpecialTemplatePathMappingType)):
			if err := filesystem.TemplateCopy(
				pm.SrcPath,
				destPath,
				filesystem.AddOnConfig{
					OpeningDelimiter: filesystem.SpecialOpeningDelimiter,
					ClosingDelimiter: filesystem.SpecialClosingDelimiter,
					Config:           pm.TemplateConfig,
				},
			); err != nil {
				if failOnFirstError {
					return fmt.Errorf("failed to copy the special template for the path mapping %+v . Error: %w", pm, err)
				}
				logrus.Errorf("Error while copying sourcepath for %+v . Error: %q", pm, err)
			}
		default:
			if !copiedDefaultDests[getpair(pm.SrcPath, pm.DestPath)] {
				if err := filesystem.Merge(pm.SrcPath, destPath, failOnFirstError); err != nil {
					if failOnFirstError {
						return fmt.Errorf("failed to merge for the path mapping %+v . Error: %w", pm, err)
					}
					logrus.Errorf("Error while copying sourcepath for %+v . Error: %q", pm, err)
				}
				copiedDefaultDests[getpair(pm.SrcPath, pm.DestPath)] = true
			}
		}
	}

	for _, pm := range pms {
		if !strings.EqualFold(string(pm.Type), string(transformertypes.DeletePathMappingType)) {
			continue
		}
		destPath := pm.DestPath
		if !filepath.IsAbs(pm.DestPath) {
			destPath = filepath.Join(outputPath, pm.DestPath)
		}
		if err := os.RemoveAll(destPath); err != nil {
			if failOnFirstError {
				return fmt.Errorf("failed to remove the destination path '%s' . Error: %w", destPath, err)
			}
			logrus.Errorf("Path [%s] marked by delete-path-mapping could not be deleted. Error: %q", destPath, err)
			continue
		}
		logrus.Debugf("Path [%s] marked by delete-path-mapping has been deleted", destPath)
	}
	return nil
}
