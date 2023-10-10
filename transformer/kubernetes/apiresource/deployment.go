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
	"strings"

	argorollouts "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer/kubernetes/k8sschema"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	okdappsv1 "github.com/openshift/api/apps/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	apps "k8s.io/kubernetes/pkg/apis/apps"
	batch "k8s.io/kubernetes/pkg/apis/batch"
	core "k8s.io/kubernetes/pkg/apis/core"
	corev1conversions "k8s.io/kubernetes/pkg/apis/core/v1"
)

type RolloutType string

const (
	BlueGreenRollout RolloutType = "BlueGreen"
	CanaryRollout    RolloutType = "Canary"
)

//TODO: Add support for replicaset, cronjob and statefulset

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
	// statefulSetKind defines StatefulSet Kind
	statefulSetKind string = "StatefulSet"
	rolloutKind     string = "Rollout"
)

// Deployment handles all objects like a Deployment
type Deployment struct {
}

// getSupportedKinds returns kinds supported by the deployment
func (d *Deployment) getSupportedKinds() []string {
	return []string{podKind, jobKind, common.DeploymentKind, deploymentConfigKind, replicationControllerKind, daemonSetKind, statefulSetKind, rolloutKind}
}

// createNewResources converts ir to runtime object
func (d *Deployment) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	for _, service := range ir.Services {
		var obj runtime.Object
		if service.Daemon {
			if !common.IsPresent(supportedKinds, daemonSetKind) {
				logrus.Errorf("Creating Daemonset even though not supported by target cluster.")
			}
			obj = d.createDaemonSet(service, targetCluster.Spec)
		} else if service.DeploymentType == irtypes.StatefulSet {
			if !common.IsPresent(supportedKinds, statefulSetKind) {
				logrus.Errorf("Creating Statefulset even though not supported by target cluster.")
			}
			obj = d.createStatefulSet(service, targetCluster.Spec)
		} else if service.DeploymentType == irtypes.ArgoRollout {
			var err error
			obj, err = d.createArgorollout(service, targetCluster.Spec)
			if err != nil {
				logrus.Errorf("Error creating Argo Rollout: %v", err)
			}
		} else if service.RestartPolicy == core.RestartPolicyNever || service.RestartPolicy == core.RestartPolicyOnFailure {
			if common.IsPresent(supportedKinds, jobKind) {
				obj = d.createJob(service, targetCluster.Spec)
			} else {
				logrus.Errorf("Could not find a valid resource type in cluster to create a job/pod.")
			}
			pod := d.createPod(service, targetCluster.Spec)
			pod.Spec.RestartPolicy = core.RestartPolicyOnFailure
			obj = pod
		} else if common.IsPresent(supportedKinds, common.DeploymentKind) {
			obj = d.createDeployment(service, targetCluster.Spec)
		} else if common.IsPresent(supportedKinds, deploymentConfigKind) {
			obj = d.createDeploymentConfig(service, targetCluster.Spec)
		} else if common.IsPresent(supportedKinds, replicationControllerKind) {
			obj = d.createReplicationController(service, targetCluster.Spec)
		} else if common.IsPresent(supportedKinds, podKind) {
			obj = d.createPod(service, targetCluster.Spec)
		} else {
			logrus.Errorf("Could not find a valid resource type in cluster to create a deployment. Creating Deployment anyhow")
			obj = d.createDeployment(service, targetCluster.Spec)
		}
		if obj != nil {
			objs = append(objs, obj)
		}
	}
	return objs
}

