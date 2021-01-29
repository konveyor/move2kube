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
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	admissionregistration "k8s.io/kubernetes/pkg/apis/admissionregistration"
	apps "k8s.io/kubernetes/pkg/apis/apps"
	authentication "k8s.io/kubernetes/pkg/apis/authentication"
	authorization "k8s.io/kubernetes/pkg/apis/authorization"
	autoscaling "k8s.io/kubernetes/pkg/apis/autoscaling"
	batch "k8s.io/kubernetes/pkg/apis/batch"
	certificates "k8s.io/kubernetes/pkg/apis/certificates"
	coordination "k8s.io/kubernetes/pkg/apis/coordination"
	core "k8s.io/kubernetes/pkg/apis/core"
	discovery "k8s.io/kubernetes/pkg/apis/discovery"
	events "k8s.io/kubernetes/pkg/apis/events"
	extensions "k8s.io/kubernetes/pkg/apis/extensions"
	flowcontrol "k8s.io/kubernetes/pkg/apis/flowcontrol"
	networking "k8s.io/kubernetes/pkg/apis/networking"
	node "k8s.io/kubernetes/pkg/apis/node"
	policy "k8s.io/kubernetes/pkg/apis/policy"
	rbac "k8s.io/kubernetes/pkg/apis/rbac"
	scheduling "k8s.io/kubernetes/pkg/apis/scheduling"
	settings "k8s.io/kubernetes/pkg/apis/settings"
	storage "k8s.io/kubernetes/pkg/apis/storage"

	admissionregistrationinstall "k8s.io/kubernetes/pkg/apis/admissionregistration/install"
	appsinstall "k8s.io/kubernetes/pkg/apis/apps/install"
	authenticationinstall "k8s.io/kubernetes/pkg/apis/authentication/install"
	authorizationinstall "k8s.io/kubernetes/pkg/apis/authorization/install"
	autoscalinginstall "k8s.io/kubernetes/pkg/apis/autoscaling/install"
	batchinstall "k8s.io/kubernetes/pkg/apis/batch/install"
	certificatesinstall "k8s.io/kubernetes/pkg/apis/certificates/install"
	coordinationinstall "k8s.io/kubernetes/pkg/apis/coordination/install"
	coreinstall "k8s.io/kubernetes/pkg/apis/core/install"
	discoveryinstall "k8s.io/kubernetes/pkg/apis/discovery/install"
	eventsinstall "k8s.io/kubernetes/pkg/apis/events/install"
	extensionsinstall "k8s.io/kubernetes/pkg/apis/extensions/install"
	flowcontrolinstall "k8s.io/kubernetes/pkg/apis/flowcontrol/install"
	networkinginstall "k8s.io/kubernetes/pkg/apis/networking/install"
	nodeinstall "k8s.io/kubernetes/pkg/apis/node/install"
	policyinstall "k8s.io/kubernetes/pkg/apis/policy/install"
	rbacinstall "k8s.io/kubernetes/pkg/apis/rbac/install"
	schedulinginstall "k8s.io/kubernetes/pkg/apis/scheduling/install"
	settingsinstall "k8s.io/kubernetes/pkg/apis/settings/install"
	storageinstall "k8s.io/kubernetes/pkg/apis/storage/install"

	collecttypes "github.com/konveyor/move2kube/types/collection"
	okdapi "github.com/openshift/api"
	tektonscheme "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	k8sapischeme "k8s.io/client-go/kubernetes/scheme"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
)

var (
	scheme       = runtime.NewScheme()
	liasonscheme = runtime.NewScheme()
)

