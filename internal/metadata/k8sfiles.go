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

package metadata

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/konveyor/move2kube/internal/apiresourceset"
	common "github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

//K8sFilesLoader Implements Loader interface
type K8sFilesLoader struct {
}

// UpdatePlan - output a plan based on the input directory contents
func (i K8sFilesLoader) UpdatePlan(inputPath string, plan *plantypes.Plan) error {
	codecs := serializer.NewCodecFactory((&apiresourceset.K8sAPIResourceSet{}).GetScheme())

	files, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		return err
	}
	for _, path := range files {
		relpath, _ := plan.GetRelativePath(path)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Debugf("ignoring file %s", path)
			continue
		}
		_, _, err = codecs.UniversalDeserializer().Decode(data, nil, nil)
		if err != nil {
			log.Debugf("ignoring file %s since serialization failed", path)
			continue
		}

		plan.Spec.Inputs.K8sFiles = append(plan.Spec.Inputs.K8sFiles, relpath)
	}
	return nil
}

// LoadToIR loads k8s files as cached objects
func (i K8sFilesLoader) LoadToIR(p plantypes.Plan, ir *irtypes.IR) error {
	codecs := serializer.NewCodecFactory((&apiresourceset.K8sAPIResourceSet{}).GetScheme())
	for _, path := range p.Spec.Inputs.K8sFiles {
		fullpath := p.GetFullPath(path)
		data, err := ioutil.ReadFile(fullpath)
		if err != nil {
			log.Debugf("ignoring file %s", path)
			continue
		}
		obj, _, err := codecs.UniversalDeserializer().Decode(data, nil, nil)
		if err != nil {
			log.Debugf("ignoring file %s since serialization failed", path)
			continue
		}

		ir.CachedObjects = append(ir.CachedObjects, obj)
	}
	return nil
}