// convertToClusterSupportedKinds converts objects to kind supported by the cluster
func (d *Deployment) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, ir irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	lobj, _ := k8sschema.ConvertToLiasonScheme(obj)
	if d1, ok := lobj.(*apps.DaemonSet); ok {
		return []runtime.Object{d1}, true
	}
	if d1, ok := lobj.(*core.Pod); ok && (d1.Spec.RestartPolicy == core.RestartPolicyOnFailure || d1.Spec.RestartPolicy == core.RestartPolicyNever) {
		if common.IsPresent(supportedKinds, jobKind) {
			return []runtime.Object{d.podToJob(*d1, targetCluster.Spec)}, true
		}
		return []runtime.Object{obj}, true
	} else if d1, ok := lobj.(*batch.Job); ok && !common.IsPresent(supportedKinds, jobKind) {
		if !common.IsPresent(supportedKinds, jobKind) && common.IsPresent(supportedKinds, podKind) {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, core.RestartPolicyOnFailure, targetCluster.Spec)}, true
		}
		logrus.Warnf("Both Job and Pod not supported. No other valid way to transform this object. : %+v", obj)
		return []runtime.Object{obj}, true
	}
	if common.IsPresent(supportedKinds, common.DeploymentKind) {
		if d1, ok := obj.(*okdappsv1.DeploymentConfig); ok {
			return []runtime.Object{d.toDeployment(d1.ObjectMeta, k8sschema.ConvertToPodSpec(&d1.Spec.Template.Spec), d1.Spec.Replicas, targetCluster.Spec)}, true
		} else if d1, ok := lobj.(*core.ReplicationController); ok {
			return []runtime.Object{d.toDeployment(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas, targetCluster.Spec)}, true
		} else if d1, ok := lobj.(*core.Pod); ok {
			var replicas int32 = 2
			return []runtime.Object{d.toDeployment(d1.ObjectMeta, d1.Spec, replicas, targetCluster.Spec)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsPresent(supportedKinds, deploymentConfigKind) {
		if d1, ok := lobj.(*apps.Deployment); ok {
			return []runtime.Object{d.toDeploymentConfig(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas, targetCluster.Spec)}, true
		} else if d1, ok := lobj.(*core.ReplicationController); ok {
			return []runtime.Object{d.toDeploymentConfig(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas, targetCluster.Spec)}, true
		} else if d1, ok := lobj.(*core.Pod); ok {
			var replicas int32 = 2
			return []runtime.Object{d.toDeploymentConfig(d1.ObjectMeta, d1.Spec, replicas, targetCluster.Spec)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsPresent(supportedKinds, replicationControllerKind) {
		if d1, ok := obj.(*okdappsv1.DeploymentConfig); ok {
			return []runtime.Object{d.toReplicationController(d1.ObjectMeta, k8sschema.ConvertToPodSpec(&d1.Spec.Template.Spec), d1.Spec.Replicas, targetCluster.Spec)}, true
		} else if d1, ok := lobj.(*apps.Deployment); ok {
			return []runtime.Object{d.toReplicationController(d1.ObjectMeta, d1.Spec.Template.Spec, d1.Spec.Replicas, targetCluster.Spec)}, true
		} else if d1, ok := lobj.(*core.Pod); ok {
			var replicas int32 = 2
			return []runtime.Object{d.toReplicationController(d1.ObjectMeta, d1.Spec, replicas, targetCluster.Spec)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsPresent(supportedKinds, podKind) {
		if d1, ok := obj.(*okdappsv1.DeploymentConfig); ok {
			return []runtime.Object{d.toPod(d1.ObjectMeta, k8sschema.ConvertToPodSpec(&d1.Spec.Template.Spec), core.RestartPolicyAlways, targetCluster.Spec)}, true
		} else if d1, ok := lobj.(*apps.Deployment); ok {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, core.RestartPolicyAlways, targetCluster.Spec)}, true
		} else if d1, ok := lobj.(*core.ReplicationController); ok {
			return []runtime.Object{d.toPod(d1.ObjectMeta, d1.Spec.Template.Spec, core.RestartPolicyAlways, targetCluster.Spec)}, true
		}
		return []runtime.Object{obj}, true
	}
	if common.IsPresent(d.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}

// Create section

func (d *Deployment) createDeployment(service irtypes.Service, cluster collecttypes.ClusterMetadataSpec) *apps.Deployment {
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	podSpec := service.PodSpec
	podSpec = irtypes.PodSpec(d.convertVolumesKindsByPolicy(core.PodSpec(podSpec), cluster))
	podSpec.RestartPolicy = core.RestartPolicyAlways
	logrus.Debugf("Created deployment for %s", service.Name)
	return d.toDeployment(meta, core.PodSpec(podSpec), int32(service.Replicas), cluster)
}

func (d *Deployment) createDeploymentConfig(service irtypes.Service, cluster collecttypes.ClusterMetadataSpec) *okdappsv1.DeploymentConfig {
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	podSpec := service.PodSpec
	podSpec = irtypes.PodSpec(d.convertVolumesKindsByPolicy(core.PodSpec(podSpec), cluster))
	podSpec.RestartPolicy = core.RestartPolicyAlways
	logrus.Debugf("Created DeploymentConfig for %s", service.Name)
	return d.toDeploymentConfig(meta, core.PodSpec(podSpec), int32(service.Replicas), cluster)
}

// createReplicationController initializes Kubernetes ReplicationController object
func (d *Deployment) createReplicationController(service irtypes.Service, cluster collecttypes.ClusterMetadataSpec) *core.ReplicationController {
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	podSpec := service.PodSpec
	podSpec = irtypes.PodSpec(d.convertVolumesKindsByPolicy(core.PodSpec(podSpec), cluster))
	podSpec.RestartPolicy = core.RestartPolicyAlways
	logrus.Debugf("Created DeploymentConfig for %s", service.Name)
	return d.toReplicationController(meta, core.PodSpec(podSpec), int32(service.Replicas), cluster)
}

func (d *Deployment) createPod(service irtypes.Service, cluster collecttypes.ClusterMetadataSpec) *core.Pod {
	podSpec := service.PodSpec
	podSpec = irtypes.PodSpec(d.convertVolumesKindsByPolicy(core.PodSpec(podSpec), cluster))
	podSpec.RestartPolicy = core.RestartPolicyAlways
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	return d.toPod(meta, core.PodSpec(podSpec), podSpec.RestartPolicy, cluster)
}

func (d *Deployment) createDaemonSet(service irtypes.Service, cluster collecttypes.ClusterMetadataSpec) *apps.DaemonSet {
	podSpec := service.PodSpec
	podSpec = irtypes.PodSpec(d.convertVolumesKindsByPolicy(core.PodSpec(podSpec), cluster))
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
				Spec:       core.PodSpec(podSpec),
			},
		},
	}
	return &pod
}

func (d *Deployment) createJob(service irtypes.Service, cluster collecttypes.ClusterMetadataSpec) *batch.Job {
	podspec := service.PodSpec
	podspec = irtypes.PodSpec(d.convertVolumesKindsByPolicy(core.PodSpec(podspec), cluster))
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
				Spec:       core.PodSpec(podspec),
			},
		},
	}
	return &pod
}

func (d *Deployment) createStatefulSet(service irtypes.Service, cluster collecttypes.ClusterMetadataSpec) *apps.StatefulSet {
	podSpec := service.PodSpec
	podSpec = irtypes.PodSpec(d.convertVolumesKindsByPolicy(core.PodSpec(podSpec), cluster))
	podSpec.RestartPolicy = core.RestartPolicyAlways
	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	statefulset := apps.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       statefulSetKind,
			APIVersion: metav1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta,
		Spec: apps.StatefulSetSpec{
			Replicas: int32(service.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: getServiceLabels(meta.Name),
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       core.PodSpec(podSpec),
			},
			ServiceName: service.Name,
		},
	}
	return &statefulset
}

