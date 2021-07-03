/*
 *  Copyright IBM Corporation 2020, 2021
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

package ir

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/deepcopy"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	core "k8s.io/kubernetes/pkg/apis/core"
	networking "k8s.io/kubernetes/pkg/apis/networking"
)

// IRArtifactType represents artifact type of IR
const IRArtifactType transformertypes.ArtifactType = "IR"

// IRConfigType represents config type of IR
const IRConfigType transformertypes.ConfigType = "IR"

const (
	// DockerfileContainerBuildType represents dockerfile container build type
	DockerfileContainerBuildType ContainerBuildTypeValue = "Dockerfile"
	// S2IContainerBuildTypeValue represents s2i container build type
	S2IContainerBuildTypeValue ContainerBuildTypeValue = "S2I"
	// CNBContainerBuildTypeValue represents CNB container build type
	CNBContainerBuildTypeValue ContainerBuildTypeValue = "CNB"
)

// IR is the intermediate representation filled by source transformers
type IR struct {
	ContainerImages map[string]ContainerImage // [imageName]
	Services        map[string]Service
	Storages        []Storage
}

// Service defines structure of an IR service
type Service struct {
	core.PodSpec

	Name                        string
	BackendServiceName          string // Optional field when ingress name is not the same as backend service name
	Annotations                 map[string]string
	Labels                      map[string]string
	ServiceToPodPortForwardings []ServiceToPodPortForwarding
	Replicas                    int
	Networks                    []string
	ServiceRelPath              string //Ingress fan-out path
	OnlyIngress                 bool
	Daemon                      bool //Gets converted to DaemonSet
}

// Port is a port number with an optional port name.
type Port networking.ServiceBackendPort

// ServiceToPodPortForwarding forwards a k8s service port to a k8s pod port
type ServiceToPodPortForwarding struct {
	ServicePort Port
	PodPort     Port
}

// ContainerBuildTypeValue stores the container build type
type ContainerBuildTypeValue string

// ContainerBuildArtifactTypeValue stores the container build artifact type
type ContainerBuildArtifactTypeValue string

// ContainerImage defines images that need to be built or reused.
type ContainerImage struct {
	ExposedPorts []int    `yaml:"ports"`
	UserID       int      `yaml:"userID"`
	AccessedDirs []string `yaml:"accessedDirs"`
	Build        ContainerBuild
}

// ContainerBuild stores information about the container build
type ContainerBuild struct {
	ContainerBuildType ContainerBuildTypeValue                      `yaml:"-"`
	ContextPath        string                                       `yaml:"-"`
	Artifacts          map[ContainerBuildArtifactTypeValue][]string `yaml:"-"` //[artifacttype]value
}

// StorageKindType defines storage type kind
type StorageKindType string

// Storage defines structure of a storage
type Storage struct {
	Name                           string
	Annotations                    map[string]string // Optional field to store arbitrary metadata
	core.PersistentVolumeClaimSpec                   //This promotion contains the volumeName which is used by configmap, secrets and pvc.
	StorageType                    StorageKindType   //Type of storage cfgmap, secret, pvc
	SecretType                     core.SecretType   // Optional field to store the type of secret data
	Content                        map[string][]byte //Optional field meant to store content for cfgmap or secret
}

const (
	// SecretKind defines storage type of Secret
	SecretKind StorageKindType = "Secret"
	// ConfigMapKind defines storage type of ConfigMap
	ConfigMapKind StorageKindType = "ConfigMap"
	// PVCKind defines storage type of PersistentVolumeClaim
	PVCKind StorageKindType = "PersistentVolumeClaim"
	// PullSecretKind defines storage type of pull secret
	PullSecretKind StorageKindType = "PullSecret"
)

// NewEnhancedIRFromIR returns a new EnhancedIR given an IR
// It makes a deep copy of the IR before embedding it in the EnhancedIR.
func NewEnhancedIRFromIR(ir IR) EnhancedIR {
	irCopy := deepcopy.DeepCopy(ir).(IR)
	return EnhancedIR{IR: irCopy}
}

func (ir *IR) addService(service Service) {
	if os, ok := ir.Services[service.Name]; !ok {
		ir.Services[service.Name] = service
	} else {
		os.merge(service)
		ir.Services[service.Name] = os
	}
}

func (service *Service) merge(nService Service) {
	if service.Name != nService.Name {
		return
	}
	if service.BackendServiceName != nService.BackendServiceName && service.BackendServiceName != "" {
		logrus.Errorf("BackendServiceNames (%s, %s) don't seem to match during merge for service : %s. Using %s", service.BackendServiceName, nService.BackendServiceName, service.Name, nService.BackendServiceName)
	}
	if nService.BackendServiceName != "" {
		service.BackendServiceName = nService.BackendServiceName
	}
	podSpecJSON, err1 := json.Marshal(service.PodSpec)
	if err1 != nil {
		logrus.Errorf("Merge failed. Failed to marshal the first object %v to json. Error: %q", service.PodSpec, err1)
	}
	nPodSpecJSON, err2 := json.Marshal(nService.PodSpec)
	if err2 != nil {
		logrus.Errorf("Merge failed. Failed to marshal the second object %v to json. Error: %q", nService.PodSpec, err2)
	}
	if err1 != nil || err2 != nil {
		podSpec := core.PodSpec{}
		mergedJSON, err := strategicpatch.StrategicMergePatch(podSpecJSON, nPodSpecJSON, podSpec) // need to provide in reverse for proper ordering
		if err != nil {
			logrus.Errorf("Failed to merge the objects \n%s\n and \n%s\n Error: %q", podSpecJSON, nPodSpecJSON, err)
		} else {
			err := json.Unmarshal(mergedJSON, &podSpec)
			if err != nil {
				logrus.Errorf("Failed to unmarshall object (%+v): %q", podSpec, err)
			} else {
				service.PodSpec = podSpec
			}
		}
	}
	service.Annotations = common.MergeStringMaps(service.Annotations, nService.Annotations)
	service.Labels = common.MergeStringMaps(service.Labels, nService.Labels)
	if nService.Replicas != 0 {
		service.Replicas = nService.Replicas
	}
	service.Networks = common.MergeStringSlices(service.Networks, nService.Networks...)
	if nService.ServiceRelPath != "" {
		service.ServiceRelPath = nService.ServiceRelPath
	}
	service.OnlyIngress = service.OnlyIngress && nService.OnlyIngress
	service.Daemon = service.Daemon && nService.Daemon
	// TODO: Check if this needs a more intelligent merge
	service.ServiceToPodPortForwardings = append(service.ServiceToPodPortForwardings, nService.ServiceToPodPortForwardings...)
}

// AddPortForwarding adds a new port forwarding to the service.
func (service *Service) AddPortForwarding(servicePort Port, podPort Port) error {
	for _, forwarding := range service.ServiceToPodPortForwardings {
		if servicePort.Name != "" && forwarding.ServicePort.Name == servicePort.Name {
			err := fmt.Errorf("the port name %s on %s service is already in use. Not adding the new forwarding", servicePort.Name, service.Name)
			logrus.Warn(err)
			return err
		}
		if forwarding.ServicePort.Number == servicePort.Number {
			err := fmt.Errorf("the port number %d on %s service is already in use. Not adding the new forwarding", servicePort.Number, service.Name)
			logrus.Warn(err)
			return err
		}
	}
	newForwarding := ServiceToPodPortForwarding{ServicePort: servicePort, PodPort: podPort}
	service.ServiceToPodPortForwardings = append(service.ServiceToPodPortForwardings, newForwarding)
	return nil
}

// AddVolume adds a volume to a service
func (service *Service) AddVolume(volume core.Volume) {
	merged := false
	for _, existingVolume := range service.Volumes {
		if existingVolume.Name == volume.Name {
			logrus.Debugf("Found an existing volume. Ignoring new volume : %+v", volume)
			merged = true
			break
		}
	}
	if !merged {
		service.Volumes = append(service.Volumes, volume)
	}
}

// HasValidAnnotation returns if an annotation is set for the service
func (service *Service) HasValidAnnotation(annotation string) bool {
	val, ok := service.Annotations[annotation]
	return ok && val == common.AnnotationLabelValue
}

// NewContainer creates a new container
func NewContainer() ContainerImage {
	return ContainerImage{
		ExposedPorts: []int{},
		UserID:       -1,
		AccessedDirs: []string{},
	}
}

// Merge merges containers
func (c *ContainerImage) Merge(newc ContainerImage) bool {
	if c.UserID != newc.UserID {
		logrus.Errorf("Two different users found for image : %d in %d. Ignoring new users.", c.UserID, newc.UserID)
	}
	c.ExposedPorts = common.MergeIntSlices(c.ExposedPorts, newc.ExposedPorts)
	c.AccessedDirs = common.MergeStringSlices(c.AccessedDirs, newc.AccessedDirs...)
	c.Build.Merge(newc.Build)
	return true
}

// Merge merges two container build structs
func (c *ContainerBuild) Merge(newc ContainerBuild) bool {
	if c.ContainerBuildType == "" {
		c = &newc
		return true
	}
	if newc.ContainerBuildType == "" {
		return true
	}
	if c.ContainerBuildType != newc.ContainerBuildType {
		logrus.Errorf("Incompatible container build types %s and %s can not be merged", c.ContainerBuildType, newc.ContainerBuildType)
		return false
	}
	if c.ContextPath == "" {
		c.ContextPath = newc.ContextPath
	}
	return true
}

// AddExposedPort adds an exposed port to a container
func (c *ContainerImage) AddExposedPort(port int) {
	if !common.IsIntPresent(c.ExposedPorts, port) {
		c.ExposedPorts = append(c.ExposedPorts, port)
	}
}

// AddAccessedDirs adds accessed directories to container
func (c *ContainerImage) AddAccessedDirs(dirname string) {
	if !common.IsStringPresent(c.AccessedDirs, dirname) {
		c.AccessedDirs = append(c.AccessedDirs, dirname)
	}
}

// NewIR creates a new IR
func NewIR() IR {
	ir := IR{}
	ir.ContainerImages = map[string]ContainerImage{}
	ir.Services = map[string]Service{}
	ir.Storages = []Storage{}
	return ir
}

// Merge merges IRs
func (ir *IR) Merge(newir IR) {
	for _, sc := range newir.Services {
		ir.addService(sc)
	}
	for in, newcontainer := range newir.ContainerImages {
		ir.AddContainer(in, newcontainer)
	}
	for _, newst := range newir.Storages {
		ir.AddStorage(newst)
	}
}

// NewServiceWithName initializes a service with just the name.
func NewServiceWithName(serviceName string) Service {
	return Service{Name: serviceName, ServiceRelPath: "/" + serviceName}
}

// Merge merges storage
func (s *Storage) Merge(newst Storage) bool {
	if strings.Compare(s.Name, newst.Name) == 0 {
		if s.Content != nil && newst.Content != nil {
			s.Content = newst.Content
		}
		s.StorageType = newst.StorageType
		s.PersistentVolumeClaimSpec = newst.PersistentVolumeClaimSpec
		return true
	}
	logrus.Debugf("Mismatching storages [%s, %s]", s.Name, newst.Name)
	return false
}

// AddContainer adds a conatainer to IR
func (ir *IR) AddContainer(imageName string, container ContainerImage) {
	if im, ok := ir.ContainerImages[imageName]; ok {
		im.Merge(container)
		ir.ContainerImages[imageName] = im
	} else {
		ir.ContainerImages[imageName] = container
	}
}

// AddStorage adds a storage to IR
func (ir *IR) AddStorage(st Storage) {
	merged := false
	for i := range ir.Storages {
		if ir.Storages[i].Merge(st) {
			merged = true
			break
		}
	}
	if !merged {
		ir.Storages = append(ir.Storages, st)
	}
}
