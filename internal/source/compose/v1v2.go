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

package compose

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/libcompose/config"
	"github.com/docker/libcompose/lookup"
	"github.com/docker/libcompose/project"
	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// V1V2Loader loads a compoose file of versions 1 or 2
type V1V2Loader struct {
}

// ParseV2 parses version 2 compose files
func ParseV2(path string) (*project.Project, error) {
	// fileData, err := ioutil.ReadFile(path)
	// if err != nil {
	// 	log.Errorf("Unable to load Compose file at path %s Error: %q", path, err)
	// 	return nil, err
	// }

	context := project.Context{}
	context.ComposeFiles = []string{path}
	context.ResourceLookup = new(lookup.FileResourceLookup)
	//TODO: Check if any variable is mandatory
	someEnvFilePath := ".env"
	if !common.IgnoreEnvironment {
		absSomeEnvFilePath, err := filepath.Abs(someEnvFilePath)
		if err != nil {
			log.Errorf("Failed to make the path %s absolute. Error: %q", someEnvFilePath, err)
			return nil, err
		}
		someEnvFilePath = absSomeEnvFilePath
	}
	context.EnvironmentLookup = &lookup.ComposableEnvLookup{
		Lookups: []config.EnvironmentLookup{
			&lookup.EnvfileLookup{Path: someEnvFilePath},
			&lookup.OsEnvLookup{},
		},
	}
	proj := project.NewProject(&context, nil, nil)
	if err := proj.Parse(); err != nil {
		log.Errorf("Failed to load docker compose file at path %s Error: %q", path, err)
		return nil, err
	}
	return proj, nil
}

// ConvertToIR loads a compose file to IR
func (c *V1V2Loader) ConvertToIR(composefilepath string, plan plantypes.Plan, service plantypes.Service) (ir irtypes.IR, err error) {
	proj, err := ParseV2(composefilepath)
	if err != nil {
		return irtypes.IR{}, err
	}
	return c.convertToIR(filepath.Dir(composefilepath), proj, plan, service)
}