func init() {
	must(okdapi.Install(scheme))
	must(okdapi.InstallKube(scheme))

	must(k8sapischeme.AddToScheme(scheme))
	must(tektonscheme.AddToScheme(scheme))

	appsinstall.Install(scheme)
	admissionregistrationinstall.Install(scheme)
	authenticationinstall.Install(scheme)
	authorizationinstall.Install(scheme)
	autoscalinginstall.Install(scheme)
	batchinstall.Install(scheme)
	certificatesinstall.Install(scheme)
	coordinationinstall.Install(scheme)
	coreinstall.Install(scheme)
	discoveryinstall.Install(scheme)
	eventsinstall.Install(scheme)
	extensionsinstall.Install(scheme)
	flowcontrolinstall.Install(scheme)
	networkinginstall.Install(scheme)
	nodeinstall.Install(scheme)
	policyinstall.Install(scheme)
	rbacinstall.Install(scheme)
	schedulinginstall.Install(scheme)
	settingsinstall.Install(scheme)
	storageinstall.Install(scheme)

	must(apps.AddToScheme(liasonscheme))
	must(admissionregistration.AddToScheme(liasonscheme))
	must(authentication.AddToScheme(liasonscheme))
	must(authorization.AddToScheme(liasonscheme))
	must(autoscaling.AddToScheme(liasonscheme))
	must(batch.AddToScheme(liasonscheme))
	must(certificates.AddToScheme(liasonscheme))
	must(coordination.AddToScheme(liasonscheme))
	must(core.AddToScheme(liasonscheme))
	must(discovery.AddToScheme(liasonscheme))
	must(events.AddToScheme(liasonscheme))
	must(extensions.AddToScheme(liasonscheme))
	must(flowcontrol.AddToScheme(liasonscheme))
	must(networking.AddToScheme(liasonscheme))
	must(node.AddToScheme(liasonscheme))
	must(policy.AddToScheme(liasonscheme))
	must(rbac.AddToScheme(liasonscheme))
	must(scheduling.AddToScheme(liasonscheme))
	must(settings.AddToScheme(liasonscheme))
	must(storage.AddToScheme(liasonscheme))
}

// K8sAPIResourceSet for handling K8s related resources
type K8sAPIResourceSet struct {
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// GetScheme returns K8s scheme
func (*K8sAPIResourceSet) GetScheme() *runtime.Scheme {
	return scheme
}

// CreateAPIResources converts IR to runtime objects
func (k8sAPIResourceSet *K8sAPIResourceSet) CreateAPIResources(oldir irtypes.IR) []runtime.Object {
	ir := irtypes.NewEnhancedIRFromIR(oldir)
	targetObjs := []runtime.Object{}
	ignoredObjs := ir.CachedObjects
	for _, apiResource := range k8sAPIResourceSet.getAPIResources(ir) {
		apiResource.SetClusterContext(ir.TargetClusterSpec)
		resourceIgnoredObjs := apiResource.LoadResources(ir.CachedObjects, ir)
		ignoredObjs = intersection(ignoredObjs, resourceIgnoredObjs)
		resourceObjs := apiResource.GetUpdatedResources(ir)
		targetObjs = append(targetObjs, resourceObjs...)
	}
	targetObjs = append(targetObjs, ignoredObjs...)
	return targetObjs
}

func (k8sAPIResourceSet *K8sAPIResourceSet) getAPIResources(ir irtypes.EnhancedIR) []apiresource.APIResource {
	return []apiresource.APIResource{
		{
			Scheme:       k8sAPIResourceSet.GetScheme(),
			IAPIResource: &apiresource.Deployment{Cluster: ir.TargetClusterSpec},
		},
		{
			Scheme:       k8sAPIResourceSet.GetScheme(),
			IAPIResource: &apiresource.Storage{Cluster: ir.TargetClusterSpec},
		},
		{
			Scheme:       k8sAPIResourceSet.GetScheme(),
			IAPIResource: &apiresource.Service{Cluster: ir.TargetClusterSpec},
		},
		{
			Scheme:       k8sAPIResourceSet.GetScheme(),
			IAPIResource: &apiresource.ImageStream{Cluster: ir.TargetClusterSpec},
		},
		{
			Scheme:       k8sAPIResourceSet.GetScheme(),
			IAPIResource: &apiresource.NetworkPolicy{Cluster: ir.TargetClusterSpec},
		},
	}
}

// GetServiceOptions analyses a directory and returns possible plan services
func (k8sAPIResourceSet *K8sAPIResourceSet) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	services := []plantypes.Service{}
	//TODO: Should we add service analysis too, to get service name?

	codecs := serializer.NewCodecFactory(k8sAPIResourceSet.GetScheme())

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
		docs, err := common.SplitYAML(data)
		if err != nil {
			log.Errorf("Failed to split the file at path %q into YAML documents. Error: %q", filePath, err)
			continue
		}
		for _, doc := range docs {
			// filter to get only valid k8s Deployments
			obj, _, err := codecs.UniversalDeserializer().Decode(doc, nil, nil)
			if err != nil {
				continue
			}
			name, _, err := new(apiresource.Deployment).GetNameAndPodSpec(obj)
			if err != nil {
				continue
			}
			service := newK8sService(name)
			service.SourceArtifacts[plantypes.K8sFileArtifactType] = []string{filePath}
			services = append(services, service)
		}
	}
	return services, nil
}

