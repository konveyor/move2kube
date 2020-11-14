/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package move2kube

import (
	"os"
	"strings"

	collector "github.com/konveyor/move2kube/internal/collector"
	"github.com/konveyor/move2kube/internal/common"
	log "github.com/sirupsen/logrus"
)

//Collect gets the metadata from multiple sources, filters it and dumps it into files within source directory
func Collect(inputPath string, outputPath string, annotations []string) {
	collectors, err := collector.GetCollectors()
	if err != nil {
		log.Fatalf("Failed to get the collectors. Error: %q", err)
	}
	if err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission); err != nil {
		log.Fatalf("Unable to create output directory at path %q Error: %q", outputPath, err)
	}
	log.Infoln("Begin collection")
	for _, collector := range collectors {
		if len(annotations) > 0 {
			if !hasOverlap(annotations, collector.GetAnnotations()) {
				continue
			}
		}
		log.Infof("[%T] Begin collection", collector)
		if err = collector.Collect(inputPath, outputPath); err != nil {
			log.Warnf("[%T] failed. Error: %q", collector, err)
			continue
		}
		log.Infof("[%T] Done", collector)
	}
	log.Infoln("Collection done")
}

func hasOverlap(a []string, b []string) bool {
	for _, val1 := range a {
		for _, val2 := range b {
			if strings.EqualFold(val1, val2) {
				return true
			}
		}
	}
	return false
}
