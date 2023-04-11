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

package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/docker/libcompose/config"
	"github.com/docker/libcompose/lookup"
	"github.com/docker/libcompose/project"
	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"k8s.io/apimachinery/pkg/api/resource"
	core "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/networking"
)

// v1v2Loader loads a compoose file of versions 1 or 2
type v1v2Loader struct {
}

type preprocessFunc func(rawServiceMap config.RawServiceMap) (config.RawServiceMap, error)

func removeNonExistentEnvFilesV2(path string) preprocessFunc {
	composeFileDir := filepath.Dir(path)
	return func(rawServiceMap config.RawServiceMap) (config.RawServiceMap, error) {
		// Remove unresolvable env files, so that the parser does not throw error
		for serviceName, vals := range rawServiceMap {
			if envfilesvals, ok := vals[envFile]; ok {
				// env_file can be a string or list of strings
				// https://docs.docker.com/compose/compose-file/compose-file-v2/#env_file
				if envfilesstr, ok := envfilesvals.(string); ok {
					envFilePath := envfilesstr
					if !filepath.IsAbs(envFilePath) {
						envFilePath = filepath.Join(composeFileDir, envFilePath)
					}
					finfo, err := os.Stat(envFilePath)
					if os.IsNotExist(err) || finfo.IsDir() {
						logrus.Warnf("Unable to find env config file %s referred in service %s in file %s. Ignoring it.", envFilePath, serviceName, path)
						delete(vals, envFile)
					}
				} else if envfilesvalsint, ok := envfilesvals.([]interface{}); ok {
					envfiles := []interface{}{}
					for _, envfilesval := range envfilesvalsint {
						if envfilesstr, ok := envfilesval.(string); ok {
							envFilePath := envfilesstr
							if !filepath.IsAbs(envFilePath) {
								envFilePath = filepath.Join(composeFileDir, envFilePath)
							}
							finfo, err := os.Stat(envFilePath)
							if os.IsNotExist(err) || finfo.IsDir() {
								logrus.Warnf("Unable to find env config file %s referred in service %s in file %s. Ignoring it.", envFilePath, serviceName, path)
								continue
							}
							envfiles = append(envfiles, envfilesstr)
						}
					}
					vals[envFile] = envfiles
				}
			}
		}

		return rawServiceMap, nil
	}
}

// parseV2 parses version 2 compose files
func parseV2(path string, interpolate bool) (*project.Project, error) {
	context := project.Context{}
	context.ComposeFiles = []string{path}
	context.ResourceLookup = new(lookup.FileResourceLookup)
	//TODO: Check if any variable is mandatory
	var lookUps []config.EnvironmentLookup
	composeFileDir := filepath.Dir(path)
	someEnvFilePath := filepath.Join(composeFileDir, defaultEnvFile)
	_, err := os.Stat(someEnvFilePath)
	if err != nil {
		logrus.Debugf("Failed to find the env path %s. Error: %q", someEnvFilePath, err)
	} else {
		lookUps = append(lookUps, &lookup.EnvfileLookup{Path: someEnvFilePath})
	}
	if !common.IgnoreEnvironment {
		lookUps = append(lookUps, &lookup.OsEnvLookup{})
	}
	context.EnvironmentLookup = &lookup.ComposableEnvLookup{Lookups: lookUps}
	parseOptions := config.ParseOptions{
		Interpolate: interpolate,
		Validate:    true,
		Preprocess:  removeNonExistentEnvFilesV2(path),
	}
	proj := project.NewProject(&context, nil, &parseOptions)
	originalLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.FatalLevel) // TODO: this is a hack to prevent libcompose from printing errors to the console.
	err = proj.Parse()
	logrus.SetLevel(originalLevel) // TODO: this is a hack to prevent libcompose from printing errors to the console.
	if err != nil {
		err := fmt.Errorf("failed to load docker compose file at path %s Error: %q", path, err)
		logrus.Debug(err)
		return nil, err
	}
	return proj, nil
}