func (c *V1V2Loader) convertToIR(filedir string, composeObject *project.Project, plan plantypes.Plan, service plantypes.Service) (ir irtypes.IR, err error) {
	serviceName := service.ServiceName
	ir = irtypes.IR{
		Services: map[string]irtypes.Service{},
	}

	for name, composeServiceConfig := range composeObject.ServiceConfigs.All() {
		if name != service.ServiceName {
			continue
		}
		serviceConfig := irtypes.NewServiceWithName(common.NormalizeForServiceName(name))
		serviceConfig.Annotations = map[string]string(composeServiceConfig.Labels)
		if composeServiceConfig.Hostname != "" {
			serviceConfig.Hostname = composeServiceConfig.Hostname
		}
		if composeServiceConfig.DomainName != "" {
			serviceConfig.Subdomain = composeServiceConfig.DomainName
		}
		serviceContainer := corev1.Container{}
		serviceContainer.Image = composeServiceConfig.Image
		if serviceContainer.Image == "" {
			serviceContainer.Image = name + ":latest"
		}
		if composeServiceConfig.Build.Dockerfile != "" && composeServiceConfig.Build.Context != "" {
			//TODO: Add support for args and labels
			con, err := new(containerizer.ReuseDockerfileContainerizer).GetContainer(plan, service)
			if err != nil {
				log.Warnf("Unable to get containization script even though build parameters are present : %s", err)
			} else {
				ir.AddContainer(con)
			}
		}
		serviceContainer.Name = strings.ToLower(composeServiceConfig.ContainerName)
		if serviceContainer.Name != composeServiceConfig.ContainerName {
			log.Debugf("Container name in service %q has been changed from %q to %q", name, composeServiceConfig.ContainerName, serviceContainer.Name)
		}
		if serviceContainer.Name == "" {
			serviceContainer.Name = serviceConfig.Name
		}
		serviceContainer.Command = composeServiceConfig.Entrypoint
		serviceContainer.Args = composeServiceConfig.Command
		serviceContainer.Env = c.getEnvs(composeServiceConfig.Environment)
		serviceContainer.WorkingDir = composeServiceConfig.WorkingDir
		serviceContainer.Stdin = composeServiceConfig.StdinOpen
		serviceContainer.TTY = composeServiceConfig.Tty
		ports := c.getPorts(composeServiceConfig.Ports, composeServiceConfig.Expose)
		serviceContainer.Ports = ports
		podSecurityContext := &corev1.PodSecurityContext{}
		securityContext := &corev1.SecurityContext{}
		if composeServiceConfig.Privileged {
			securityContext.Privileged = &composeServiceConfig.Privileged
		}
		if composeServiceConfig.User != "" {
			uid, err := strconv.ParseInt(composeServiceConfig.User, 10, 64)
			if err != nil {
				log.Warn("Ignoring user directive. User to be specified as a UID (numeric).")
			} else {
				securityContext.RunAsUser = &uid
			}

		}
		capsAdd := []corev1.Capability{}
		capsDrop := []corev1.Capability{}
		for _, capAdd := range composeServiceConfig.CapAdd {
			capsAdd = append(capsAdd, corev1.Capability(capAdd))
		}
		for _, capDrop := range composeServiceConfig.CapDrop {
			capsDrop = append(capsDrop, corev1.Capability(capDrop))
		}
		if len(capsAdd) > 0 || len(capsDrop) > 0 {
			securityContext.Capabilities = &corev1.Capabilities{
				Add:  capsAdd,
				Drop: capsDrop,
			}
		}
		if *securityContext != (corev1.SecurityContext{}) {
			serviceContainer.SecurityContext = securityContext
		}
		if !cmp.Equal(*podSecurityContext, corev1.PodSecurityContext{}) {
			serviceConfig.SecurityContext = podSecurityContext
		}
		// group should be in gid format not group name
		groupAdd, err := getGroupAdd(composeServiceConfig.GroupAdd)
		if err != nil {
			log.Warnf("GroupAdd should be in gid format, not as group name : %s", err)
		}
		if groupAdd != nil {
			podSecurityContext.SupplementalGroups = groupAdd
		}
		if composeServiceConfig.StopGracePeriod != "" {
			serviceConfig.TerminationGracePeriodSeconds, err = durationInSeconds(composeServiceConfig.StopGracePeriod)
			if err != nil {
				log.Warnf("Failed to parse duration %v for service %v", composeServiceConfig.StopGracePeriod, name)
			}
		}
		if composeServiceConfig.MemLimit != 0 {
			resourceLimit := corev1.ResourceList{}
			if composeServiceConfig.MemLimit != 0 {
				resourceLimit[corev1.ResourceMemory] = *resource.NewQuantity(int64(composeServiceConfig.MemLimit), "RandomStringForFormat")
			}
			serviceContainer.Resources.Limits = resourceLimit
		}

		restart := composeServiceConfig.Restart
		if restart == "unless-stopped" {
			log.Warnf("Restart policy 'unless-stopped' in service %s is not supported, convert it to 'always'", name)
			serviceConfig.RestartPolicy = corev1.RestartPolicyAlways
		}

		if composeServiceConfig.Networks != nil && len(composeServiceConfig.Networks.Networks) > 0 {
			for _, value := range composeServiceConfig.Networks.Networks {
				if value.Name != "default" {
					serviceConfig.Networks = append(serviceConfig.Networks, value.RealName)
				}
			}
		}

		vml, vl := makeVolumesFromTmpFS(name, composeServiceConfig.Tmpfs)
		for _, v := range vl {
			serviceConfig.AddVolume(v)
		}
		serviceContainer.VolumeMounts = append(serviceContainer.VolumeMounts, vml...)

		if composeServiceConfig.VolumesFrom != nil {
			log.Warnf("Ignoring VolumeFrom in compose for service %s : %s", serviceName, composeServiceConfig.VolumesFrom)
		}

		if composeServiceConfig.Volumes != nil {
			for _, vol := range composeServiceConfig.Volumes.Volumes {
				if isPath(vol.Source) {
					hPath := vol.Source
					if !filepath.IsAbs(vol.Source) {
						hPath, err := filepath.Abs(vol.Source)
						if err != nil {
							log.Debugf("Could not create an absolute path for [%s]", hPath)
						}
					}
					// Generate a hash Id for the given source file path to be mounted.
					hashID := getHash([]byte(hPath))
					volumeName := fmt.Sprintf("%s%d", common.VolumePrefix, hashID)
					serviceContainer.VolumeMounts = append(serviceContainer.VolumeMounts, corev1.VolumeMount{
						Name:      volumeName,
						ReadOnly:  vol.AccessMode == modeReadOnly,
						MountPath: vol.Destination,
					})

					serviceConfig.AddVolume(corev1.Volume{
						Name: volumeName,
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: vol.Source},
						},
					})
				} else {
					serviceContainer.VolumeMounts = append(serviceContainer.VolumeMounts, corev1.VolumeMount{
						Name:      vol.Source,
						ReadOnly:  vol.AccessMode == modeReadOnly,
						MountPath: vol.Destination,
					})

					serviceConfig.AddVolume(corev1.Volume{
						Name: vol.Source,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: vol.Source,
								ReadOnly:  vol.AccessMode == modeReadOnly,
							},
						},
					})
					accessMode := corev1.ReadWriteMany
					if vol.AccessMode == modeReadOnly {
						accessMode = corev1.ReadOnlyMany
					}
					storageObj := irtypes.Storage{StorageType: irtypes.PVCKind, Name: vol.Source, Content: nil}
					storageObj.PersistentVolumeClaimSpec = corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
					}
					ir.AddStorage(storageObj)
				}
			}
		}

		serviceConfig.Containers = []v1.Container{serviceContainer}
		ir.Services[name] = serviceConfig
	}

	return ir, nil
}