func (d *Deployment) createArgorollout(service irtypes.Service, cluster collecttypes.ClusterMetadataSpec) (*argorollouts.Rollout, error) {
	podSpec := service.PodSpec
	podSpec = irtypes.PodSpec(d.convertVolumesKindsByPolicy(core.PodSpec(podSpec), cluster))

	meta := metav1.ObjectMeta{
		Name:        service.Name,
		Labels:      getPodLabels(service.Name, service.Networks),
		Annotations: getAnnotations(service),
	}
	replicas := int32(service.Replicas)
	var v1spec corev1.PodSpec
	var corespec core.PodSpec = core.PodSpec(podSpec)
	err := corev1conversions.Convert_core_PodSpec_To_v1_PodSpec(&corespec, &v1spec, nil)
	if err != nil {
		return nil, err
	}

	// prompt type of rollout
	qaKey := common.JoinQASubKeys(common.ConfigServicesKey, common.ConfigDeploymentTypeKey, common.ConfigArgoRolloutTypeKey)
	desc := "Which type of Argo rollout should be generated?"
	def := string(BlueGreenRollout)
	options := []string{string(BlueGreenRollout), string(CanaryRollout)}
	rolloutType := qaengine.FetchSelectAnswer(qaKey, desc, nil, def, options, nil)

	var rolloutStrategy argorollouts.RolloutStrategy
	if rolloutType == string(BlueGreenRollout) {
		rolloutStrategy = argorollouts.RolloutStrategy{
			BlueGreen: &argorollouts.BlueGreenStrategy{
				ActiveService:        service.Name,
				PreviewService:       service.Name + "-preview",
				AutoPromotionEnabled: func(b bool) *bool { return &b }(false),
			},
		}
	} else {
		rolloutStrategy = argorollouts.RolloutStrategy{
			Canary: &argorollouts.CanaryStrategy{
				StableService: service.Name,
				CanaryService: service.Name + "-preview",
				MaxSurge:      &intstr.IntOrString{StrVal: "25%"},
			},
		}
	}

	rollout := argorollouts.Rollout{
		TypeMeta: metav1.TypeMeta{
			Kind:       rolloutKind,
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: meta,
		Spec: argorollouts.RolloutSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getServiceLabels(meta.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: meta,
				Spec:       v1spec,
			},
			Strategy: rolloutStrategy,
		},
	}

	return &rollout, nil
}