// ConvertToIR loads a compose file to IR
func (c *v1v2Loader) ConvertToIR(composefilepath string, serviceName string, parseNetwork bool) (ir irtypes.IR, err error) {
	proj, err := parseV2(composefilepath, true)
	if err != nil {
		return irtypes.IR{}, err
	}
	return c.convertToIR(filepath.Dir(composefilepath), proj, serviceName, parseNetwork)
}

func (c *v1v2Loader) convertToIR(filedir string, composeObject *project.Project, serviceName string, parseNetwork bool) (ir irtypes.IR, err error) {
	ir = irtypes.IR{
		Services: map[string]irtypes.Service{},
	}
	storageMap := map[string]bool{}
	for name, composeServiceConfig := range composeObject.ServiceConfigs.All() {
		if name != serviceName {
			continue
		}
		serviceConfig := irtypes.NewServiceWithName(common.NormalizeForMetadataName(name))
		serviceConfig.Annotations = map[string]string(composeServiceConfig.Labels)
		if composeServiceConfig.Hostname != "" {
			serviceConfig.Hostname = composeServiceConfig.Hostname
		}
		if composeServiceConfig.DomainName != "" {
			serviceConfig.Subdomain = composeServiceConfig.DomainName
		}
		serviceContainer := core.Container{}
		serviceContainer.Image = composeServiceConfig.Image
		if serviceContainer.Image == "" {
			serviceContainer.Image = name + ":latest"
		}
		if composeServiceConfig.Build.Dockerfile != "" || composeServiceConfig.Build.Context != "" {
			if composeServiceConfig.Build.Dockerfile == "" {
				composeServiceConfig.Build.Dockerfile = filepath.Join(composeServiceConfig.Build.Context, common.DefaultDockerfileName)
			}
			if ir.ContainerImages == nil {
				ir.ContainerImages = map[string]irtypes.ContainerImage{}
			}
			ir.ContainerImages[serviceContainer.Image] = irtypes.ContainerImage{
				Build: irtypes.ContainerBuild{
					ContainerBuildType: irtypes.DockerfileContainerBuildType,
					ContextPath:        filepath.Join(filedir, composeServiceConfig.Build.Context),
					Artifacts: map[irtypes.ContainerBuildArtifactTypeValue][]string{
						irtypes.DockerfileContainerBuildArtifactTypeValue: {filepath.Join(filedir, composeServiceConfig.Build.Dockerfile)},
					},
				},
			}
		}
		if composeServiceConfig.ContainerName == "" {
			composeServiceConfig.ContainerName = serviceConfig.Name
		}
		serviceContainer.Name = common.NormalizeForMetadataName(composeServiceConfig.ContainerName)
		if serviceContainer.Name != composeServiceConfig.ContainerName {
			logrus.Debugf("Container name in service %q has been changed from %q to %q", name, composeServiceConfig.ContainerName, serviceContainer.Name)
		}
		serviceContainer.Command = composeServiceConfig.Entrypoint
		serviceContainer.Args = composeServiceConfig.Command
		serviceContainer.Env = c.getEnvs(composeServiceConfig.Environment)
		serviceContainer.WorkingDir = composeServiceConfig.WorkingDir
		serviceContainer.Stdin = composeServiceConfig.StdinOpen
		serviceContainer.TTY = composeServiceConfig.Tty
		serviceContainer.Ports = c.getPorts(composeServiceConfig.Ports, composeServiceConfig.Expose)
		c.addPorts(composeServiceConfig.Ports, composeServiceConfig.Expose, &serviceConfig)
		podSecurityContext := &core.PodSecurityContext{}
		securityContext := &core.SecurityContext{}
		if composeServiceConfig.Privileged {
			securityContext.Privileged = &composeServiceConfig.Privileged
		}
		if composeServiceConfig.User != "" {
			uid, err := cast.ToInt64E(composeServiceConfig.User)
			if err != nil {
				logrus.Warn("Ignoring user directive. User to be specified as a UID (numeric).")
			} else {
				securityContext.RunAsUser = &uid
			}

		}
		capsAdd := []core.Capability{}
		capsDrop := []core.Capability{}
		for _, capAdd := range composeServiceConfig.CapAdd {
			capsAdd = append(capsAdd, core.Capability(capAdd))
		}
		for _, capDrop := range composeServiceConfig.CapDrop {
			capsDrop = append(capsDrop, core.Capability(capDrop))
		}
		if len(capsAdd) > 0 || len(capsDrop) > 0 {
			securityContext.Capabilities = &core.Capabilities{
				Add:  capsAdd,
				Drop: capsDrop,
			}
		}
		if *securityContext != (core.SecurityContext{}) {
			serviceContainer.SecurityContext = securityContext
		}
		if !cmp.Equal(*podSecurityContext, core.PodSecurityContext{}) {
			serviceConfig.SecurityContext = podSecurityContext
		}
		// group should be in gid format not group name
		groupAdd, err := getGroupAdd(composeServiceConfig.GroupAdd)
		if err != nil {
			logrus.Warnf("GroupAdd should be in gid format, not as group name : %s", err)
		}
		if groupAdd != nil {
			podSecurityContext.SupplementalGroups = groupAdd
		}
		if composeServiceConfig.StopGracePeriod != "" {
			serviceConfig.TerminationGracePeriodSeconds, err = durationInSeconds(composeServiceConfig.StopGracePeriod)
			if err != nil {
				logrus.Warnf("Failed to parse duration %v for service %v", composeServiceConfig.StopGracePeriod, name)
			}
		}
		if composeServiceConfig.MemLimit != 0 {
			resourceLimit := core.ResourceList{}
			if composeServiceConfig.MemLimit != 0 {
				resourceLimit[core.ResourceMemory] = *resource.NewQuantity(int64(composeServiceConfig.MemLimit), "RandomStringForFormat")
			}
			serviceContainer.Resources.Limits = resourceLimit
		}

		restart := composeServiceConfig.Restart
		if restart == "unless-stopped" {
			logrus.Warnf("Restart policy 'unless-stopped' in service %s is not supported, convert it to 'always'", name)
			serviceConfig.RestartPolicy = core.RestartPolicyAlways
		}

		if parseNetwork && composeServiceConfig.Networks != nil && len(composeServiceConfig.Networks.Networks) > 0 {
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
			logrus.Warnf("Ignoring VolumeFrom in compose for service %s : %s", serviceName, composeServiceConfig.VolumesFrom)
		}
		if composeServiceConfig.Volumes != nil {
			for _, vol := range composeServiceConfig.Volumes.Volumes {
				volumeMount, volume, storage, err := applyVolumePolicy(filedir, serviceName, vol.Source, vol.Destination, vol.AccessMode, storageMap)
				if err != nil {
					logrus.Warnf("Could not create storage: [%s]", err)
					continue
				}
				if volumeMount != nil {
					serviceContainer.VolumeMounts = append(serviceContainer.VolumeMounts, *volumeMount)
				}
				if volume != nil {
					serviceConfig.AddVolume(*volume)
				}
				if storage != nil {
					ir.AddStorage(*storage)
					ir.Storages = append(ir.Storages, *storage)
					storageMap[storage.Name] = true
				}
			}
		}
		serviceConfig.Containers = []core.Container{serviceContainer}
		ir.Services[name] = serviceConfig
	}
	return ir, nil
}

