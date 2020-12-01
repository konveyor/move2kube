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
	"fmt"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	internaltypes "github.com/konveyor/move2kube/internal/types"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	okdappsv1 "github.com/openshift/api/apps/v1"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//TODO: Add support for replicaset, cronjob ad statefulset

const (
	// podKind defines Pod Kind
	podKind string = "Pod"
	// jobKind defines Job Kind
	jobKind string = "Job"
	// deploymentKind defines Deployment Kind
	deploymentKind string = "Deployment"
	// deploymentConfigKind defines DeploymentConfig Kind
	deploymentConfigKind string = "DeploymentConfig"
	// replicationControllerKind defines ReplicationController Kind
	replicationControllerKind string = "ReplicationController"
	// daemonSetKind defines DaemonSet Kind
	daemonSetKind string = "DaemonSet"
)

// Deployment handles all objects like a Deployment
type Deployment struct {
	ClusterSpec collecttypes.ClusterMetadataSpec
}

// GetSupportedKinds returns kinds supported by the deployment
func (d *Deployment) GetSupportedKinds() []string {
	return []string{podKind, jobKind, deploymentKind, deploymentConfigKind, replicationControllerKind}
}

// CreateNewResources converts ir to runtime object
func (d *Deployment) CreateNewResources(ir irtypes.EnhancedIR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	for _, service := range ir.Services {
		var obj runtime.Object
		if service.Daemon {
			if common.IsStringPresent(supportedKinds, daemonSetKind) {
				obj = d.createDaemonSet(service)
			} else {
				log.Errorf("Could not find a valid resource type in cluster to create a daemon set.")
			}
		} else if service.RestartPolicy == corev1.RestartPolicyNever || service.RestartPolicy == corev1.RestartPolicyOnFailure {
			if common.IsStringPresent(supportedKinds, jobKind) {
				obj = d.createJob(service)
			} else if common.IsStringPresent(supportedKinds, podKind) {
				pod := d.createPod(service)
				pod.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
				obj = pod
			} else {
				log.Errorf("Could not find a valid resource type in cluster to create a job/pod.")
			}
		} else if common.IsStringPresent(supportedKinds, deploymentConfigKind) {
			obj = d.createDeploymentConfig(service)
		} else if common.IsStringPresent(supportedKinds, deploymentKind) {
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

// ConvertToClusterSupportedKinds converts objects to kind supported by the cluster
func (d *Deployment) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR) ([]runtime.Object, bool) {
	if d1, ok := obj.(*appsv1.DaemonSet); ok {
		if common.IsStringPresent(supportedKinds, daemonSetKind) {
			return []runtime.Object{d1}, true
		}
		return nil, false
	}
	if d1, ok := obj.(*corev1.Pod); ok && (d1.Spec.RestartPolicy == corev1.RestartPolicyOnFailure || d1.Spec.RestartPolicy == corev1.RestartPolicyNever) {
		if common.IsStringPresent(supportedKinds, jobKind) {
			return []runtime.Object{d.podToJob(*d1)}, true
		}
		return []runtime.Object{obj}, true
	} else if d1, ok := obj.(*batchv1.Job); ok && !common.IsStringPresent(supportedKinds, jobKind) {
		if !common.IsStringPresent(supportedKinds, jobKind) && common.IsStringPresent(supportedKinds, podKind) {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, corev1.RestartPolicyOnFailure)}, true
		}
		log.Warnf("Both Job and Pod not supported. No other valid way to translate this object. : %+v", obj)
		return []runtime.Object{obj}, true
	}
	if common.IsStringPresent(supportedKinds, deploymentConfigKind) {
		if d1, ok := obj.(*appsv1.Deployment); ok {
			return []runtime.Object{d.toDeploymentConfig(d1.ObjectMeta, d1.Spec.Template.Spec, *d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*corev1.ReplicationController); ok {
			return []runtime.Object{d.toDeploymentConfig(d1.ObjectMeta, d1.Spec.Template.Spec, *d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*corev1.Pod); ok {
			var replicas int32 = 2
			return []runtime.Object{d.toDeploymentConfig(d1.ObjectMeta, d1.Spec, replicas)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsStringPresent(supportedKinds, deploymentKind) {
		if d1, ok := obj.(*okdappsv1.DeploymentConfig); ok {
			return []runtime.Object{d.toDeployment(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*corev1.ReplicationController); ok {
			return []runtime.Object{d.toDeployment(d1.ObjectMeta, d1.Spec.Template.Spec, *d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*corev1.Pod); ok {
			var replicas int32 = 2
			return []runtime.Object{d.toDeployment(d1.ObjectMeta, d1.Spec, replicas)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsStringPresent(supportedKinds, replicationControllerKind) {
		if d1, ok := obj.(*okdappsv1.DeploymentConfig); ok {
			return []runtime.Object{d.toReplicationController(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*appsv1.Deployment); ok {
			return []runtime.Object{d.toReplicationController(d1.ObjectMeta, d1.Spec.Template.Spec, *d1.Spec.Replicas)}, true
		} else if d1, ok := obj.(*corev1.Pod); ok {
			var replicas int32 = 2
			return []runtime.Object{d.toReplicationController(d1.ObjectMeta, d1.Spec, replicas)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsStringPresent(supportedKinds, podKind) {
		if d1, ok := obj.(*okdappsv1.DeploymentConfig); ok {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, corev1.RestartPolicyAlways)}, true
		} else if d1, ok := obj.(*appsv1.Deployment); ok {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, corev1.RestartPolicyAlways)}, true
		} else if d1, ok := obj.(*corev1.ReplicationController); ok {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, corev1.RestartPolicyAlways)}, true
		}
		return []runtime.Object{obj}, true
	}
	return nil, false
}

// GetNameAndPodSpec returns the name and podspec used by the deployment
func (d *Deployment) GetNameAndPodSpec(obj runtime.Object) (name string, podSpec v1.PodSpec, err error) {
	switch d1 := obj.(type) {
	case *okdappsv1.DeploymentConfig:
		return d1.Name, d1.Spec.Template.Spec, nil
	case *appsv1.Deployment:
		return d1.Name, d1.Spec.Template.Spec, nil
	case *corev1.ReplicationController:
		return d1.Name, d1.Spec.Template.Spec, nil
	case *corev1.Pod:
		return d1.Name, d1.Spec, nil
	case *batchv1.Job:
		return d1.Name, d1.Spec.Template.Spec, nil
	case *appsv1.DaemonSet:
		return d1.Name, d1.Spec.Template.Spec, nil
	default:
		return "", v1.PodSpec{}, fmt.Errorf("Incompatible object type")
	}
}

// Create section

func (d *Deployment) createDeployment(service irtypes.Service) *appsv1.Deployment {

	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	podSpec := service.PodSpec
	podSpec = d.convertVolumesKindsByPolicy(podSpec)
	podSpec.RestartPolicy = corev1.RestartPolicyAlways
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
	podSpec.RestartPolicy = corev1.RestartPolicyAlways
	log.Debugf("Created DeploymentConfig for %s", service.Name)
	return d.toDeploymentConfig(meta, podSpec, int32(service.Replicas))
}

// createReplicationController initializes Kubernetes ReplicationController object
func (d *Deployment) createReplicationController(service internaltypes.Service) *corev1.ReplicationController {
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	podSpec := service.PodSpec
	podSpec = d.convertVolumesKindsByPolicy(podSpec)
	podSpec.RestartPolicy = corev1.RestartPolicyAlways
	log.Debugf("Created DeploymentConfig for %s", service.Name)
	return d.toReplicationController(meta, podSpec, int32(service.Replicas))
}

func (d *Deployment) createPod(service irtypes.Service) *corev1.Pod {
	podSpec := service.PodSpec
	podSpec = d.convertVolumesKindsByPolicy(podSpec)
	podSpec.RestartPolicy = corev1.RestartPolicyAlways
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	return d.toPod(meta, podSpec, podSpec.RestartPolicy)
}

func (d *Deployment) createDaemonSet(service irtypes.Service) *appsv1.DaemonSet {
	podSpec := service.PodSpec
	podSpec = d.convertVolumesKindsByPolicy(podSpec)
	podSpec.RestartPolicy = corev1.RestartPolicyAlways
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	pod := appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       daemonSetKind,
			APIVersion: metav1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       podSpec,
			},
		},
	}
	return &pod
}

func (d *Deployment) createJob(service irtypes.Service) *batchv1.Job {
	podspec := service.PodSpec
	podspec = d.convertVolumesKindsByPolicy(podspec)
	podspec.RestartPolicy = corev1.RestartPolicyOnFailure
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	pod := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       jobKind,
			APIVersion: batchv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       podspec,
			},
		},
	}
	return &pod
}

// Conversions section

func (d *Deployment) toDeploymentConfig(meta metav1.ObjectMeta, podspec corev1.PodSpec, replicas int32) *okdappsv1.DeploymentConfig {
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
				Spec:       podspec, // obj.Spec.Template.Spec,
			},
			Triggers: triggerPolicies,
		},
	}
	return &dc
}

func (d *Deployment) toDeployment(meta metav1.ObjectMeta, podspec corev1.PodSpec, replicas int32) *appsv1.Deployment {
	podspec = d.convertVolumesKindsByPolicy(podspec)
	dc := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       deploymentKind,
			APIVersion: appsv1.SchemeGroupVersion.String(), //"apps/v1",
		},
		ObjectMeta: meta,
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getServiceLabels(meta.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       podspec,
			},
		},
	}
	return dc
}

// toReplicationController initializes Kubernetes ReplicationController object
func (d *Deployment) toReplicationController(meta metav1.ObjectMeta, podspec corev1.PodSpec, replicas int32) *corev1.ReplicationController {
	podspec = d.convertVolumesKindsByPolicy(podspec)
	nReplicas := int32(replicas)
	rc := &corev1.ReplicationController{
		TypeMeta: metav1.TypeMeta{
			Kind:       replicationControllerKind,
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec: corev1.ReplicationControllerSpec{
			Replicas: &nReplicas,
			Selector: getServiceLabels(meta.Name),
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       podspec,
			},
		},
	}
	return rc
}

func (d *Deployment) podToJob(obj corev1.Pod) *batchv1.Job {
	podspec := obj.Spec
	podspec = d.convertVolumesKindsByPolicy(podspec)
	podspec.RestartPolicy = corev1.RestartPolicyOnFailure
	pod := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       jobKind,
			APIVersion: batchv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: obj.ObjectMeta,
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: obj.ObjectMeta,
				Spec:       podspec,
			},
		},
	}
	return &pod
}

