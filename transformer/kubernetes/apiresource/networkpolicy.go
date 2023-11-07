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
	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/types"
	collecttypes "github.com/konveyor/move2kube-wasm/types/collection"
	irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	networking "k8s.io/kubernetes/pkg/apis/networking"
)

const (
	networkPolicyKind = "NetworkPolicy"
	networkSelector   = types.GroupName + "/network"
)

// NetworkPolicy handles NetworkPolicy objects
type NetworkPolicy struct {
}

// getSupportedKinds returns all kinds supported by the class
func (d *NetworkPolicy) getSupportedKinds() []string {
	return []string{networkPolicyKind}
}

// createNewResources converts ir to runtime objects
func (d *NetworkPolicy) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	if !common.IsPresent(supportedKinds, networkPolicyKind) {
		logrus.Errorf("Could not find a valid resource type in cluster to create a NetworkPolicy")
		return nil
	}

	for _, service := range ir.Services {
		// Create services depending on whether the service needs to be externally exposed
		for _, net := range service.Networks {
			logrus.Debugf("Network %s is detected at Source, shall be converted to equivalent NetworkPolicy at Destination", net)
			obj, err := d.createNetworkPolicy(net)
			if err != nil {
				logrus.Warnf("Unable to create Network Policy for network %v for service %s : %s", net, service.Name, err)
				continue
			}
			objs = append(objs, obj)
		}
	}
	return objs
}

// convertToClusterSupportedKinds converts kinds to cluster supported kinds
func (d *NetworkPolicy) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if common.IsPresent(d.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}

// CreateNetworkPolicy initializes Network policy
func (d *NetworkPolicy) createNetworkPolicy(networkName string) (*networking.NetworkPolicy, error) {

	np := &networking.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       networkPolicyKind,
			APIVersion: networking.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: networkName,
		},
		Spec: networking.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{networkSelector + "-" + networkName: common.AnnotationLabelValue},
			},
			Ingress: []networking.NetworkPolicyIngressRule{{
				From: []networking.NetworkPolicyPeer{{
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
		networklabels[networkSelector+"-"+network] = common.AnnotationLabelValue
	}
	return networklabels
}
