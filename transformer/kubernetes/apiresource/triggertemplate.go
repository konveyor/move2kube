/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package apiresource

import (
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
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

// getSupportedKinds returns the kinds that this type supports.
func (*TriggerTemplate) getSupportedKinds() []string {
	return []string{triggerTemplateKind}
}

// createNewResources creates the runtime objects from the intermediate representation.
func (tt *TriggerTemplate) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	// Since tekton is an extension, the tekton resources are put in a separate folder from the main application.
	// We ignore supported kinds because these resources are optional and it's upto the user to install the extension if they need it.
	irresources := ir.TektonResources.TriggerTemplates
	for _, irresource := range irresources {
		objs = append(objs, tt.createNewResource(irresource, ir))
	}
	return objs
}

func (*TriggerTemplate) createNewResource(tt irtypes.TriggerTemplate, ir irtypes.EnhancedIR) *triggersv1beta1.TriggerTemplate {
	registryURL := commonqa.ImageRegistry()
	registryNamespace := commonqa.ImageRegistryNamespace()

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
			{
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
			{Name: "image-registry-url", Value: v1beta1.ArrayOrString{Type: "string", StringVal: registryURL + "/" + registryNamespace}},
		},
	}
	gitRepoSSHSecretName := qaengine.FetchStringAnswer(
		common.ConfigCICDTektonGitRepoSSHSecretNameKey,
		"Enter the name of an existing K8s secret that has ssh credentials for cloning the git repo",
		[]string{"If this is not relevant to you then give an empty string to remove it from the YAML."},
		"",
		nil,
	)
	if gitRepoSSHSecretName != "" {
		pipelineRun.Spec.Workspaces = append(pipelineRun.Spec.Workspaces, v1beta1.WorkspaceBinding{
			Name: gitRepoSSHCredsWorkspace,
			Secret: &corev1.SecretVolumeSource{
				SecretName: gitRepoSSHSecretName,
			},
		})
	}
	gitRepoBasicAuthSecretName := qaengine.FetchStringAnswer(
		common.ConfigCICDTektonGitRepoBasicAuthSecretNameKey,
		"Enter the name of an existing K8s secret that has username and password for cloning the git repo",
		[]string{"If this is not relevant to you then give an empty string to remove it from the YAML."},
		"",
		nil,
	)
	if gitRepoBasicAuthSecretName != "" {
		pipelineRun.Spec.Workspaces = append(pipelineRun.Spec.Workspaces, v1beta1.WorkspaceBinding{
			Name: gitRepoBasicAuthCredsWorkspace,
			Secret: &corev1.SecretVolumeSource{
				SecretName: gitRepoBasicAuthSecretName,
			},
		})
	}
	imageRegistryAuthSecretName := qaengine.FetchStringAnswer(
		common.ConfigCICDTektonRegistryPushSecretNameKey,
		"Enter the name of an existing K8s secret that has Docker config.json for pushing images to the registry",
		[]string{"If this is not relevant to you then give an empty string to remove it from the YAML."},
		"",
		nil,
	)
	if imageRegistryAuthSecretName != "" {
		pipelineRun.Spec.Workspaces = append(pipelineRun.Spec.Workspaces, v1beta1.WorkspaceBinding{
			Name: registryCredsWorkspace,
			Secret: &corev1.SecretVolumeSource{
				SecretName: imageRegistryAuthSecretName,
			},
		})
	}
	// trigger template
	triggerTemplate := new(triggersv1beta1.TriggerTemplate)
	triggerTemplate.TypeMeta = metav1.TypeMeta{
		Kind:       triggerTemplateKind,
		APIVersion: triggersv1beta1.SchemeGroupVersion.String(),
	}
	triggerTemplate.ObjectMeta = metav1.ObjectMeta{Name: tt.Name}

	triggerTemplate.Spec = triggersv1beta1.TriggerTemplateSpec{
		ResourceTemplates: []triggersv1beta1.TriggerResourceTemplate{
			{
				RawExtension: runtime.RawExtension{Object: pipelineRun},
			},
		},
	}

	return triggerTemplate
}

// convertToClusterSupportedKinds converts the object to supported types if possible.
func (tt *TriggerTemplate) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if common.IsPresent(tt.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}