func (d *Deployment) toPod(meta metav1.ObjectMeta, podspec corev1.PodSpec, restartPolicy corev1.RestartPolicy) *corev1.Pod {
	podspec = d.convertVolumesKindsByPolicy(podspec)
	podspec.RestartPolicy = restartPolicy
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       podKind,
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec:       podspec,
	}
	return &pod
}

//Volumes and volume mounts of all containers are translated as follows:
//1. Each container's volume mount list and corresponding volumes are translated
//2. Unreferenced volumes are discarded
func (d *Deployment) convertVolumesKindsByPolicy(podspec corev1.PodSpec) corev1.PodSpec {
	if podspec.Volumes == nil || len(podspec.Volumes) == 0 {
		return podspec
	}

	// Remove unused volumes
	for _, container := range podspec.Containers {
		volMounts := []v1.VolumeMount{}
		for _, vm := range container.VolumeMounts {
			volume := getMatchingVolume(vm, podspec.Volumes)
			if volume == (corev1.Volume{}) {
				log.Warnf("Couldn't find a corresponding volume for volume mount %s", vm.Name)
				continue
			}
			volMounts = append(volMounts, vm)
		}
		container.VolumeMounts = volMounts
	}

	volumes := []v1.Volume{}
	for _, v := range podspec.Volumes {
		volumes = append(volumes, convertVolumeBySupportedKind(v, d.ClusterSpec))
	}
	podspec.Volumes = volumes

	return podspec
}

func getMatchingVolume(vm corev1.VolumeMount, vList []corev1.Volume) corev1.Volume {
	for _, v := range vList {
		if strings.Compare(v.Name, vm.Name) == 0 {
			return v
		}
	}
	return corev1.Volume{}
}
