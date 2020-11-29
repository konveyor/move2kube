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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	common "github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
)

const (
	networkPolicyKind = "NetworkPolicy"
	networkSelector   = types.GroupName + "/network"
)

// NetworkPolicy handles NetworkPolicy objects
type NetworkPolicy struct {
	Cluster collecttypes.ClusterMetadataSpec
}

// GetSupportedKinds returns all kinds supported by the class
func (d *NetworkPolicy) GetSupportedKinds() []string {
	return []string{networkPolicyKind}
}

// CreateNewResources converts ir to runtime objects
func (d *NetworkPolicy) CreateNewResources(ir irtypes.IR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	if !common.IsStringPresent(supportedKinds, networkPolicyKind) {
		log.Errorf("Could not find a valid resource type in cluster to create a NetworkPolicy")
		return nil
	}

	for _, service := range ir.Services {
		// Create services depending on whether the service needs to be externally exposed
		for _, net := range service.Networks {
			log.Debugf("Network %s is detected at Source, shall be converted to equivalent NetworkPolicy at Destination", net)
			obj, err := d.createNetworkPolicy(net)
			if err != nil {
				log.Warnf("Unable to create Network Policy for network %v for service %s : %s", net, service.Name, err)
				continue
			}
			objs = append(objs, obj)
		}
	}
	return objs
}

// ConvertToClusterSupportedKinds converts kinds to cluster supported kinds
func (d *NetworkPolicy) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.IR) ([]runtime.Object, bool) {
	if common.IsStringPresent(supportedKinds, networkPolicyKind) {
		if _, ok := obj.(*networkingv1.NetworkPolicy); ok {
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}

// CreateNetworkPolicy initializes Network policy
func (d *NetworkPolicy) createNetworkPolicy(networkName string) (*networkingv1.NetworkPolicy, error) {

	np := &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       networkPolicyKind,
			APIVersion: networkingv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: networkName,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{networkSelector + "/" + networkName: common.AnnotationLabelValue},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: getNetworkPolicyLabels([]string{networkName}),
					},
				}},
			}},
		},
	}

	return np, nil
}

func getNetworkPolicyLabels(networks []string) map[string]string {
	networklabels := map[string]string{}
	for _, network := range networks {
		networklabels[networkSelector+"/"+network] = common.AnnotationLabelValue
	}
	return networklabels
}
