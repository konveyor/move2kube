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

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/k8sschema"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

//K8sFilesLoader implements the Loader interface
type K8sFilesLoader struct {
}

// UpdatePlan - output a plan based on the input directory contents
func (*K8sFilesLoader) UpdatePlan(inputPath string, plan *plantypes.Plan) error {
	codecs := serializer.NewCodecFactory(k8sschema.GetSchema())

	filePaths, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Errorf("Unable to fetch yaml files at path %q Error: %q", inputPath, err)
		return err
	}
	for _, filePath := range filePaths {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Debugf("Failed to read the yaml file at path %q Error: %q", filePath, err)
			continue
		}
		docs, err := common.SplitYAML(data)
		if err != nil {
			log.Debugf("Failed to split the file at path %q into YAML documents. Error: %q", filePath, err)
			continue
		}
		for _, doc := range docs {
			_, _, err = codecs.UniversalDeserializer().Decode(doc, nil, nil)
			if err != nil {
				continue
			}
			plan.Spec.Inputs.K8sFiles = append(plan.Spec.Inputs.K8sFiles, filePath)
			break
		}
	}
	return nil
}

// LoadToIR loads k8s files as cached objects
func (*K8sFilesLoader) LoadToIR(plan plantypes.Plan, ir *irtypes.IR) error {
	codecs := serializer.NewCodecFactory(k8sschema.GetSchema())
	for _, filePath := range plan.Spec.Inputs.K8sFiles {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Errorf("Failed to read the k8s file at path %q Error: %q", filePath, err)
			continue
		}
		docs, err := common.SplitYAML(data)
		if err != nil {
			log.Debugf("Failed to split the file at path %q into YAML documents. Error: %q", filePath, err)
			continue
		}
		for i, doc := range docs {
			obj, _, err := codecs.UniversalDeserializer().Decode(doc, nil, nil)
			if err != nil {
				log.Errorf("Failed to decode the YAML document %d in file at path %q as a k8s resource. Error: %q", i, filePath, err)
				continue
			}
			ir.CachedObjects = append(ir.CachedObjects, obj)
		}
	}
	return nil
}
