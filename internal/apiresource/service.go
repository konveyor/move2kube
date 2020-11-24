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

	okdroutev1 "github.com/openshift/api/route/v1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	common "github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
)

const (
	serviceKind = "Service"
	ingressKind = "Ingress"
	routeKind   = "Route"
)

// Service handles all objects related to a service
type Service struct {
	Cluster collecttypes.ClusterMetadataSpec
}

// GetSupportedKinds returns supported kinds
func (d *Service) GetSupportedKinds() []string {
	return []string{serviceKind, ingressKind, routeKind}
}

// CreateNewResources converts IR to runtime objects
func (d *Service) CreateNewResources(ir irtypes.IR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	ingressEnabled := false
	for _, service := range ir.Services {
		exposeobjectcreated := false
		if service.HasValidAnnotation(common.ExposeSelector) || service.OnlyIngress {
			// Create services depending on whether the service needs to be externally exposed
			if common.IsStringPresent(supportedKinds, routeKind) {
				//Create Route
				obj := d.createRoute(service)
				objs = append(objs, obj)
				exposeobjectcreated = true
			} else if common.IsStringPresent(supportedKinds, ingressKind) {
				//Create Ingress
				// obj := d.createIngress(service)
				// objs = append(objs, obj)
				exposeobjectcreated = true
				ingressEnabled = true
			}
		}
		if service.OnlyIngress {
			if !exposeobjectcreated {
				log.Errorf("Failed to create the ingress for service %q . Probable cause: The cluster doesn't support ingress resources.", service.Name)
			}
			continue
		}
		if !common.IsStringPresent(supportedKinds, serviceKind) {
			log.Errorf("Could not find a valid resource type in cluster to create a Service")
			continue
		}
		if exposeobjectcreated || !service.HasValidAnnotation(common.ExposeSelector) {
			//Create clusterip service
			obj := d.createService(service, v1.ServiceTypeClusterIP)
			objs = append(objs, obj)
		} else {
			//Create Nodeport service - TODO: Should it be load balancer or Nodeport? Should it be QA?
			obj := d.createService(service, v1.ServiceTypeNodePort)
			objs = append(objs, obj)
		}
	}

	// Create one ingress for all services
	if ingressEnabled {
		obj := d.createIngress(ir)
		objs = append(objs, obj)
	}

	return objs
}