// Conversions section

func (d *Deployment) toDeploymentConfig(meta metav1.ObjectMeta, podspec core.PodSpec, replicas int32, cluster collecttypes.ClusterMetadataSpec) *okdappsv1.DeploymentConfig {
	podspec = d.convertVolumesKindsByPolicy(podspec, cluster)
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

func (d *Deployment) toDeployment(meta metav1.ObjectMeta, podspec core.PodSpec, replicas int32, cluster collecttypes.ClusterMetadataSpec) *apps.Deployment {
	podspec = d.convertVolumesKindsByPolicy(podspec, cluster)
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
func (d *Deployment) toReplicationController(meta metav1.ObjectMeta, podspec core.PodSpec, replicas int32, cluster collecttypes.ClusterMetadataSpec) *core.ReplicationController {
	podspec = d.convertVolumesKindsByPolicy(podspec, cluster)
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

func (d *Deployment) podToJob(obj core.Pod, cluster collecttypes.ClusterMetadataSpec) *batch.Job {
	podspec := obj.Spec
	podspec = d.convertVolumesKindsByPolicy(podspec, cluster)
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

func (d *Deployment) toPod(meta metav1.ObjectMeta, podspec core.PodSpec, restartPolicy core.RestartPolicy, cluster collecttypes.ClusterMetadataSpec) *core.Pod {
	podspec = d.convertVolumesKindsByPolicy(podspec, cluster)
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

// Volumes and volume mounts of all containers are transformed as follows:
// 1. Each container's volume mount list and corresponding volumes are transformed
// 2. Unreferenced volumes are discarded
func (d *Deployment) convertVolumesKindsByPolicy(podspec core.PodSpec, cluster collecttypes.ClusterMetadataSpec) core.PodSpec {
	if podspec.Volumes == nil || len(podspec.Volumes) == 0 {
		return podspec
	}

	// Remove unused volumes
	for _, container := range podspec.Containers {
		volMounts := []core.VolumeMount{}
		for _, vm := range container.VolumeMounts {
			volume := getMatchingVolume(vm, podspec.Volumes)
			if volume == (core.Volume{}) {
				logrus.Warnf("Couldn't find a corresponding volume for volume mount %s", vm.Name)
				continue
			}
			volMounts = append(volMounts, vm)
		}
		container.VolumeMounts = volMounts
	}

	volumes := []core.Volume{}
	for _, v := range podspec.Volumes {
		volumes = append(volumes, convertVolumeBySupportedKind(v, cluster))
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
