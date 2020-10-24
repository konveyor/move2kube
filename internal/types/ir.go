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

package types

import (
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/types/tekton"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	outputtypes "github.com/konveyor/move2kube/types/output"
	"github.com/konveyor/move2kube/types/plan"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// IR is the intermediate representation
type IR struct {
	Name            string
	Services        map[string]Service
	Storages        []Storage
	Containers      []Container
	Roles           []Role
	RoleBindings    []RoleBinding
	ServiceAccounts []ServiceAccount

	Kubernetes plan.KubernetesOutput

	TargetClusterSpec collecttypes.ClusterMetadataSpec
	CachedObjects     []runtime.Object

	Values outputtypes.HelmValues

	IngressTLSSecretName string
	TektonResources      tekton.Resources
}

// Service defines structure of an IR service
type Service struct {
	corev1.PodSpec

	Name               string
	BackendServiceName string // Optional field when ingress name is not the same as backend service name
	Annotations        map[string]string
	Labels             map[string]string
	Replicas           int
	Networks           []string
	ServiceRelPath     string //Ingress fan-out path
	ExposeService      bool
	OnlyIngress        bool
	Daemon             bool //Gets converted to DaemonSet
}

// AddVolume adds a volume to a service
func (service *Service) AddVolume(volume corev1.Volume) {
	merged := false
	for _, existingVolume := range service.Volumes {
		if existingVolume.Name == volume.Name {
			log.Debugf("Found an existing volume. Ignoring new volume : %+v", volume)
			merged = true
			break
		}
	}
	if !merged {
		service.Volumes = append(service.Volumes, volume)
	}
}

// Container defines structure of a container
type Container struct {
	ContainerBuildType plantypes.ContainerBuildTypeValue
	RepoInfo           plantypes.RepoInfo
	ImageNames         []string
	New                bool
	NewFiles           map[string]string //[filename][filecontents]
	ExposedPorts       []int
	UserID             int
	AccessedDirs       []string
}

// NewContainer creates a new container
func NewContainer(containerBuildType plantypes.ContainerBuildTypeValue, imagename string, new bool) Container {
	return Container{
		ContainerBuildType: containerBuildType,
		ImageNames:         []string{imagename},
		New:                new,
		NewFiles:           map[string]string{},
		ExposedPorts:       []int{},
		UserID:             -1,
		AccessedDirs:       []string{},
	}
}

// NewContainerFromImageInfo creates a new container from image info
func NewContainerFromImageInfo(i collecttypes.ImageInfo) Container {
	imagename := ""
	if len(i.Spec.Tags) > 0 {
		imagename = i.Spec.Tags[0]
	} else {
		log.Errorf("The image info %v has no tags. Leaving the tag empty for the container.", i)
	}
	c := NewContainer(plantypes.ReuseContainerBuildTypeValue, imagename, false)
	c.ImageNames = i.Spec.Tags
	c.ExposedPorts = i.Spec.PortsToExpose
	c.UserID = i.Spec.UserID
	c.AccessedDirs = i.Spec.AccessedDirs
	return c
}

// Merge merges containers
func (c *Container) Merge(newc Container) bool {
	if c.ContainerBuildType != newc.ContainerBuildType {
		return false
	}
	for _, imagename := range newc.ImageNames {
		if common.IsStringPresent(c.ImageNames, imagename) {
			if c.New != newc.New {
				log.Errorf("Both old and new image seems to share the same tag for container %s.", imagename)
			} else if c.New && newc.New {
				for filepath, filecontents := range newc.NewFiles {
					if contents, ok := c.NewFiles[filepath]; ok {
						if contents != filecontents {
							log.Errorf("Two build scripts found for image : %s in %s. Ignoring new script.", imagename, filepath)
						}
					} else {
						c.NewFiles[filepath] = filecontents
					}
				}
				if c.UserID != newc.UserID {
					log.Errorf("Two different users found for image : %d in %d. Ignoring new users.", c.UserID, newc.UserID)
				}
			}
			c.ImageNames = common.MergeStringSlices(c.ImageNames, newc.ImageNames)
			c.ExposedPorts = common.MergeIntSlices(c.ExposedPorts, newc.ExposedPorts)
			c.AccessedDirs = common.MergeStringSlices(c.AccessedDirs, newc.AccessedDirs) //Needs to be clarified
			if !c.New {
				c.NewFiles = newc.NewFiles
				c.UserID = newc.UserID //Needs to be clarified
			}
			return true
		}
		log.Debugf("Mismatching during container merge [%s, %s]", c.ImageNames, imagename)
	}
	return false
}

// IsServiceExposed checks if a service is exposed through ingress or not
func (service *Service) IsServiceExposed() bool {
	return service.ServiceRelPath != ""
}

// AddFile adds a file to a container
func (c *Container) AddFile(path string, newcontents string) {
	if contents, ok := c.NewFiles[path]; ok {
		if contents != newcontents {
			log.Errorf("Script already exists for image at %s. Ignoring new script.", path)
		}
	} else {
		c.NewFiles[path] = newcontents
	}
}

// AddExposedPort adds an exposed port to a container
func (c *Container) AddExposedPort(port int) {
	if !common.IsIntPresent(c.ExposedPorts, port) {
		c.ExposedPorts = append(c.ExposedPorts, port)
	}
}

// AddImageName adds image name to a container
func (c *Container) AddImageName(imagename string) {
	if !common.IsStringPresent(c.ImageNames, imagename) {
		c.ImageNames = append(c.ImageNames, imagename)
	}
}

// AddAccessedDirs adds accessed directories to container
func (c *Container) AddAccessedDirs(dirname string) {
	if !common.IsStringPresent(c.AccessedDirs, dirname) {
		c.AccessedDirs = append(c.AccessedDirs, dirname)
	}
}

// NewIR creates a new IR
func NewIR(p plan.Plan) IR {
	var ir IR
	ir.Name = p.Name
	ir.Kubernetes = p.Spec.Outputs.Kubernetes
	ir.Containers = []Container{}
	ir.Services = map[string]Service{}
	ir.Storages = []Storage{}
	ir.TargetClusterSpec = collecttypes.ClusterMetadataSpec{
		StorageClasses:    []string{},
		APIKindVersionMap: map[string][]string{},
		Host:              "",
	}
	ir.Values.GlobalVariables = map[string]string{}
	return ir
}

// Merge merges IRs
func (ir *IR) Merge(newir IR) {
	if ir.Name != newir.Name {
		if ir.Name == "" {
			ir.Name = newir.Name
		}
	}
	ir.Kubernetes.Merge(newir.Kubernetes)
	for scname, sc := range newir.Services {
		if _, ok := ir.Services[scname]; ok {
			log.Warnf("Two services of same service name %s. Using the new object.", scname)
		}
		ir.Services[scname] = sc
	}
	for _, newcontainer := range newir.Containers {
		ir.AddContainer(newcontainer)
	}
	for _, newst := range newir.Storages {
		ir.AddStorage(newst)
	}
	ir.TargetClusterSpec.Merge(newir.TargetClusterSpec)
	ir.CachedObjects = append(ir.CachedObjects, newir.CachedObjects...)
	ir.Values.Merge(newir.Values)
}

// IsIngressTLSEnabled checks if TLS is enabled for the ingress.
func (ir *IR) IsIngressTLSEnabled() bool {
	return ir.IngressTLSSecretName != ""
}

// NewServiceFromPlanService initializes a service with just the plan object parameters.
func NewServiceFromPlanService(service plantypes.Service) Service {
	return Service{Name: service.ServiceName, ServiceRelPath: service.ServiceRelPath}
}

// NewServiceWithName initializes a service with just the name.
func NewServiceWithName(serviceName string) Service {
	return Service{Name: serviceName, ServiceRelPath: "/" + serviceName}
}

// StorageKindType defines storage type kind
type StorageKindType string

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

// Storage defines structure of a storage
type Storage struct {
	Name                             string
	Annotations                      map[string]string // Optional field to store arbitrary metadata
	corev1.PersistentVolumeClaimSpec                   //This promotion contains the volumeName which is used by configmap, secrets and pvc.
	StorageType                      StorageKindType   //Type of storage cfgmap, secret, pvc
	SecretType                       corev1.SecretType // Optional field to store the type of secret data
	Content                          map[string][]byte //Optional field meant to store content for cfgmap or secret
	StringData                       map[string]string //Optional field to store string content for cfmap or secret
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
	log.Debugf("Mismatching storages [%s, %s]", s.Name, newst.Name)
	return false
}

// ServiceAccount holds the details about the service account resource
type ServiceAccount struct {
	Name        string
	SecretNames []string
}

// RoleBinding holds the details about the role binding resource
type RoleBinding struct {
	Name               string
	RoleName           string
	ServiceAccountName string
}

// Role holds the details about the role resource
type Role struct {
	Name        string
	PolicyRules []PolicyRule
}

// PolicyRule holds the details about the policy rules for the service account resources
type PolicyRule struct {
	APIGroups []string
	Resources []string
	Verbs     []string
}

// Secret holds the details about the secret resource
// type Secret struct {
// 	Name        string
// 	SecretType  corev1.SecretType
// 	Annotations map[string]string
// 	StringData  map[string]string
// }

// AddContainer adds a conatainer to IR
func (ir *IR) AddContainer(container Container) {
	merged := false
	for i := range ir.Containers {
		if ir.Containers[i].Merge(container) {
			merged = true
			break
		}
	}
	if !merged {
		ir.Containers = append(ir.Containers, container)
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

// GetContainer returns container which has the imagename
func (ir *IR) GetContainer(imagename string) (con Container, exists bool) {
	for _, c := range ir.Containers {
		if common.IsStringPresent(c.ImageNames, imagename) {
			return c, true
		} else if c.New {
			parts := strings.Split(imagename, "/")
			if len(parts) > 2 && parts[0] == ir.Kubernetes.RegistryURL && common.IsStringPresent(c.ImageNames, parts[len(parts)-1]) {
				return c, true
			}
		}
	}
	return Container{}, false
}