func (c *V1V2Loader) getEnvs(envars []string) []corev1.EnvVar {
	envs := []corev1.EnvVar{}
	for _, e := range envars {
		m := regexp.MustCompile(`[=:]`)
		locs := m.FindStringIndex(e)
		if locs == nil || len(locs) < 1 {
			envs = append(envs, corev1.EnvVar{
				Name:  e,
				Value: "unknown",
			})
		} else {
			envs = append(envs, corev1.EnvVar{
				Name:  e[locs[0]+1:],
				Value: e[:locs[0]],
			})
		}
	}
	return envs
}

func (c *V1V2Loader) getPorts(composePorts []string, expose []string) []corev1.ContainerPort {
	ports := []corev1.ContainerPort{}
	exist := map[int32]bool{}
	for _, port := range composePorts {
		cp := c.getContainerPort(port)
		if !exist[cp.ContainerPort] && cp != (corev1.ContainerPort{}) {
			ports = append(ports, cp)
			exist[cp.ContainerPort] = true
		}
	}
	for _, port := range expose {
		cp := c.getContainerPort(port)
		if !exist[cp.ContainerPort] && cp != (corev1.ContainerPort{}) {
			ports = append(ports, cp)
			exist[cp.ContainerPort] = true
		}
	}

	return ports
}

func (c *V1V2Loader) getContainerPort(value string) (cp corev1.ContainerPort) {
	// 15000:15000/tcp  Default protocol TCP
	proto := corev1.ProtocolTCP
	parts := strings.Split(value, "/")
	if len(parts) == 2 && strings.EqualFold(string(corev1.ProtocolUDP), parts[1]) {
		proto = corev1.ProtocolUDP
	}
	// Split up the ports and IP
	justPorts := strings.Split(parts[0], ":")
	if len(justPorts) > 0 {
		// ex. 127.0.0.1:80:80
		// Get the container port
		portStr := justPorts[len(justPorts)-1]
		p, err := strconv.Atoi(portStr)
		if err != nil {
			log.Warnf("Invalid container port in %s ; Example: 127.0.0.1:80:80 or 80:80 or 80", parts[0])
		} else {
			cp = corev1.ContainerPort{
				ContainerPort: int32(p),
				Protocol:      proto,
			}
		}
	}
	return
}

func getGroupAdd(group []string) ([]int64, error) {
	var groupAdd []int64
	for _, i := range group {
		j, err := strconv.Atoi(i)
		if err != nil {
			return nil, errors.Wrap(err, "unable to get group_add")
		}
		groupAdd = append(groupAdd, int64(j))

	}
	return groupAdd, nil
}

func durationInSeconds(s string) (*int64, error) {
	if s == "" {
		return nil, nil
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return nil, err
	}
	r := (int64)(duration.Seconds())
	return &r, nil
}
