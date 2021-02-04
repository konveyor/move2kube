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
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	core "k8s.io/kubernetes/pkg/apis/core"
	coreinstall "k8s.io/kubernetes/pkg/apis/core/install"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	coreinstall.Install(scheme)
}

func convertCoreObjectVersion(in interface{}, out interface{}) error {
	err := scheme.Convert(in, out, nil)
	if err != nil {
		log.Errorf("Unable to convert from %T to %T : %s", in, out, err)
	}
	return err
}

func convertToV1PodSpec(podSpec *core.PodSpec) corev1.PodSpec {
	vPodSpec := corev1.PodSpec{}
	err := convertCoreObjectVersion(podSpec, &vPodSpec)
	if err != nil {
		log.Errorf("Unable to convert PodSpec to versioned PodSpec : %s", err)
	}
	return vPodSpec
}

func convertToPodSpec(podspec *corev1.PodSpec) core.PodSpec {
	uvPodSpec := core.PodSpec{}
	err := convertCoreObjectVersion(podspec, &uvPodSpec)
	if err != nil {
		log.Errorf("Unable to convert versioned PodSpec to unversioned PodSpec : %s", err)
	}
	return uvPodSpec
}
