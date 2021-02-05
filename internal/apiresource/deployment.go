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
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/k8sschema"
	internaltypes "github.com/konveyor/move2kube/internal/types"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	okdappsv1 "github.com/openshift/api/apps/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apps "k8s.io/kubernetes/pkg/apis/apps"
	batch "k8s.io/kubernetes/pkg/apis/batch"
	core "k8s.io/kubernetes/pkg/apis/core"
)

//TODO: Add support for replicaset, cronjob ad statefulset

const (
	// podKind defines Pod Kind
	podKind string = "Pod"
	// jobKind defines Job Kind
	jobKind string = "Job"
	// deploymentConfigKind defines DeploymentConfig Kind
	deploymentConfigKind string = "DeploymentConfig"
	// replicationControllerKind defines ReplicationController Kind
	replicationControllerKind string = "ReplicationController"
	// daemonSetKind defines DaemonSet Kind
	daemonSetKind string = "DaemonSet"
)

// Deployment handles all objects like a Deployment
type Deployment struct {
	Cluster collecttypes.ClusterMetadataSpec
}

// getSupportedKinds returns kinds supported by the deployment
func (d *Deployment) getSupportedKinds() []string {
	return []string{podKind, jobKind, common.DeploymentKind, deploymentConfigKind, replicationControllerKind}
}

// createNewResources converts ir to runtime object
func (d *Deployment) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	for _, service := range ir.Services {
		var obj runtime.Object
		if service.Daemon {
			if common.IsStringPresent(supportedKinds, daemonSetKind) {
				obj = d.createDaemonSet(service)
			} else {
				log.Errorf("Could not find a valid resource type in cluster to create a daemon set.")
			}
		} else if service.RestartPolicy == core.RestartPolicyNever || service.RestartPolicy == core.RestartPolicyOnFailure {
			if common.IsStringPresent(supportedKinds, jobKind) {
				obj = d.createJob(service)
			} else if common.IsStringPresent(supportedKinds, podKind) {
				pod := d.createPod(service)
				pod.Spec.RestartPolicy = core.RestartPolicyOnFailure
				obj = pod
			} else {
				log.Errorf("Could not find a valid resource type in cluster to create a job/pod.")
			}
		} else if common.IsStringPresent(supportedKinds, deploymentConfigKind) {
			obj = d.createDeploymentConfig(service)
		} else if common.IsStringPresent(supportedKinds, common.DeploymentKind) {
			obj = d.createDeployment(service)
		} else if common.IsStringPresent(supportedKinds, replicationControllerKind) {
			obj = d.createReplicationController(service)
		} else if common.IsStringPresent(supportedKinds, podKind) {
			obj = d.createPod(service)
		} else {
			log.Errorf("Could not find a valid resource type in cluster to create a deployment")
		}
		if obj != nil {
			objs = append(objs, obj)
		}
	}
	return objs
}

