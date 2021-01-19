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

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	knative "knative.dev/serving/pkg/client/clientset/versioned/scheme"
)

// KnativeAPIResourceSet manages knative related objects
type KnativeAPIResourceSet struct {
}

// GetScheme returns knative scheme object
func (*KnativeAPIResourceSet) GetScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	must(knative.AddToScheme(scheme))
	return scheme
}

// CreateAPIResources converts ir object to runtime objects
func (knativeAPIResourceSet *KnativeAPIResourceSet) CreateAPIResources(oldir irtypes.IR) []runtime.Object {
	ir := irtypes.NewEnhancedIRFromIR(oldir)
	targetObjs := []runtime.Object{}
	ignoredObjs := ir.CachedObjects
	for _, apiResource := range knativeAPIResourceSet.getAPIResources(ir) {
		apiResource.SetClusterContext(ir.TargetClusterSpec)
		resourceIgnoredObjs := apiResource.LoadResources(ir.CachedObjects, ir)
		ignoredObjs = intersection(ignoredObjs, resourceIgnoredObjs)
		resourceObjs := apiResource.GetUpdatedResources(ir)
		targetObjs = append(targetObjs, resourceObjs...)
	}
	targetObjs = append(targetObjs, ignoredObjs...)
	return targetObjs
}

func (knativeAPIResourceSet *KnativeAPIResourceSet) getAPIResources(ir irtypes.EnhancedIR) []apiresource.APIResource {
	apiresources := []apiresource.APIResource{
		{
			Scheme:       knativeAPIResourceSet.GetScheme(),
			IAPIResource: &apiresource.KnativeService{Cluster: ir.TargetClusterSpec},
		},
	}
	return apiresources
}

// GetServiceOptions returns plan service options for an input folder
func (knativeAPIResourceSet *KnativeAPIResourceSet) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	services := []plantypes.Service{}

	codecs := serializer.NewCodecFactory(knativeAPIResourceSet.GetScheme())

	filePaths, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Errorf("Unable to fetch yaml files at path %q Error: %q", inputPath, err)
		return services, err
	}
	for _, filePath := range filePaths {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Debugf("Failed to read the yaml file at path %q Error: %q", filePath, err)
			continue
		}
		obj, _, err := codecs.UniversalDeserializer().Decode(data, nil, nil)
		if err != nil {
			log.Debugf("Failed to decode the file at path %q as a knative file. Error: %q", filePath, err)
			continue
		}
		d1, ok := obj.(*knativev1.Service)
		if ok {
			continue
		}
		service := newKnativeService(d1.Name)
		service.SourceArtifacts[plantypes.KnativeFileArtifactType] = []string{filePath}
		services = append(services, service)
	}
	return services, nil
}

// Translate translates plan services to IR
func (knativeAPIResourceSet *KnativeAPIResourceSet) Translate(services []plantypes.Service, p plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(p)
	ir.Services = map[string]irtypes.Service{}
	codecs := serializer.NewCodecFactory(knativeAPIResourceSet.GetScheme())

	for _, service := range services {
		if len(service.SourceArtifacts[plantypes.KnativeFileArtifactType]) == 0 {
			log.Warnf("No knative artifacts found in service %s", service.ServiceName)
			continue
		}
		irService := irtypes.NewServiceFromPlanService(service)
		filePath := service.SourceArtifacts[plantypes.KnativeFileArtifactType][0]
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Errorf("Unable to read the knative file at path %q Error: %q", filePath, err)
			continue
		}
		obj, _, err := codecs.UniversalDeserializer().Decode(data, nil, nil)
		if err != nil {
			log.Errorf("Failed to decode the knative file at path %q Error: %q", filePath, err)
			continue
		}
		d1, ok := obj.(*knativev1.Service)
		if !ok {
			log.Errorf("The knative file at path %q does not contain the required type. Expected: %T Actual: %T", filePath, new(knativev1.Service), obj)
			continue
		}
		irService.PodSpec = d1.Spec.ConfigurationSpec.Template.Spec.PodSpec
		ir.Services[service.ServiceName] = irService
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