// ConvertToClusterSupportedKinds converts kinds to cluster supported kinds
func (d *Service) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object) ([]runtime.Object, bool) {
	if common.IsStringPresent(supportedKinds, routeKind) {
		if _, ok := obj.(*okdroutev1.Route); ok {
			return []runtime.Object{obj}, true
		} else if i, ok := obj.(*networkingv1.Ingress); ok {
			return d.ingressToRoute(*i), true
		} else if s, ok := obj.(*v1.Service); ok {
			if s.Spec.Type == v1.ServiceTypeLoadBalancer || s.Spec.Type == v1.ServiceTypeNodePort {
				return d.serviceToRoute(*s), true
			}
			return []runtime.Object{obj}, true
		}
	} else if common.IsStringPresent(supportedKinds, ingressKind) {
		if r, ok := obj.(*okdroutev1.Route); ok {
			return d.routeToIngress(*r), true
		} else if _, ok := obj.(*networkingv1.Ingress); ok {
			return []runtime.Object{obj}, true
		} else if s, ok := obj.(*v1.Service); ok {
			if s.Spec.Type == v1.ServiceTypeLoadBalancer || s.Spec.Type == v1.ServiceTypeNodePort {
				return d.serviceToIngress(*s), true
			}
			return []runtime.Object{obj}, true
		}
	} else if common.IsStringPresent(supportedKinds, serviceKind) {
		if r, ok := obj.(*okdroutev1.Route); ok {
			return d.routeToService(*r), true
		} else if i, ok := obj.(*networkingv1.Ingress); ok {
			return d.ingressToService(*i), true
		} else if _, ok := obj.(*v1.Service); ok {
			//TODO: Check if the destination cluster supports loadbalancer or nodeport and change between them.
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}

func (d *Service) ingressToRoute(ingress networkingv1.Ingress) []runtime.Object {
	var weight int32 = 1                       //Hard-coded to 1 to avoid Helm v3 errors
	var ingressArray []okdroutev1.RouteIngress //Hard-coded to empty list to avoid Helm v3 errors
	ingressArray = append(ingressArray, okdroutev1.RouteIngress{Host: ""})

	objs := []runtime.Object{}

	for _, ingressspec := range ingress.Spec.Rules {
		for _, path := range ingressspec.IngressRuleValue.HTTP.Paths {
			targetPort := intstr.IntOrString{}
			if path.Backend.Service.Port.Name != "" {
				targetPort.Type = intstr.String
				targetPort.StrVal = path.Backend.Service.Port.Name
			} else {
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
					Port: &okdroutev1.RoutePort{
						TargetPort: targetPort,
					},
					To: okdroutev1.RouteTargetReference{
						Kind:   serviceKind,
						Name:   path.Backend.Service.Name,
						Weight: &weight,
					},
					Host: ingressspec.Host,
				},
				Status: okdroutev1.RouteStatus{
					Ingress: ingressArray,
				},
			}
			objs = append(objs, route)
		}
	}

	return objs
}

func (d *Service) serviceToRoute(service v1.Service) []runtime.Object {
	var weight int32 = 1                       //Hard-coded to 1 to avoid Helm v3 errors
	var ingressArray []okdroutev1.RouteIngress //Hard-coded to empty list to avoid Helm v3 errors
	ingressArray = append(ingressArray, okdroutev1.RouteIngress{Host: ""})

	objs := []runtime.Object{}
	for _, serviceport := range service.Spec.Ports {
		port := intstr.IntOrString{
			IntVal: serviceport.Port,
		}
		route := &okdroutev1.Route{
			TypeMeta: metav1.TypeMeta{
				Kind:       routeKind,
				APIVersion: okdroutev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: service.ObjectMeta,
			Spec: okdroutev1.RouteSpec{
				Port: &okdroutev1.RoutePort{
					TargetPort: port,
				},
				To: okdroutev1.RouteTargetReference{
					Kind:   serviceKind,
					Name:   service.Name,
					Weight: &weight,
				},
				Host: "",
			},
			Status: okdroutev1.RouteStatus{
				Ingress: ingressArray,
			},
		}
		objs = append(objs, route)
	}
	service.Spec.Type = v1.ServiceTypeClusterIP
	objs = append(objs, &service)

	return objs
}

func (d *Service) routeToIngress(route okdroutev1.Route) []runtime.Object {
	targetPort := networkingv1.ServiceBackendPort{}
	if route.Spec.Port.TargetPort.Type == intstr.String {
		targetPort.Name = route.Spec.Port.TargetPort.StrVal
	} else {
		targetPort.Number = route.Spec.Port.TargetPort.IntVal
	}

	ingress := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       ingressKind,
			APIVersion: networkingv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: route.ObjectMeta,
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
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

	if strings.HasPrefix(route.Spec.Host, "https") {
		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      []string{route.Spec.Host},
				SecretName: "tlssecret-replaceme",
			},
		}
	}

	return []runtime.Object{ingress}
}

func (d *Service) serviceToIngress(service v1.Service) []runtime.Object {
	rules := []networkingv1.IngressRule{}
	pathPrefix := "/" + service.Name
	for _, serviceport := range service.Spec.Ports {
		path := pathPrefix
		if len(service.Spec.Ports) > 1 {
			// All ports cannot be exposed as /svcname because they will clash
			path = pathPrefix + "/" + serviceport.Name
			if serviceport.Name == "" {
				path = pathPrefix + "/" + fmt.Sprintf("%d", serviceport.Port)
			}
		}
		rule := networkingv1.IngressRule{
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						{
							Path: path,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: service.Name,
									Port: networkingv1.ServiceBackendPort{Number: serviceport.Port},
								},
							},
						},
					},
				},
			},
			Host: "",
		}
		rules = append(rules, rule)
	}
	ingress := networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       ingressKind,
			APIVersion: networkingv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: service.ObjectMeta,
		Spec: networkingv1.IngressSpec{
			Rules: rules,
		},
	}
	service.Spec.Type = v1.ServiceTypeClusterIP
	return []runtime.Object{&ingress, &service}
}

func (d *Service) routeToService(route okdroutev1.Route) []runtime.Object {
	//TODO: Think through how will the clusterip service that was originally there will behave when merged with this service?
	svc := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       serviceKind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: route.ObjectMeta,
		Spec: v1.ServiceSpec{
			//TODO: How to choose between nodeport and loadbalancer?
			Type: v1.ServiceTypeNodePort,
			Ports: []v1.ServicePort{
				{
					Name: route.Spec.Port.TargetPort.StrVal,
					Port: route.Spec.Port.TargetPort.IntVal,
				},
			},
		},
	}
	svc.Name = route.Spec.To.Name

	return []runtime.Object{svc}
}