// convertToClusterSupportedKinds converts objects to kind supported by the cluster
func (d *Deployment) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR) ([]runtime.Object, bool) {
	if d1, ok := obj.(*apps.DaemonSet); ok {
		if common.IsStringPresent(supportedKinds, daemonSetKind) {
			return []runtime.Object{d1}, true
		}
		return nil, false
	}
	if d1, ok := obj.(*core.Pod); ok && (d1.Spec.RestartPolicy == core.RestartPolicyOnFailure || d1.Spec.RestartPolicy == core.RestartPolicyNever) {
		if common.IsStringPresent(supportedKinds, jobKind) {
			return []runtime.Object{d.podToJob(*d1)}, true
		}
		return []runtime.Object{obj}, true
	} else if d1, ok := obj.(*batch.Job); ok && !common.IsStringPresent(supportedKinds, jobKind) {
		if !common.IsStringPresent(supportedKinds, jobKind) && common.IsStringPresent(supportedKinds, podKind) {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, core.RestartPolicyOnFailure)}, true
		}
		log.Warnf("Both Job and Pod not supported. No other valid way to translate this object. : %+v", obj)
		return []runtime.Object{obj}, true
	}
	if common.IsStringPresent(supportedKinds, deploymentConfigKind) {
		if d1, ok := obj.(*apps.Deployment); ok {
			return []runtime.Object{d.toDeploymentConfig(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*core.ReplicationController); ok {
			return []runtime.Object{d.toDeploymentConfig(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*core.Pod); ok {
			var replicas int32 = 2
			return []runtime.Object{d.toDeploymentConfig(d1.ObjectMeta, d1.Spec, replicas)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsStringPresent(supportedKinds, common.DeploymentKind) {
		if d1, ok := obj.(*okdappsv1.DeploymentConfig); ok {
			return []runtime.Object{d.toDeployment(d1.ObjectMeta, k8sschema.ConvertToPodSpec(&d1.Spec.Template.Spec), d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*core.ReplicationController); ok {
			return []runtime.Object{d.toDeployment(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*core.Pod); ok {
			var replicas int32 = 2
			return []runtime.Object{d.toDeployment(d1.ObjectMeta, d1.Spec, replicas)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsStringPresent(supportedKinds, replicationControllerKind) {
		if d1, ok := obj.(*okdappsv1.DeploymentConfig); ok {
			return []runtime.Object{d.toReplicationController(d1.ObjectMeta, k8sschema.ConvertToPodSpec(&d1.Spec.Template.Spec), d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*apps.Deployment); ok {
			return []runtime.Object{d.toReplicationController(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*core.Pod); ok {
			var replicas int32 = 2
			return []runtime.Object{d.toReplicationController(d1.ObjectMeta, d1.Spec, replicas)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsStringPresent(supportedKinds, podKind) {
		if d1, ok := obj.(*okdappsv1.DeploymentConfig); ok {
			return []runtime.Object{d.toPod(d1.ObjectMeta, k8sschema.ConvertToPodSpec(&d1.Spec.Template.Spec), core.RestartPolicyAlways)}, true
		} else if d1, ok := obj.(*apps.Deployment); ok {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, core.RestartPolicyAlways)}, true
		} else if d1, ok := obj.(*core.ReplicationController); ok {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, core.RestartPolicyAlways)}, true
		}
		return []runtime.Object{obj}, true
	}
	return nil, false
}

// Create section

func (d *Deployment) createDeployment(service irtypes.Service) *apps.Deployment {

	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	podSpec := service.PodSpec
	podSpec = d.convertVolumesKindsByPolicy(podSpec)
	podSpec.RestartPolicy = core.RestartPolicyAlways
	log.Debugf("Created deployment for %s", service.Name)
	return d.toDeployment(meta, podSpec, int32(service.Replicas))
}

func (d *Deployment) createDeploymentConfig(service irtypes.Service) *okdappsv1.DeploymentConfig {
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	podSpec := service.PodSpec
	podSpec = d.convertVolumesKindsByPolicy(podSpec)
	podSpec.RestartPolicy = core.RestartPolicyAlways
	log.Debugf("Created DeploymentConfig for %s", service.Name)
	return d.toDeploymentConfig(meta, podSpec, int32(service.Replicas))
}

// createReplicationController initializes Kubernetes ReplicationController object
func (d *Deployment) createReplicationController(service internaltypes.Service) *core.ReplicationController {
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	podSpec := service.PodSpec
	podSpec = d.convertVolumesKindsByPolicy(podSpec)
	podSpec.RestartPolicy = core.RestartPolicyAlways
	log.Debugf("Created DeploymentConfig for %s", service.Name)
	return d.toReplicationController(meta, podSpec, int32(service.Replicas))
}

func (d *Deployment) createPod(service irtypes.Service) *core.Pod {
	podSpec := service.PodSpec
	podSpec = d.convertVolumesKindsByPolicy(podSpec)
	podSpec.RestartPolicy = core.RestartPolicyAlways
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	return d.toPod(meta, podSpec, podSpec.RestartPolicy)
}

func (d *Deployment) createDaemonSet(service irtypes.Service) *apps.DaemonSet {
	podSpec := service.PodSpec
	podSpec = d.convertVolumesKindsByPolicy(podSpec)
	podSpec.RestartPolicy = core.RestartPolicyAlways
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	pod := apps.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       daemonSetKind,
			APIVersion: metav1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec: apps.DaemonSetSpec{
			Template: core.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       podSpec,
			},
		},
	}
	return &pod
}

func (d *Deployment) createJob(service irtypes.Service) *batch.Job {
	podspec := service.PodSpec
	podspec = d.convertVolumesKindsByPolicy(podspec)
	podspec.RestartPolicy = core.RestartPolicyOnFailure
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	pod := batch.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       jobKind,
			APIVersion: batch.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec: batch.JobSpec{
			Template: core.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       podspec,
			},
		},
	}
	return &pod
}

// Conversions section

func (d *Deployment) toDeploymentConfig(meta metav1.ObjectMeta, podspec core.PodSpec, replicas int32) *okdappsv1.DeploymentConfig {
	podspec = d.convertVolumesKindsByPolicy(podspec)
	triggerPolicies := []okdappsv1.DeploymentTriggerPolicy{{
		Type: okdappsv1.DeploymentTriggerOnConfigChange,
	}}
	for _, container := range podspec.Containers {
		imageStreamName, imageStreamTag := new(ImageStream).GetImageStreamNameAndTag(container.Image)
		triggerPolicies = append(triggerPolicies, okdappsv1.DeploymentTriggerPolicy{
			Type: okdappsv1.DeploymentTriggerOnImageChange,
			ImageChangeParams: &okdappsv1.DeploymentTriggerImageChangeParams{
				Automatic:      true,
				ContainerNames: []string{container.Name},
				From: corev1.ObjectReference{
					Kind: "ImageStreamTag",
					Name: imageStreamName + ":" + imageStreamTag,
				},
			},
		})
	}
	dc := okdappsv1.DeploymentConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       deploymentConfigKind,
			APIVersion: okdappsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec: okdappsv1.DeploymentConfigSpec{
			Replicas: int32(replicas),
			Selector: getServiceLabels(meta.Name),
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       k8sschema.ConvertToV1PodSpec(&podspec), // obj.Spec.Template.Spec,
			},
			Triggers: triggerPolicies,
		},
	}
	return &dc
}

func (d *Deployment) toDeployment(meta metav1.ObjectMeta, podspec core.PodSpec, replicas int32) *apps.Deployment {
	podspec = d.convertVolumesKindsByPolicy(podspec)
	dc := &apps.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       common.DeploymentKind,
			APIVersion: apps.SchemeGroupVersion.String(), //"apps/v1",
		},
		ObjectMeta: meta,
		Spec: apps.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getServiceLabels(meta.Name),
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       podspec,
			},
		},
	}
	return dc
}

// toReplicationController initializes Kubernetes ReplicationController object
func (d *Deployment) toReplicationController(meta metav1.ObjectMeta, podspec core.PodSpec, replicas int32) *core.ReplicationController {
	podspec = d.convertVolumesKindsByPolicy(podspec)
	nReplicas := int32(replicas)
	rc := &core.ReplicationController{
		TypeMeta: metav1.TypeMeta{
			Kind:       replicationControllerKind,
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec: core.ReplicationControllerSpec{
			Replicas: nReplicas,
			Selector: getServiceLabels(meta.Name),
			Template: &core.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       podspec,
			},
		},
	}
	return rc
}

func (d *Deployment) podToJob(obj core.Pod) *batch.Job {
	podspec := obj.Spec
	podspec = d.convertVolumesKindsByPolicy(podspec)
	podspec.RestartPolicy = core.RestartPolicyOnFailure
	pod := batch.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       jobKind,
			APIVersion: batch.SchemeGroupVersion.String(),
		},
		ObjectMeta: obj.ObjectMeta,
		Spec: batch.JobSpec{
			Template: core.PodTemplateSpec{
				ObjectMeta: obj.ObjectMeta,
				Spec:       podspec,
			},
		},
	}
	return &pod
}

func (d *Deployment) toPod(meta metav1.ObjectMeta, podspec core.PodSpec, restartPolicy core.RestartPolicy) *core.Pod {
	podspec = d.convertVolumesKindsByPolicy(podspec)
	podspec.RestartPolicy = restartPolicy
	pod := core.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       podKind,
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec:       podspec,
	}
	return &pod
}

//Volumes and volume mounts of all containers are translated as follows:
//1. Each container's volume mount list and corresponding volumes are translated
//2. Unreferenced volumes are discarded
func (d *Deployment) convertVolumesKindsByPolicy(podspec core.PodSpec) core.PodSpec {
	if podspec.Volumes == nil || len(podspec.Volumes) == 0 {
		return podspec
	}

	// Remove unused volumes
	for _, container := range podspec.Containers {
		volMounts := []core.VolumeMount{}
		for _, vm := range container.VolumeMounts {
			volume := getMatchingVolume(vm, podspec.Volumes)
			if volume == (core.Volume{}) {
				log.Warnf("Couldn't find a corresponding volume for volume mount %s", vm.Name)
				continue
			}
			volMounts = append(volMounts, vm)
		}
		container.VolumeMounts = volMounts
	}

	volumes := []core.Volume{}
	for _, v := range podspec.Volumes {
		volumes = append(volumes, convertVolumeBySupportedKind(v, d.Cluster))
	}
	podspec.Volumes = volumes

	return podspec
}

func getMatchingVolume(vm core.VolumeMount, vList []core.Volume) core.Volume {
	for _, v := range vList {
		if strings.Compare(v.Name, vm.Name) == 0 {
			return v
		}
	}
	return core.Volume{}
}