func (c *v1v2Loader) getEnvs(envars []string) []core.EnvVar {
	envs := []core.EnvVar{}
	for _, e := range envars {
		m := regexp.MustCompile(`[=:]`)
		locs := m.FindStringIndex(e)
		if locs == nil || len(locs) < 1 {
			envs = append(envs, core.EnvVar{
				Name:  e,
				Value: "unknown",
			})
		} else {
			envs = append(envs, core.EnvVar{
				Name:  e[:locs[0]],
				Value: e[locs[0]+1:],
			})
		}
	}
	return envs
}

func (c *v1v2Loader) getPorts(composePorts []string, expose []string) []core.ContainerPort {
	ports := []core.ContainerPort{}
	exist := map[int]bool{}
	for _, port := range composePorts {
		_, podPort, protocol, err := c.parseContainerPort(port)
		if err != nil {
			continue
		}
		if !exist[podPort] {
			ports = append(ports, core.ContainerPort{ContainerPort: int32(podPort), Protocol: protocol})
			exist[podPort] = true
		}
	}
	for _, port := range expose {
		_, podPort, protocol, err := c.parseContainerPort(port)
		if err != nil {
			continue
		}
		if !exist[podPort] {
			ports = append(ports, core.ContainerPort{ContainerPort: int32(podPort), Protocol: protocol})
			exist[podPort] = true
		}
	}
	return ports
}

