/*
Copyright IBM Corporation 2020, 2021

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

package api

import (
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// Translate translates the artifacts and writes output
func Translate(plan plantypes.Plan, outputPath string, qadisablecli bool) {
	/*sourceIR, err := translator.Translate(plan)
	if err != nil {
		log.Fatalf("Failed to translate the plan to intermediate representation. Error: %q", err)
	}
	log.Debugf("Total storages loaded : %d", len(sourceIR.Storages))

	log.Infoln("Begin Metadata loading")
	metadataLoaders := metadata.GetLoaders()
	for _, metadataLoader := range metadataLoaders {
		log.Debugf("[%T] Begin metadata loading", metadataLoader)
		err := metadataLoader.LoadToIR(plan, &sourceIR)
		if err != nil {
			log.Warnf("Metadata loader [%T] failed. Error: %q", metadataLoader, err)
		} else {
			log.Debugf("[%T] Done", metadataLoader)
		}
	}
	log.Infoln("Metadata loading done")

	log.Debugf("Total services loaded : %d", len(sourceIR.Services))
	log.Debugf("Total containers loaded : %d", len(sourceIR.Containers))

	optimizedIR, err := irpreprocessor.Preprocess(sourceIR)
	if err != nil {
		log.Errorf("Error occurred while running the optimizers. Error: %q", err)
		optimizedIR = sourceIR
	}
	log.Debugf("Total services optimized : %d", len(optimizedIR.Services))

	composeTransformer := transform.ComposeTransformer{}
	if err := composeTransformer.Transform(optimizedIR); err != nil {
		log.Errorf("Error while translating docker compose file. Error: %q", err)
	} else if err := composeTransformer.WriteObjects(outputPath, nil); err != nil {
		log.Errorf("Unable to write docker compose objects. Error: %q", err)
	}
	log.Info("Translation completed")*/
}
