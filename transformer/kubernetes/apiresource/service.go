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
	"fmt"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer/kubernetes/k8sschema"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	okdroutev1 "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	core "k8s.io/kubernetes/pkg/apis/core"
	networking "k8s.io/kubernetes/pkg/apis/networking"
)

const (
	routeKind = "Route"
)

// Service handles all objects related to a service
type Service struct {
}

// getSupportedKinds returns supported kinds
func (d *Service) getSupportedKinds() []string {
	return []string{common.ServiceKind, common.IngressKind, routeKind}
}

// createNewResources converts IR to runtime objects
func (d *Service) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	ingressEnabled := false
	for _, service := range ir.Services {
		exposeobjectcreated := false
		if _, _, _, st := d.getExposeInfo(service); st != "" || service.OnlyIngress {
			// Create services depending on whether the service needs to be externally exposed
			if common.IsPresent(supportedKinds, routeKind) {
				//Create Route
				routeObjs := d.createRoutes(service, ir, targetCluster)
				for _, routeObj := range routeObjs {
					objs = append(objs, routeObj)
				}
				exposeobjectcreated = true
			} else if common.IsPresent(supportedKinds, common.IngressKind) {
				//Create Ingress
				// obj := d.createIngress(service)
				// objs = append(objs, obj)
				exposeobjectcreated = true
				ingressEnabled = true
			}
		}
		if service.OnlyIngress {
			if !exposeobjectcreated {
				logrus.Errorf("Failed to create the ingress for service %q . Probable cause: The cluster doesn't support ingress resources.", service.Name)
			}
			continue
		}
		obj := d.createService(service)
		objs = append(objs, obj)
	}

	// Create one ingress for all services
	if ingressEnabled {
		obj := d.createIngress(ir, targetCluster)
		if obj != nil {
			objs = append(objs, obj)
		}
	}

	return objs
}

// convertToClusterSupportedKinds converts kinds to cluster supported kinds
func (d *Service) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, ir irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	lobj, _ := k8sschema.ConvertToLiasonScheme(obj)
	if common.IsPresent(supportedKinds, routeKind) {
		if _, ok := obj.(*okdroutev1.Route); ok {
			return []runtime.Object{obj}, true
		}
		if ingress, ok := lobj.(*networking.Ingress); ok {
			return d.ingressToRoute(*ingress), true
		}
		if _, ok := lobj.(*core.Service); ok {
			return []runtime.Object{obj}, true
		}
	} else if common.IsPresent(supportedKinds, common.IngressKind) {
		if route, ok := obj.(*okdroutev1.Route); ok {
			return d.routeToIngress(*route, ir, targetCluster.Spec), true
		}
		if _, ok := lobj.(*networking.Ingress); ok {
			return []runtime.Object{obj}, true
		}
		if _, ok := lobj.(*core.Service); ok {
			return []runtime.Object{obj}, true
		}
	} else {
		if route, ok := obj.(*okdroutev1.Route); ok {
			return d.routeToService(*route), true
		}
		if ingress, ok := lobj.(*networking.Ingress); ok {
			return d.ingressToService(*ingress), true
		}
		if _, ok := lobj.(*core.Service); ok {
			//TODO: Check if the destination cluster supports loadbalancer or nodeport and change between them.
			return []runtime.Object{obj}, true
		}
	}
	if common.IsPresent(d.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}

