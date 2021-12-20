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

package k8sschema

import (
	"k8s.io/apimachinery/pkg/runtime"

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
	flowcontrol "k8s.io/kubernetes/pkg/apis/flowcontrol"
	networking "k8s.io/kubernetes/pkg/apis/networking"
	node "k8s.io/kubernetes/pkg/apis/node"
	policy "k8s.io/kubernetes/pkg/apis/policy"
	rbac "k8s.io/kubernetes/pkg/apis/rbac"
	scheduling "k8s.io/kubernetes/pkg/apis/scheduling"
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
	storageinstall "k8s.io/kubernetes/pkg/apis/storage/install"

	okdapi "github.com/openshift/api"
	tektonscheme "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	k8sapischeme "k8s.io/client-go/kubernetes/scheme"
)

// K8sResourceT represents type used to process K8s objects. Not using type alias breaks parameterizer currently.
type K8sResourceT = map[string]interface{}

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
	must(flowcontrol.AddToScheme(liasonscheme))
	must(networking.AddToScheme(liasonscheme))
	must(node.AddToScheme(liasonscheme))
	must(policy.AddToScheme(liasonscheme))
	must(rbac.AddToScheme(liasonscheme))
	must(scheduling.AddToScheme(liasonscheme))
	must(storage.AddToScheme(liasonscheme))
}

// GetSchema returns the scheme
func GetSchema() *runtime.Scheme {
	return scheme
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
