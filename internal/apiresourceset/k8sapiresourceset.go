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

	okdapi "github.com/openshift/api"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	k8sapischeme "k8s.io/client-go/kubernetes/scheme"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	tektonscheme "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
)

// K8sAPIResourceSet for handling K8s related resources
type K8sAPIResourceSet struct {
}

// GetScheme returns K8s scheme
func (k *K8sAPIResourceSet) GetScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = okdapi.Install(scheme)
	_ = okdapi.InstallKube(scheme)
	_ = k8sapischeme.AddToScheme(scheme)
	_ = tektonscheme.AddToScheme(scheme)
	return scheme
}

func (k *K8sAPIResourceSet) getAPIResources(ir irtypes.IR) []apiresource.APIResource {
	apiresources := []apiresource.APIResource{{IAPIResource: &apiresource.Deployment{ClusterSpec: ir.TargetClusterSpec}}, {IAPIResource: &apiresource.Storage{Cluster: ir.TargetClusterSpec}}, {IAPIResource: &apiresource.Service{Cluster: ir.TargetClusterSpec}}, {IAPIResource: &apiresource.ImageStream{Cluster: ir.TargetClusterSpec}}, {IAPIResource: &apiresource.NetworkPolicy{Cluster: ir.TargetClusterSpec}}}
	return apiresources
}

// CreateAPIResources converts IR to runtime objects
func (k *K8sAPIResourceSet) CreateAPIResources(ir irtypes.IR) []runtime.Object {
	targetobjs := []runtime.Object{}
	ignoredobjs := ir.CachedObjects
	for _, a := range k.getAPIResources(ir) {
		a.SetClusterContext(ir.TargetClusterSpec)
		resourceignoredobjs := a.LoadResources(ir.CachedObjects)
		ignoredobjs = intersection(ignoredobjs, resourceignoredobjs)
		objs := a.GetUpdatedResources(ir)
		targetobjs = append(targetobjs, objs...)
	}
	targetobjs = append(targetobjs, ignoredobjs...)
	return targetobjs
}

// GetServiceOptions analyses a directory and returns possible plan services
func (k *K8sAPIResourceSet) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	services := make([]plantypes.Service, 0)
	//TODO: Should we add service analysis too, to get service name?

	codecs := serializer.NewCodecFactory(k.GetScheme())

	files, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Warnf("Unable to fetch yaml files and recognize k8 yamls : %s", err)
	}
	for _, path := range files {
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
			name, _, err := (&apiresource.Deployment{}).GetNameAndPodSpec(obj)
			if err == nil {
				service := newK8sService(name)
				relpath, _ := plan.GetRelativePath(path)
				service.SourceArtifacts[plantypes.K8sFileArtifactType] = []string{relpath}
				services = append(services, service)
			}
		}
	}
	return services, nil
}

// Translate tanslates plan services to IR
func (k *K8sAPIResourceSet) Translate(services []plantypes.Service, p plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(p)
	ir.Services = make(map[string]irtypes.Service)
	codecs := serializer.NewCodecFactory(k.GetScheme())

	for _, service := range services {
		irservice := irtypes.Service{Name: service.ServiceName}
		if len(service.SourceArtifacts[plantypes.K8sFileArtifactType]) > 0 {
			fullpath := p.GetFullPath(service.SourceArtifacts[plantypes.K8sFileArtifactType][0])
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
			_, podSpec, err := (&apiresource.Deployment{}).GetNameAndPodSpec(obj)
			if err == nil {
				irservice.PodSpec = podSpec
			}
		} else {
			log.Warnf("No k8s artifacts found in service %s", service.ServiceName)
		}
		ir.Services[service.ServiceName] = irservice
	}
	return ir, nil
}

func newK8sService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, plantypes.Kube2KubeTranslation)
	service.ContainerBuildType = plantypes.ReuseContainerBuildTypeValue
	service.SourceTypes = []plantypes.SourceTypeValue{plantypes.K8sSourceTypeValue}
	service.UpdateContainerBuildPipeline = false
	service.UpdateDeployPipeline = true
	return service
}