// Translate translates plan services to IR
func (k8sAPIResourceSet *K8sAPIResourceSet) Translate(services []plantypes.Service, plan plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(plan)
	codecs := serializer.NewCodecFactory(k8sAPIResourceSet.GetScheme())

	for _, service := range services {
		if len(service.SourceArtifacts[plantypes.K8sFileArtifactType]) == 0 {
			log.Warnf("No k8s artifacts found in service %s", service.ServiceName)
			continue
		}
		filePath := service.SourceArtifacts[plantypes.K8sFileArtifactType][0] // TODO: what about the other k8s files?
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Errorf("Unable to read the k8s file at path %q Error: %q", filePath, err)
			continue
		}
		docs, err := common.SplitYAML(data)
		if err != nil {
			log.Errorf("Failed to split the k8s file at path %q into YAML documents. Error: %q", filePath, err)
			continue
		}
		for i, doc := range docs {
			obj, _, err := codecs.UniversalDeserializer().Decode(doc, nil, nil)
			if err != nil {
				log.Errorf("Failed to decode document %d of file at path %q as a k8s resource. Error: %q", i, filePath, err)
				continue
			}
			name, podSpec, err := new(apiresource.Deployment).GetNameAndPodSpec(obj)
			if err != nil {
				continue
			}
			if name != service.ServiceName {
				continue
			}
			irService := irtypes.NewServiceFromPlanService(service)
			irService.PodSpec = podSpec
			for _, container := range podSpec.Containers {
				for _, port := range container.Ports {
					podPort := irtypes.Port{Name: port.Name, Number: port.ContainerPort}
					servicePort := podPort
					irService.AddPortForwarding(servicePort, podPort)
				}
			}
			ir.Services[service.ServiceName] = irService
			break
		}
	}
	return ir, nil
}

func newK8sService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, plantypes.Kube2KubeTranslation)
	service.ContainerBuildType = plantypes.ReuseContainerBuildTypeValue
	service.AddSourceType(plantypes.K8sSourceTypeValue)
	service.UpdateContainerBuildPipeline = false
	service.UpdateDeployPipeline = true
	return service
}

// ConvertToSupportedVersion converts obj to a supported Version
func (k8sAPIResourceSet *K8sAPIResourceSet) ConvertToSupportedVersion(obj runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec) (newobj runtime.Object, err error) {
	objvk := obj.GetObjectKind().GroupVersionKind()
	objgv := objvk.GroupVersion()
	kind := objvk.Kind
	fixFn := fixFuncs[kind]
	versions := clusterSpec.GetSupportedVersions(kind)
	if versions == nil || len(versions) == 0 {
		return nil, fmt.Errorf("Kind %s unsupported in target cluster : %+v", kind, obj.GetObjectKind())
	}
	for _, v := range versions {
		if kind == "Service" && strings.HasPrefix(v, knativev1.SchemeGroupVersion.Group) {
			continue
		}
		groupversion, err := schema.ParseGroupVersion(v)
		if err != nil {
			log.Errorf("Unable to parse group version %s : %s", v, err)
			continue
		}
		if fixFn == nil {
			if objgv == groupversion {
				scheme.Default(obj)
				return obj, nil
			}
			//Change to supported version
			newobj, err := scheme.ConvertToVersion(obj, groupversion)
			if err == nil {
				scheme.Default(newobj)
				return newobj, nil
			}
			log.Debugf("Unable to do direct translation : %s", err)
		}
		akt := liasonscheme.AllKnownTypes()
		uvcreated := false
		for kt := range akt {
			if kind == kt.Kind {
				log.Debugf("Attempting conversion of %s obj to %s", objgv, kt)
				uvobj, err := liasonscheme.New(kt)
				if err != nil {
					log.Errorf("Unable to create obj of type %+v : %s", kt, err)
					continue
				}
				err = scheme.Convert(obj, uvobj, nil)
				if err != nil {
					log.Errorf("Unable to convert to unversioned object : %s", err)
					continue
				}
				uvcreated = true
				if fixFn != nil {
					fuvobj, err := fixFn(uvobj)
					if err != nil {
						log.Errorf("Error while executing fix function : %s", err)
					} else {
						uvobj = fuvobj
					}
				}
				obj = uvobj
				log.Debugf("Converted %s obj to %s", objgv, kt)
				break
			}
		}
		if !uvcreated {
			log.Errorf("Unable to convert to unversioned object. Will try conversion as it is : %s", objgv)
		}
		//Change to supported version
		newobj, err = scheme.ConvertToVersion(obj, groupversion)
		if err != nil {
			log.Errorf("Error while transforming version : %s", err)
			continue
		}
		scheme.Default(newobj)
		return newobj, nil
	}
	return nil, fmt.Errorf("Unable to convert to a supported version : %+v", obj.GetObjectKind())
}