func (d *Service) ingressToRoute(ingress networking.Ingress) []runtime.Object {
	weight := int32(1)                                    //Hard-coded to 1 to avoid Helm v3 errors
	ingressArray := []okdroutev1.RouteIngress{{Host: ""}} //Hard-coded to empty string to avoid Helm v3 errors

	objs := []runtime.Object{}

	for _, ingressspec := range ingress.Spec.Rules {
		for _, path := range ingressspec.IngressRuleValue.HTTP.Paths {
			targetPort := intstr.IntOrString{Type: intstr.String, StrVal: path.Backend.Service.Port.Name}
			if path.Backend.Service.Port.Name == "" {
				targetPort.Type = intstr.Int
				targetPort.IntVal = path.Backend.Service.Port.Number
			}
			route := &okdroutev1.Route{
				TypeMeta: metav1.TypeMeta{
					Kind:       routeKind,
					APIVersion: okdroutev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: ingress.ObjectMeta,
				Spec: okdroutev1.RouteSpec{
					Host: ingressspec.Host,
					Path: path.Path,
					To: okdroutev1.RouteTargetReference{
						Kind:   common.ServiceKind,
						Name:   path.Backend.Service.Name,
						Weight: &weight,
					},
					Port: &okdroutev1.RoutePort{TargetPort: targetPort},
				},
				Status: okdroutev1.RouteStatus{Ingress: ingressArray},
			}
			objs = append(objs, route)
		}
	}

	return objs
}

func (d *Service) routeToIngress(route okdroutev1.Route, ir irtypes.EnhancedIR, targetClusterSpec collecttypes.ClusterMetadataSpec) []runtime.Object {
	targetPort := networking.ServiceBackendPort{}
	if route.Spec.Port != nil {
		if route.Spec.Port.TargetPort.Type == intstr.String {
			targetPort.Name = route.Spec.Port.TargetPort.StrVal
		} else {
			targetPort.Number = route.Spec.Port.TargetPort.IntVal
		}
	} else {
		targetName := route.Spec.To.Name
		s, ok := ir.Services[targetName]
		if !ok {
			for _, s1 := range ir.Services {
				if s1.BackendServiceName == targetName {
					s = s1
					ok = true
					break
				}
			}
		}
		if !ok || len(s.ServiceToPodPortForwardings) == 0 {
			logrus.Errorf("failed to find the service the route is pointing to. Exposing a default port (8080) in the Ingress")
			targetPort.Number = 8080
		} else {
			portForwarding := s.ServiceToPodPortForwardings[0]
			if portForwarding.ServicePort.Name != "" {
				targetPort.Name = portForwarding.ServicePort.Name
			} else {
				targetPort.Number = portForwarding.ServicePort.Number
			}
		}
	}

	ingress := networking.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       common.IngressKind,
			APIVersion: networking.SchemeGroupVersion.String(),
		},
		ObjectMeta: route.ObjectMeta,
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: route.Spec.Path,
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: route.Spec.To.Name,
											Port: targetPort,
										},
									},
								},
							},
						},
					},
					Host: route.Spec.Host,
				},
			},
		},
	}

	return []runtime.Object{&ingress}
}

func (d *Service) routeToService(route okdroutev1.Route) []runtime.Object {
	// TODO: Think through how will the clusterip service that was originally there will behave when merged with this service?
	svc := &core.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       common.ServiceKind,
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: route.ObjectMeta,
		Spec: core.ServiceSpec{
			// TODO: we are expecting the pod label selector to be merged in from other existing services
			// TODO: How to choose between nodeport and loadbalancer?
			Type: core.ServiceTypeNodePort,
			Ports: []core.ServicePort{
				{
					Name: route.Spec.Port.TargetPort.StrVal,
					Port: route.Spec.Port.TargetPort.IntVal,
					// TODO: what about targetPort?
				},
			},
		},
	}
	svc.Name = route.Spec.To.Name

	return []runtime.Object{svc}
}

func (d *Service) ingressToService(ingress networking.Ingress) []runtime.Object {
	objs := []runtime.Object{}
	for _, ingressspec := range ingress.Spec.Rules {
		for _, path := range ingressspec.IngressRuleValue.HTTP.Paths {
			svc := &core.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       common.ServiceKind,
					APIVersion: core.SchemeGroupVersion.String(),
				},
				ObjectMeta: ingress.ObjectMeta,
				Spec: core.ServiceSpec{
					// TODO: we are expecting the pod label selector to be merged in from other existing services
					Type: core.ServiceTypeNodePort,
					Ports: []core.ServicePort{
						{
							//TODO: Check if this is the right mapping
							Name: path.Backend.Service.Port.Name,
							Port: path.Backend.Service.Port.Number,
							// TODO: what about targetPort?
						},
					},
				},
			}
			svc.Name = path.Backend.Service.Name
			objs = append(objs, svc)
		}
	}
	return objs
}

