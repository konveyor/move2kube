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

	customize "github.com/konveyor/move2kube/internal/customizer"
	"github.com/konveyor/move2kube/internal/metadata"
	optimize "github.com/konveyor/move2kube/internal/optimizer"
	parameterize "github.com/konveyor/move2kube/internal/parameterizer"
	"github.com/konveyor/move2kube/internal/source"
	transform "github.com/konveyor/move2kube/internal/transformer"
	plantypes "github.com/konveyor/move2kube/types/plan"

	log "github.com/sirupsen/logrus"
)

// Translate translates the artifacts and writes output
func Translate(p plantypes.Plan, outpath string, qadisablecli bool) {
	sourceir, err := source.Translate(p)
	if err != nil {
		log.Fatalf("Failed to translate the plan to intermediate representation. Error: %q", err)
	}
	log.Debugf("Total storages loaded : %d", len(sourceir.Storages))

	log.Infoln("Begin Metadata loading")
	metadataPlanners := metadata.GetLoaders()
	for _, l := range metadataPlanners {
		log.Debugf("[%T] Begin metadata loading", l)
		err := l.LoadToIR(p, &sourceir)
		if err != nil {
			log.Warnf("[%T] Failed : %s", l, err.Error())
		} else {
			log.Debugf("[%T] Done", l)
		}
	}
	log.Infoln("Metadata loading done")

	log.Debugf("Total services loaded : %d", len(sourceir.Services))
	log.Debugf("Total containers loaded : %d", len(sourceir.Containers))

	optimizedir, err := optimize.Optimize(sourceir)
	if err != nil {
		log.Warnf("Failed to optimize the intermediate representation. Error: %q", err)
		optimizedir = sourceir
	}
	log.Debugf("Total services optimized : %d", len(optimizedir.Services))

	if err := os.RemoveAll(outpath); err != nil {
		log.Errorf("Failed to remove the existing file/directory at the output path %q Error: %q", outpath, err)
		log.Errorf("Anything in the output path will get overwritten.")
	}

	dct := transform.ComposeTransformer{}
	if err := dct.Transform(optimizedir); err != nil {
		log.Errorf("Error during translate docker compose file : %s", err)
	} else if err = dct.WriteObjects(outpath); err != nil {
		log.Errorf("Unable to write docker compose objects : %s", err)
	}

	ir, _ := customize.Customize(optimizedir)
	log.Debugf("Total storages customized : %d", len(optimizedir.Storages))
	if p.Spec.Outputs.Kubernetes.ArtifactType == plantypes.Helm {
		ir, err = parameterize.Parameterize(ir)
		if err != nil {
			log.Debugf("Failed to paramterize the IR. Error: %q", err)
		}
	}

	anyNewContainers := false
	for _, container := range ir.Containers {
		if container.New {
			anyNewContainers = true
			break
		}
	}
	if anyNewContainers {
		cicd := transform.CICDTransformer{}
		if err := cicd.Transform(ir); err != nil {
			log.Errorf("Error while genrationg CI/CD resource fomr the IR. Error: %q", err)
		} else if err = cicd.WriteObjects(outpath); err != nil {
			log.Errorf("Unable to write the CI/CD artifacts to files. Error: %q", err)
		}
	}

	ir.AddCopySourcesWarning = qadisablecli
	t := transform.GetTransformer(ir)
	if err := t.Transform(ir); err != nil {
		log.Fatalf("Error during translate. Error: %q", err)
	} else if err := t.WriteObjects(outpath); err != nil {
		log.Fatalf("Unable to write objects Error: %q", err)
	}

	log.Info("Execution completed")
}
