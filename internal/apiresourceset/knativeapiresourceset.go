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

package apiresourceset

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	knative "knative.dev/serving/pkg/client/clientset/versioned/scheme"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// KnativeAPIResourceSet manages knative related objects
type KnativeAPIResourceSet struct {
}

// GetScheme returns knative scheme object
func (k *KnativeAPIResourceSet) GetScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = knative.AddToScheme(scheme)
	return scheme
}

func (k *KnativeAPIResourceSet) getAPIResources(ir irtypes.IR) []apiresource.APIResource {
	apiresources := []apiresource.APIResource{{IAPIResource: &apiresource.KnativeService{Cluster: ir.TargetClusterSpec}}}
	return apiresources
}

// CreateAPIResources converts ir object to runtime objects
func (k *KnativeAPIResourceSet) CreateAPIResources(ir irtypes.IR) []runtime.Object {
	targetobjs := []runtime.Object{}
	ignoredobjs := ir.CachedObjects
	for _, apiresource := range k.getAPIResources(ir) {
		apiresource.SetClusterContext(ir.TargetClusterSpec)
		resourceignoredobjs := apiresource.LoadResources(ir.CachedObjects)
		ignoredobjs = intersection(ignoredobjs, resourceignoredobjs)
		objs := apiresource.GetUpdatedResources(ir)
		targetobjs = append(targetobjs, objs...)
	}
	targetobjs = append(targetobjs, ignoredobjs...)
	return targetobjs
}

// GetServiceOptions returns plan service options for an input folder
func (k *KnativeAPIResourceSet) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	services := make([]plantypes.Service, 0)

	codecs := serializer.NewCodecFactory(k.GetScheme())

	files, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Warnf("Unable to fetch yaml files and recognize knative yamls : %s", err)
	}
	for _, path := range files {
		//relpath, _ := plan.GetRelativePath(path)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Debugf("ignoring file %s", path)
			continue
		}
		obj, _, err := codecs.UniversalDeserializer().Decode(data, nil, nil)
		if err != nil {
			log.Debugf("ignoring file %s since serialization failed", path)
			continue
		} else {
			if d1, ok := obj.(*knativev1.Service); ok {
				service := newKnativeService(d1.Name)
				relpath, _ := plan.GetRelativePath(path)
				service.SourceArtifacts[plantypes.KnativeFileArtifactType] = []string{relpath}
				services = append(services, service)
			}
		}
	}
	return services, nil
}

// Translate translates plan services to IR
func (k *KnativeAPIResourceSet) Translate(services []plantypes.Service, p plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(p)
	ir.Services = make(map[string]irtypes.Service)
	codecs := serializer.NewCodecFactory(k.GetScheme())

	for _, service := range services {
		irservice := irtypes.Service{Name: service.ServiceName}
		if len(service.SourceArtifacts[plantypes.KnativeFileArtifactType]) > 0 {
			fullpath := p.GetFullPath(service.SourceArtifacts[plantypes.KnativeFileArtifactType][0])
			data, err := ioutil.ReadFile(fullpath)
			if err != nil {
				log.Debugf("Unable to load file : %s", fullpath)
				continue
			}
			obj, _, err := codecs.UniversalDeserializer().Decode(data, nil, nil)
			if err != nil {
				log.Debugf("ignoring file %s since serialization failed", fullpath)
				continue
			}
			if d1, ok := obj.(*knativev1.Service); ok {
				irservice.PodSpec = d1.Spec.ConfigurationSpec.Template.Spec.PodSpec
			}
		} else {
			log.Warnf("No knative artifacts found in service %s", service.ServiceName)
		}
		ir.Services[service.ServiceName] = irservice
	}
	return ir, nil
}

func newKnativeService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, plantypes.Knative2KubeTranslation)
	service.ContainerBuildType = plantypes.ReuseContainerBuildTypeValue
	service.SourceTypes = []plantypes.SourceTypeValue{plantypes.KNativeSourceTypeValue}
	service.UpdateContainerBuildPipeline = false
	service.UpdateDeployPipeline = true
	return service
}