func (d *Service) createRoutes(service irtypes.Service, ir irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) [](*okdroutev1.Route) {
	routes := [](*okdroutev1.Route){}
	servicePorts, hostPrefixes, relPaths, _ := d.getExposeInfo(service)
	for i, servicePort := range servicePorts {
		if relPaths[i] == "" {
			continue
		}
		route := d.createRoute(ir.Name, service, servicePort, hostPrefixes[i], relPaths[i], ir, targetCluster)
		routes = append(routes, route)
	}
	return routes
}

// TODO: Remove these two sections after helm v3 issue is fixed
// [https://github.com/openshift/origin/issues/24060]
// [https://bugzilla.redhat.com/show_bug.cgi?id=1773682]
// Can't use https because of this https://github.com/openshift/origin/issues/2162
// When service has multiple ports,the route needs a port name. Port number doesn't seem to work.
func (d *Service) createRoute(irName string, service irtypes.Service, port core.ServicePort, hostprefix, path string, ir irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) *okdroutev1.Route {
	weight := int32(1)                                    //Hard-coded to 1 to avoid Helm v3 errors
	ingressArray := []okdroutev1.RouteIngress{{Host: ""}} //Hard-coded to empty string to avoid Helm v3 errors

	host := targetCluster.Spec.Host
	if host == "" {
		host = commonqa.IngressHost(d.getHostName(irName), targetCluster.Labels[collecttypes.ClusterQaLabelKey])
	}
	ph := host
	if hostprefix != "" {
		ph = hostprefix + "." + ph
	}
	route := &okdroutev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       routeKind,
			APIVersion: okdroutev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   service.Name,
			Labels: getServiceLabels(service.Name),
		},
		Spec: okdroutev1.RouteSpec{
			Host: ph,
			Path: path,
			To: okdroutev1.RouteTargetReference{
				Kind:   common.ServiceKind,
				Name:   service.Name,
				Weight: &weight,
			},
			Port: &okdroutev1.RoutePort{TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: port.Name}},
		},
		Status: okdroutev1.RouteStatus{
			Ingress: ingressArray,
		},
	}
	return route
}

// createIngress creates a single ingress for all services
func (d *Service) createIngress(ir irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) *networking.Ingress {
	pathType := networking.PathTypePrefix

	hostHTTPIngressPaths := map[string][]networking.HTTPIngressPath{} //[hostprefix]
	for _, service := range ir.Services {
		backendServiceName := service.BackendServiceName
		if service.BackendServiceName == "" {
			backendServiceName = service.Name
		}
		servicePorts, hostPrefixes, relPaths, _ := d.getExposeInfo(service)
		for i, servicePort := range servicePorts {
			if relPaths[i] == "" {
				continue
			}
			backendPort := networking.ServiceBackendPort{Name: servicePort.Name}
			if servicePort.Name == "" {
				backendPort = networking.ServiceBackendPort{Number: servicePort.Port}
			}

			httpIngressPath := networking.HTTPIngressPath{
				Path:     relPaths[i],
				PathType: &pathType,
				Backend: networking.IngressBackend{
					Service: &networking.IngressServiceBackend{
						Name: backendServiceName,
						Port: backendPort,
					},
				},
			}
			hostHTTPIngressPaths[hostPrefixes[i]] = append(hostHTTPIngressPaths[hostPrefixes[i]], httpIngressPath)
		}
	}
	if len(hostHTTPIngressPaths) == 0 {
		return nil
	}
	qaLabel := collecttypes.DefaultClusterSpecificQaLabel
	if _, ok := targetCluster.Labels[collecttypes.ClusterQaLabelKey]; ok {
		qaLabel = targetCluster.Labels[collecttypes.ClusterQaLabelKey]
	}
	// QALabel prefix for cluster
	qaId := common.JoinQASubKeys(common.ConfigTargetKey, `"`+qaLabel+`"`)
	// Set the default ingressClass value
	quesKeyClass := common.JoinQASubKeys(qaId, common.ConfigIngressClassNameKeySuffix)
	descClass := "Provide the Ingress class name for ingress"
	ingressClassName := qaengine.FetchStringAnswer(quesKeyClass, descClass, []string{"Leave empty to use the cluster default"}, "", nil)

	// Configure the rule with the above fan-out paths
	rules := []networking.IngressRule{}
	host := targetCluster.Spec.Host
	secretName := ""
	defaultSecretName := ""
	if host == "" {
		host = commonqa.IngressHost(d.getHostName(ir.Name), qaLabel)
	}
	quesKeyTLS := common.JoinQASubKeys(qaId, common.ConfigIngressTLSKeySuffix)
	descTLS := "Provide the TLS secret for ingress"
	secretName = qaengine.FetchStringAnswer(quesKeyTLS, descTLS, []string{"Leave empty to use http"}, defaultSecretName, nil)
	for hostprefix, httpIngressPaths := range hostHTTPIngressPaths {
		ph := host
		if hostprefix != "" {
			ph = hostprefix + "." + ph
		}
		rules = append(rules, networking.IngressRule{
			Host: ph,
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: httpIngressPaths,
				},
			},
		})
	}

	tls := []networking.IngressTLS{}
	if secretName != "" {
		tls = []networking.IngressTLS{{Hosts: []string{host},
			SecretName: secretName,
		}}
	}

	ingressName := ir.Name
	ingress := networking.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       common.IngressKind,
			APIVersion: networking.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   ingressName,
			Labels: getServiceLabels(ingressName),
		},
		Spec: networking.IngressSpec{
			Rules: rules,
			TLS:   tls,
		},
	}
	if ingressClassName != "" {
		ingress.Spec.IngressClassName = &ingressClassName
	}

	return &ingress
}