func (d *Service) ingressToService(ingress networkingv1.Ingress) []runtime.Object {
	objs := []runtime.Object{}
	for _, ingressspec := range ingress.Spec.Rules {
		for _, path := range ingressspec.IngressRuleValue.HTTP.Paths {
			svc := &v1.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       serviceKind,
					APIVersion: v1.SchemeGroupVersion.String(),
				},
				ObjectMeta: ingress.ObjectMeta,
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeNodePort,
					Ports: []v1.ServicePort{
						{
							//TODO: Check if this is the right mapping
							Name: path.Backend.Service.Port.Name,
							Port: path.Backend.Service.Port.Number,
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

//TODO: Remove these two sections after helm v3 issue is fixed
//[https://github.com/openshift/origin/issues/24060]
//[https://bugzilla.redhat.com/show_bug.cgi?id=1773682]
func (d *Service) createRoute(service irtypes.Service) *okdroutev1.Route {
	var weight int32 = 1                       //Hard-coded to 1 to avoid Helm v3 errors
	var ingressArray []okdroutev1.RouteIngress //Hard-coded to empty list to avoid Helm v3 errors
	ingressArray = append(ingressArray, okdroutev1.RouteIngress{Host: ""})

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
			Port: &okdroutev1.RoutePort{
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: common.DefaultServicePort,
				},
			},
			To: okdroutev1.RouteTargetReference{
				Kind:   serviceKind,
				Name:   service.Name,
				Weight: &weight,
			},
		},
		Status: okdroutev1.RouteStatus{
			Ingress: ingressArray,
		},
	}
	return route
}

// createIngress creates a single ingress for all services
//TODO: Only supports fan-out. Virtual named hosting is not supported yet.
func (d *Service) createIngress(ir irtypes.IR) *networkingv1.Ingress {
	pathType := networkingv1.PathTypePrefix

	// Create the fan-out paths
	httpIngressPaths := []networkingv1.HTTPIngressPath{}
	for _, service := range ir.Services {
		if !service.HasValidAnnotation(common.ExposeSelector) {
			continue
		}
		backendServiceName := service.BackendServiceName
		if service.BackendServiceName == "" {
			backendServiceName = service.Name
		}
		servicePorts := d.getServicePorts(service)
		pathPrefix := service.ServiceRelPath
		for _, servicePort := range servicePorts {
			path := pathPrefix
			if len(servicePorts) > 1 {
				// All ports cannot be exposed as /ServiceRelPath because they will clash
				path = pathPrefix + "/" + servicePort.Name
				if servicePort.Name == "" {
					path = pathPrefix + "/" + fmt.Sprintf("%d", servicePort.Port)
				}
			}
			backendPort := networkingv1.ServiceBackendPort{Name: servicePort.Name}
			if servicePort.Name == "" {
				backendPort = networkingv1.ServiceBackendPort{Number: servicePort.Port}
			}
			httpIngressPath := networkingv1.HTTPIngressPath{
				Path:     path,
				PathType: &pathType,
				Backend: networkingv1.IngressBackend{
					Service: &networkingv1.IngressServiceBackend{
						Name: backendServiceName,
						Port: backendPort,
					},
				},
			}
			httpIngressPaths = append(httpIngressPaths, httpIngressPath)
		}
	}

	// Configure the rule with the above fan-out paths
	rules := []networkingv1.IngressRule{
		{
			Host: ir.TargetClusterSpec.Host,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: httpIngressPaths,
				},
			},
		},
	}

	ingressName := ir.Name
	if len(ir.Services) == 1 {
		for _, service := range ir.Services {
			ingressName = service.Name
		}
	}
	ingress := networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       ingressKind,
			APIVersion: networkingv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   ingressName,
			Labels: getServiceLabels(ingressName),
		},
		Spec: networkingv1.IngressSpec{Rules: rules},
	}
	// If TLS enabled, then add the TLS secret name and the host to the ingress.
	// Otherwise, skip the TLS section.
	if ir.IsIngressTLSEnabled() {
		tls := []networkingv1.IngressTLS{{Hosts: []string{ir.TargetClusterSpec.Host}, SecretName: ir.IngressTLSSecretName}}
		ingress.Spec.TLS = tls
	}

	return &ingress
}

// createService creates a service
func (d *Service) createService(service irtypes.Service, serviceType v1.ServiceType) *v1.Service {
	ports := d.getServicePorts(service)
	svc := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       serviceKind,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        service.Name,
			Labels:      getServiceLabels(service.Name),
			Annotations: getAnnotations(service),
		},
		Spec: v1.ServiceSpec{
			Type:     serviceType,
			Selector: getServiceLabels(service.Name),
			Ports:    ports,
		},
	}
	if len(ports) == 0 {
		svc.Spec.ClusterIP = "None"
	}
	return svc
}

// GetServicePorts configure the container service ports.
func (d *Service) getServicePorts(service irtypes.Service) []v1.ServicePort {
	servicePorts := []v1.ServicePort{}
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
		servicePort := v1.ServicePort{
			Name:       servicePortName,
			Port:       forwarding.ServicePort.Number,
			TargetPort: targetPort,
		}
		servicePorts = append(servicePorts, servicePort)
	}
	return servicePorts
}
