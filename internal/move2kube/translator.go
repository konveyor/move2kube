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
func Translate(p plantypes.Plan, outpath string) {
	sourceir, _ := source.Translate(p)

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
	optimizedir, _ := optimize.Optimize(sourceir)

	os.RemoveAll(outpath)
	dct := transform.ComposeTransformer{}
	err := dct.Transform(optimizedir)
	if err != nil {
		log.Errorf("Error during translate docker compose file : %s", err)
	}
	err = dct.WriteObjects(outpath)
	if err != nil {
		log.Errorf("Unable to write docker compose objects : %s", err)
	}

	log.Debugf("Total services optimized : %d", len(optimizedir.Services))
	ir, _ := customize.Customize(optimizedir)
	log.Debugf("Total storages customized : %d", len(optimizedir.Storages))
	if p.Spec.Outputs.Kubernetes.ArtifactType != plantypes.Yamls && p.Spec.Outputs.Kubernetes.ArtifactType != plantypes.Knative {
		ir, _ = parameterize.Parameterize(ir)
	}

	t := transform.GetTransformer(ir)
	err = t.Transform(ir)
	if err != nil {
		log.Errorf("Error during translate : %s", err)
	}
	err = t.WriteObjects(outpath)
	if err != nil {
		log.Errorf("Unable to write objects : %s", err)
	}

	log.Info("Execution completed")
}