func (c *v1v2Loader) addPorts(composePorts []string, expose []string, service *irtypes.Service) {
	exist := map[int]bool{}
	for _, port := range composePorts {
		servicePortNumber, podPortNumber, _, err := c.parseContainerPort(port)
		if err != nil {
			continue
		}
		if !exist[servicePortNumber] {
			// Forward the port on the k8s service to the k8s pod.
			podPort := networking.ServiceBackendPort{Number: int32(podPortNumber)}
			servicePort := networking.ServiceBackendPort{Number: int32(servicePortNumber)}
			service.AddPortForwarding(servicePort, podPort, "")
			exist[servicePortNumber] = true
		}
	}
	for _, port := range expose {
		servicePortNumber, podPortNumber, _, err := c.parseContainerPort(port)
		if err != nil {
			continue
		}
		if !exist[servicePortNumber] {
			// Forward the port on the k8s service to the k8s pod.
			podPort := networking.ServiceBackendPort{Number: int32(podPortNumber)}
			servicePort := networking.ServiceBackendPort{Number: int32(servicePortNumber)}
			service.AddPortForwarding(servicePort, podPort, "")
			exist[servicePortNumber] = true
		}
	}
}

func (*v1v2Loader) parseContainerPort(value string) (servicePort int, podPort int, protocol core.Protocol, err error) {
	protocol = core.ProtocolTCP
	if strings.Contains(value, "/") {
		parts := strings.Split(value, "/")
		value = parts[0]
		if strings.EqualFold(string(core.ProtocolUDP), parts[1]) {
			protocol = core.ProtocolUDP
		}
	}
	if !strings.Contains(value, ":") {
		// "3000"
		podPort, err = cast.ToIntE(value)
		if err != nil {
			logrus.Debugf("Failed to parse the port %s as an integer. Error: %q", value, err)
			return podPort, podPort, protocol, err
		}
		return podPort, podPort, protocol, nil
	}
	// Split up the ports and IP
	parts := strings.Split(value, ":")
	// "8000:8000"
	servicePortStr, podPortStr := parts[0], parts[1]
	if len(parts) == 3 {
		// "127.0.0.1:8001:8001"
		servicePortStr, podPortStr = parts[1], parts[2]
	} else if len(parts) > 3 {
		err := fmt.Errorf("failed to parse the port %s properly", value)
		return servicePort, podPort, protocol, err
	}
	servicePort, err = cast.ToIntE(servicePortStr)
	if err != nil {
		logrus.Debugf("Failed to parse the port %s as an integer. Error: %q", servicePortStr, err)
		return servicePort, servicePort, protocol, err
	}
	podPort, err = cast.ToIntE(podPortStr)
	if err != nil {
		logrus.Debugf("Failed to parse the port %s as an integer. Error: %q", podPortStr, err)
		return servicePort, podPort, protocol, err
	}
	return servicePort, podPort, protocol, nil
}

func getGroupAdd(group []string) ([]int64, error) {
	var groupAdd []int64
	for _, i := range group {
		j, err := cast.ToIntE(i)
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
