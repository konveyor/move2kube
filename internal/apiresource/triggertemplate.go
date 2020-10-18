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

package apiresource

import (
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/internal/types/tekton"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	triggerTemplateKind          = "TriggerTemplate"
	pipelineRunKind              = "PipelineRun"
	registryNamespacePlaceholder = "<TODO: insert your registry namespace>"
)

// TriggerTemplate handles all objects like an trigger template.
type TriggerTemplate struct {
}

// GetSupportedKinds returns the kinds that this type supports.
func (*TriggerTemplate) GetSupportedKinds() []string {
	return []string{triggerTemplateKind}
}

// CreateNewResources creates the runtime objects from the intermediate representation.
func (tt *TriggerTemplate) CreateNewResources(ir irtypes.IR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	if common.IsStringPresent(supportedKinds, triggerTemplateKind) {
		irresources := ir.TektonResources.TriggerTemplates
		for _, irresource := range irresources {
			objs = append(objs, tt.createNewResource(irresource, ir))
		}
	} else {
		log.Errorf("Could not find a valid resource type in cluster to create a trigger template.")
	}
	return objs
}

func (*TriggerTemplate) createNewResource(tt tekton.TriggerTemplate, ir irtypes.IR) *triggersv1alpha1.TriggerTemplate {
	registryURL := ir.Kubernetes.RegistryURL
	registryNamespace := ir.Kubernetes.RegistryNamespace
	if registryURL == "" {
		registryURL = common.DefaultRegistryURL
	}
	if registryNamespace == "" {
		registryNamespace = registryNamespacePlaceholder
	}

	// pipelinerun
	pipelineRun := new(v1beta1.PipelineRun)
	pipelineRun.TypeMeta = metav1.TypeMeta{
		Kind:       pipelineRunKind,
		APIVersion: v1beta1.SchemeGroupVersion.String(),
	}
	pipelineRun.ObjectMeta = metav1.ObjectMeta{Name: tt.PipelineRunName}
	pipelineRun.Spec = v1beta1.PipelineRunSpec{
		PipelineRef:        &v1beta1.PipelineRef{Name: tt.PipelineName},
		ServiceAccountName: tt.ServiceAccountName,
		Workspaces: []v1beta1.WorkspaceBinding{
			v1beta1.WorkspaceBinding{
				Name: tt.WorkspaceName,
				VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						StorageClassName: &tt.StorageClassName,
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources:        corev1.ResourceRequirements{Requests: corev1.ResourceList{"storage": resource.MustParse("1Gi")}},
					},
				},
			},
		},
		Params: []v1beta1.Param{
			v1beta1.Param{Name: "image-registry-url", Value: v1beta1.ArrayOrString{Type: "string", StringVal: registryURL + "/" + registryNamespace}},
		},
	}

	// trigger template
	triggerTemplate := new(triggersv1alpha1.TriggerTemplate)
	triggerTemplate.TypeMeta = metav1.TypeMeta{
		Kind:       triggerTemplateKind,
		APIVersion: triggersv1alpha1.SchemeGroupVersion.String(),
	}
	triggerTemplate.ObjectMeta = metav1.ObjectMeta{Name: tt.Name}

	triggerTemplate.Spec = triggersv1alpha1.TriggerTemplateSpec{
		ResourceTemplates: []triggersv1alpha1.TriggerResourceTemplate{
			triggersv1alpha1.TriggerResourceTemplate{
				RawExtension: runtime.RawExtension{Object: pipelineRun},
			},
		},
	}

	return triggerTemplate
}

// ConvertToClusterSupportedKinds converts the object to supported types if possible.
func (tt *TriggerTemplate) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object) ([]runtime.Object, bool) {
	supKinds := tt.GetSupportedKinds()
	for _, supKind := range supKinds {
		if common.IsStringPresent(supportedKinds, supKind) {
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}
