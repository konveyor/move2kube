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
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer"
	customize "github.com/konveyor/move2kube/internal/customizer"
	"github.com/konveyor/move2kube/internal/metadata"
	optimize "github.com/konveyor/move2kube/internal/optimizer"
	"github.com/konveyor/move2kube/internal/source"
	transform "github.com/konveyor/move2kube/internal/transformer"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

// Translate translates the artifacts and writes output
func Translate(plan plantypes.Plan, outputPath string, qadisablecli bool, transformPaths []string) {
	containerBuildTypes := []string{}
	for _, services := range plan.Spec.Inputs.Services {
		if len(services) > 0 && !common.IsStringPresent(containerBuildTypes, string(services[0].ContainerBuildType)) {
			containerBuildTypes = append(containerBuildTypes, string(services[0].ContainerBuildType))
		}
	}
	containerizer.InitContainerizers(plan.Spec.Inputs.RootDir, containerBuildTypes)
	sourceIR, err := source.Translate(plan)
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

	optimizedIR, err := optimize.Optimize(sourceIR)
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

	customizedIR, err := customize.Customize(optimizedIR)
	if err != nil {
		log.Errorf("Error occurred while running the customizers. Error: %q", err)
		customizedIR = optimizedIR
	}
	log.Debugf("Total storages customized : %d", len(customizedIR.Storages))

	customizedIR.AddCopySources = !qadisablecli
	if err := transform.Transform(customizedIR, outputPath, transformPaths); err != nil {
		log.Fatalf("Error occurred while running the customizers. Error: %q", err)
	}

	log.Info("Execution completed")
}