// createService creates a service
func (d *Service) createService(service irtypes.Service) *core.Service {
	ports, _, _, serviceType := d.getExposeInfo(service)
	svc := &core.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       common.ServiceKind,
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        service.Name,
			Labels:      getServiceLabels(service.Name),
			Annotations: getAnnotations(service),
		},
		Spec: core.ServiceSpec{
			Type:     serviceType,
			Selector: getServiceLabels(service.Name),
			Ports:    ports,
		},
	}
	if len(ports) == 0 || service.StatefulSet {
		svc.Spec.ClusterIP = "None"
	}
	return svc
}

// GetServicePorts configure the container service ports.
func (d *Service) getExposeInfo(service irtypes.Service) (servicePorts []core.ServicePort, hostPrefixes []string, relPaths []string, serviceType core.ServiceType) {
	servicePorts = []core.ServicePort{}
	relPaths = []string{}
	hostPrefixes = []string{}
	serviceType = core.ServiceTypeClusterIP
	for _, forwarding := range service.ServiceToPodPortForwardings {
		servicePortName := forwarding.ServicePort.Name
		if servicePortName == "" {
			servicePortName = fmt.Sprintf("port-%d", forwarding.ServicePort.Number)
		}
		targetPort := intstr.IntOrString{Type: intstr.String, StrVal: forwarding.PodPort.Name}
		if forwarding.PodPort.Name == "" {
			targetPort.Type = intstr.Int
			targetPort.IntVal = forwarding.PodPort.Number
		}
		servicePort := core.ServicePort{
			Name:       servicePortName,
			Port:       forwarding.ServicePort.Number,
			TargetPort: targetPort,
		}
		switch forwarding.ServiceType {
		case core.ServiceTypeLoadBalancer:
			serviceType = forwarding.ServiceType
		case core.ServiceTypeNodePort:
			if serviceType != core.ServiceTypeLoadBalancer {
				serviceType = forwarding.ServiceType
			}
		case core.ServiceTypeClusterIP:
			if serviceType != core.ServiceTypeLoadBalancer && serviceType != core.ServiceTypeNodePort {
				serviceType = forwarding.ServiceType
			}
		case "":
			continue
		}
		hostPrefix := ""
		relPath := forwarding.ServiceRelPath
		if relPath != "" && !strings.HasPrefix(relPath, `/`) {
			parts := []string{relPath}
			if strings.Contains(relPath, `/`) {
				parts = strings.SplitN(relPath, `/`, 2)
			}
			relPath = `/`
			if len(parts) > 1 {
				relPath += parts[1]
			}
			hostPrefix = parts[0]
		}
		relPaths = append(relPaths, relPath)
		hostPrefixes = append(hostPrefixes, hostPrefix)
		servicePorts = append(servicePorts, servicePort)
	}
	return servicePorts, hostPrefixes, relPaths, serviceType
}

func (d *Service) getHostName(irName string) string {
	return irName + ".com"
}
