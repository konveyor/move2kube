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
	"strings"
	"time"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
	libcomposeyaml "github.com/docker/libcompose/yaml"
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

// v3Loader loads a v3 compose file
type v3Loader struct {
}

func removeNonExistentEnvFilesV3(path string, parsedComposeFile map[string]interface{}) map[string]interface{} {
	// Remove unresolvable env files, so that the parser does not throw error
	composeFileDir := filepath.Dir(path)
	if val, ok := parsedComposeFile["services"]; ok {
		if services, ok := val.(map[string]interface{}); ok {
			for serviceName, val := range services {
				if vals, ok := val.(map[string]interface{}); ok {
					if envfilesvals, ok := vals[envFile]; ok {
						// env_file can be a string or list of strings
						// https://docs.docker.com/compose/compose-file/#env_file
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
			}
		}
	}
	return parsedComposeFile
}

// parseV3 parses version 3 compose files
func parseV3(path string) (*types.Config, error) {
	fileData, err := os.ReadFile(path)
	if err != nil {
		err := fmt.Errorf("unable to load Compose file at path %s Error: %q", path, err)
		logrus.Debug(err)
		return nil, err
	}
	// Parse the Compose File
	parsedComposeFile, err := loader.ParseYAML(fileData)
	if err != nil {
		err := fmt.Errorf("unable to load Compose file at path %s Error: %q", path, err)
		logrus.Debug(err)
		return nil, err
	}
	parsedComposeFile = removeNonExistentEnvFilesV3(path, parsedComposeFile)
	// Config details
	configDetails := types.ConfigDetails{
		WorkingDir:  filepath.Dir(path),
		ConfigFiles: []types.ConfigFile{{Filename: path, Config: parsedComposeFile}},
		Environment: getEnvironmentVariables(),
	}
	config, err := loader.Load(configDetails)
	if err != nil {
		err := fmt.Errorf("unable to load Compose file at path %s Error: %q", path, err)
		logrus.Debug(err)
		return nil, err
	}
	return config, nil
}

// ConvertToIR loads an v3 compose file into IR
func (c *v3Loader) ConvertToIR(composefilepath string, serviceName string) (irtypes.IR, error) {
	logrus.Debugf("About to load configuration from docker compose file at path %s", composefilepath)
	config, err := parseV3(composefilepath)
	if err != nil {
		logrus.Warnf("Error while loading docker compose config : %s", err)
		return irtypes.IR{}, err
	}
	logrus.Debugf("About to start loading docker compose to intermediate rep")
	return c.convertToIR(filepath.Dir(composefilepath), *config, serviceName)
}

func (c *v3Loader) convertToIR(filedir string, composeObject types.Config, serviceName string) (irtypes.IR, error) {
	ir := irtypes.IR{
		Services: map[string]irtypes.Service{},
	}

	//Secret volumes transformed to IR
	ir.Storages = c.getSecretStorages(composeObject.Secrets)

	//ConfigMap volumes transformed to IR
	ir.Storages = append(ir.Storages, c.getConfigStorages(composeObject.Configs)...)

	for _, composeServiceConfig := range composeObject.Services {
		if composeServiceConfig.Name != serviceName {
			continue
		}
		name := common.NormalizeForMetadataName(composeServiceConfig.Name)
		serviceConfig := irtypes.NewServiceWithName(name)
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
		serviceContainer.WorkingDir = composeServiceConfig.WorkingDir
		serviceContainer.Command = composeServiceConfig.Entrypoint
		serviceContainer.Args = composeServiceConfig.Command
		serviceContainer.Stdin = composeServiceConfig.StdinOpen
		if composeServiceConfig.ContainerName == "" {
			composeServiceConfig.ContainerName = strings.ToLower(serviceConfig.Name)
		}
		serviceContainer.Name = common.NormalizeForMetadataName(composeServiceConfig.ContainerName)
		serviceContainer.TTY = composeServiceConfig.Tty
		serviceContainer.Ports = c.getPorts(composeServiceConfig.Ports, composeServiceConfig.Expose)
		c.addPorts(composeServiceConfig.Ports, composeServiceConfig.Expose, &serviceConfig)

		serviceConfig.Annotations = map[string]string(composeServiceConfig.Labels)
		serviceConfig.Labels = common.MergeStringMaps(composeServiceConfig.Labels, composeServiceConfig.Deploy.Labels)
		if composeServiceConfig.Hostname != "" {
			serviceConfig.Hostname = composeServiceConfig.Hostname
		}
		if composeServiceConfig.DomainName != "" {
			serviceConfig.Subdomain = composeServiceConfig.DomainName
		}
		if composeServiceConfig.Pid != "" {
			if composeServiceConfig.Pid == "host" {
				serviceConfig.SecurityContext.HostPID = true
			} else {
				logrus.Warnf("Ignoring PID key for service \"%v\". Invalid value \"%v\".", name, composeServiceConfig.Pid)
			}
		}
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
		//set capabilities if it is not empty
		if len(capsAdd) > 0 || len(capsDrop) > 0 {
			securityContext.Capabilities = &core.Capabilities{
				Add:  capsAdd,
				Drop: capsDrop,
			}
		}
		// update template only if securityContext is not empty
		if *securityContext != (core.SecurityContext{}) {
			serviceContainer.SecurityContext = securityContext
		}
		podSecurityContext := &core.PodSecurityContext{}
		if !cmp.Equal(*podSecurityContext, core.PodSecurityContext{}) {
			serviceConfig.SecurityContext = podSecurityContext
		}

		if composeServiceConfig.Deploy.Mode == "global" {
			serviceConfig.Daemon = true
		}

		serviceConfig.Networks = c.getNetworks(composeServiceConfig, composeObject)

		if (composeServiceConfig.Deploy.Resources != types.Resources{}) {
			if composeServiceConfig.Deploy.Resources.Limits != nil {
				resourceLimit := core.ResourceList{}
				memLimit := libcomposeyaml.MemStringorInt(composeServiceConfig.Deploy.Resources.Limits.MemoryBytes)
				if memLimit != 0 {
					resourceLimit[core.ResourceMemory] = *resource.NewQuantity(int64(memLimit), "RandomStringForFormat")
				}
				if composeServiceConfig.Deploy.Resources.Limits.NanoCPUs != "" {
					cpuLimit, err := cast.ToFloat64E(composeServiceConfig.Deploy.Resources.Limits.NanoCPUs)
					if err != nil {
						logrus.Warnf("Unable to convert cpu limits resources value : %s", err)
					}
					CPULimit := int64(cpuLimit * 1000)
					if CPULimit != 0 {
						resourceLimit[core.ResourceCPU] = *resource.NewMilliQuantity(CPULimit, resource.DecimalSI)
					}
				}
				serviceContainer.Resources.Limits = resourceLimit
			}
			if composeServiceConfig.Deploy.Resources.Reservations != nil {
				resourceRequests := core.ResourceList{}
				MemReservation := libcomposeyaml.MemStringorInt(composeServiceConfig.Deploy.Resources.Reservations.MemoryBytes)
				if MemReservation != 0 {
					resourceRequests[core.ResourceMemory] = *resource.NewQuantity(int64(MemReservation), "RandomStringForFormat")
				}
				if composeServiceConfig.Deploy.Resources.Reservations.NanoCPUs != "" {
					cpuReservation, err := cast.ToFloat64E(composeServiceConfig.Deploy.Resources.Reservations.NanoCPUs)
					if err != nil {
						logrus.Warnf("Unable to convert cpu limits reservation value : %s", err)
					}
					CPUReservation := int64(cpuReservation * 1000)
					if CPUReservation != 0 {
						resourceRequests[core.ResourceCPU] = *resource.NewMilliQuantity(CPUReservation, resource.DecimalSI)
					}
				}
				serviceContainer.Resources.Requests = resourceRequests
			}
		}

		// HealthCheck
		if composeServiceConfig.HealthCheck != nil && !composeServiceConfig.HealthCheck.Disable {
			probe, err := c.getHealthCheck(*composeServiceConfig.HealthCheck)
			if err != nil {
				logrus.Warnf("Unable to parse health check : %s", err)
			} else {
				serviceContainer.LivenessProbe = &probe
			}
		}
		restart := composeServiceConfig.Restart
		if composeServiceConfig.Deploy.RestartPolicy != nil {
			restart = composeServiceConfig.Deploy.RestartPolicy.Condition
		}
		if restart == "unless-stopped" {
			logrus.Warnf("Restart policy 'unless-stopped' in service %s is not supported, convert it to 'always'", name)
			serviceConfig.RestartPolicy = core.RestartPolicyAlways
		}
		// replicas:
		if composeServiceConfig.Deploy.Replicas != nil {
			serviceConfig.Replicas = int(*composeServiceConfig.Deploy.Replicas)
		}
		serviceContainer.Env = c.getEnvs(composeServiceConfig)

		vml, vl := makeVolumesFromTmpFS(name, composeServiceConfig.Tmpfs)
		for _, v := range vl {
			serviceConfig.AddVolume(v)
		}
		serviceContainer.VolumeMounts = append(serviceContainer.VolumeMounts, vml...)

		for _, secret := range composeServiceConfig.Secrets {
			target := filepath.Join(defaultSecretBasePath, secret.Source)
			src := secret.Source
			if secret.Target != "" {
				tokens := strings.Split(secret.Source, "/")
				var prefix string
				if !strings.HasPrefix(secret.Target, "/") {
					prefix = defaultSecretBasePath + "/"
				}
				if tokens[len(tokens)-1] == secret.Target {
					target = prefix + secret.Source
				} else {
					target = prefix + strings.TrimSuffix(secret.Target, "/"+tokens[len(tokens)-1])
				}
				src = tokens[len(tokens)-1]
			}

			vSrc := core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: secret.Source,
					Items: []core.KeyToPath{{
						Key:  secret.Source,
						Path: src,
					}},
				},
			}

			if secret.Mode != nil {
				mode := cast.ToInt32(*secret.Mode)
				vSrc.Secret.DefaultMode = &mode
			}

			serviceConfig.AddVolume(core.Volume{
				Name:         secret.Source,
				VolumeSource: vSrc,
			})

			serviceContainer.VolumeMounts = append(serviceContainer.VolumeMounts, core.VolumeMount{
				Name:      secret.Source,
				MountPath: target,
			})
		}

		for _, c := range composeServiceConfig.Configs {
			target := c.Target
			if target == "" {
				target = "/" + c.Source
			}
			vSrc := core.ConfigMapVolumeSource{}
			vSrc.Name = common.MakeFileNameCompliant(c.Source)
			if o, ok := composeObject.Configs[c.Source]; ok {
				if o.External.External {
					logrus.Errorf("Config metadata %s has an external source", c.Source)
				} else {
					srcBaseName := filepath.Base(o.File)
					vSrc.Items = []core.KeyToPath{{Key: srcBaseName, Path: filepath.Base(target)}}
					if c.Mode != nil {
						signedMode := int32(*c.Mode)
						vSrc.DefaultMode = &signedMode
					}
				}
			} else {
				logrus.Errorf("Unable to find configmap object for %s", vSrc.Name)
			}
			serviceConfig.AddVolume(core.Volume{
				Name:         vSrc.Name,
				VolumeSource: core.VolumeSource{ConfigMap: &vSrc},
			})

			serviceContainer.VolumeMounts = append(serviceContainer.VolumeMounts,
				core.VolumeMount{
					Name:      vSrc.Name,
					MountPath: target,
					SubPath:   filepath.Base(target),
				})
		}

		for _, vol := range composeServiceConfig.Volumes {
			if isPath(vol.Source) {
				hPath := vol.Source
				if !filepath.IsAbs(vol.Source) {
					hPath, err := filepath.Abs(vol.Source)
					if err != nil {
						logrus.Debugf("Could not create an absolute path for [%s]", hPath)
					}
				}
				// Generate a hash Id for the given source file path to be mounted.
				hashID := getHash([]byte(hPath))
				volumeName := fmt.Sprintf("%s%d", common.VolumePrefix, hashID)
				serviceContainer.VolumeMounts = append(serviceContainer.VolumeMounts, core.VolumeMount{
					Name:      volumeName,
					MountPath: vol.Target,
				})

				serviceConfig.AddVolume(core.Volume{
					Name: volumeName,
					VolumeSource: core.VolumeSource{
						HostPath: &core.HostPathVolumeSource{Path: vol.Source},
					},
				})
			} else {
				volumeName := vol.Source
				if volumeName == "" {
					hashID := getHash([]byte(vol.Target))
					volumeName = fmt.Sprintf("%s%d", common.VolumePrefix, hashID)
				}
				serviceContainer.VolumeMounts = append(serviceContainer.VolumeMounts, core.VolumeMount{
					Name:      volumeName,
					MountPath: vol.Target,
				})

				serviceConfig.AddVolume(core.Volume{
					Name: volumeName,
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: volumeName,
						},
					},
				})
				storageObj := irtypes.Storage{StorageType: irtypes.PVCKind, Name: volumeName, Content: nil}
				ir.AddStorage(storageObj)
			}
		}

		serviceConfig.Containers = []core.Container{serviceContainer}
		ir.Services[name] = serviceConfig
	}

	return ir, nil
}

func (c *v3Loader) getSecretStorages(secrets map[string]types.SecretConfig) []irtypes.Storage {
	storages := make([]irtypes.Storage, len(secrets))
	for secretName, secretObj := range secrets {
		storage := irtypes.Storage{
			Name:        secretName,
			StorageType: irtypes.SecretKind,
		}

		if !secretObj.External.External {
			content, err := os.ReadFile(secretObj.File)
			if err != nil {
				logrus.Warnf("Could not read the secret file [%s]", secretObj.File)
			} else {
				storage.Content = map[string][]byte{secretName: content}
			}
		}

		storages = append(storages, storage)
	}

	return storages
}

func (c *v3Loader) getConfigStorages(configs map[string]types.ConfigObjConfig) []irtypes.Storage {
	Storages := make([]irtypes.Storage, len(configs))

	for cfgName, cfgObj := range configs {
		storage := irtypes.Storage{
			Name:        cfgName,
			StorageType: irtypes.ConfigMapKind,
		}

		if !cfgObj.External.External {
			fileInfo, err := os.Stat(cfgObj.File)
			if err != nil {
				logrus.Warnf("Could not identify the type of secret artifact [%s]. Encountered [%s]", cfgObj.File, err)
			} else {
				if !fileInfo.IsDir() {
					content, err := os.ReadFile(cfgObj.File)
					if err != nil {
						logrus.Warnf("Could not read the secret file [%s]. Encountered [%s]", cfgObj.File, err)
					} else {
						storage.Content = map[string][]byte{cfgName: content}
					}
				} else {
					dataMap, err := c.getAllDirContentAsMap(cfgObj.File)
					if err != nil {
						logrus.Warnf("Could not read the secret directory [%s]. Encountered [%s]", cfgObj.File, err)
					} else {
						storage.Content = dataMap
					}
				}
			}
		}
		Storages = append(Storages, storage)
	}

	return Storages
}

func (*v3Loader) getPorts(ports []types.ServicePortConfig, expose []string) []core.ContainerPort {
	containerPorts := []core.ContainerPort{}
	exist := map[string]bool{}
	for _, port := range ports {
		proto := core.ProtocolTCP
		if strings.EqualFold(string(core.ProtocolUDP), port.Protocol) {
			proto = core.ProtocolUDP
		}
		// Add the port to the k8s pod.
		containerPorts = append(containerPorts, core.ContainerPort{
			ContainerPort: int32(port.Target),
			Protocol:      proto,
		})
		exist[cast.ToString(port.Target)] = true
	}
	for _, port := range expose {
		portValue := port
		protocol := core.ProtocolTCP
		if strings.Contains(portValue, "/") {
			splits := strings.Split(port, "/")
			portValue = splits[0]
			protocol = core.Protocol(strings.ToUpper(splits[1]))
		}
		if exist[portValue] {
			continue
		}
		// Add the port to the k8s pod.
		containerPorts = append(containerPorts, core.ContainerPort{
			ContainerPort: cast.ToInt32(portValue),
			Protocol:      protocol,
		})
	}

	return containerPorts
}

func (*v3Loader) addPorts(ports []types.ServicePortConfig, expose []string, service *irtypes.Service) {
	exist := map[string]bool{}
	for _, port := range ports {
		// Forward the port on the k8s service to the k8s pod.
		podPort := networking.ServiceBackendPort{
			Number: int32(port.Target),
		}
		servicePort := networking.ServiceBackendPort{
			Number: int32(port.Published),
		}
		service.AddPortForwarding(servicePort, podPort, "")
		exist[cast.ToString(port.Target)] = true
	}
	for _, port := range expose {
		portValue := port
		if strings.Contains(portValue, "/") {
			splits := strings.Split(port, "/")
			portValue = splits[0]
		}
		if exist[portValue] {
			continue
		}
		// Forward the port on the k8s service to the k8s pod.
		portNumber := cast.ToInt32(portValue)
		podPort := networking.ServiceBackendPort{
			Number: portNumber,
		}
		servicePort := networking.ServiceBackendPort{
			Number: portNumber,
		}
		service.AddPortForwarding(servicePort, podPort, "")
	}
}

func (c *v3Loader) getNetworks(composeServiceConfig types.ServiceConfig, composeObject types.Config) (networks []string) {
	networks = []string{}
	for key := range composeServiceConfig.Networks {
		netName := composeObject.Networks[key].Name
		if netName == "" {
			netName = key
		}
		networks = append(networks, netName)
	}
	return networks
}

func (c *v3Loader) getHealthCheck(composeHealthCheck types.HealthCheckConfig) (core.Probe, error) {
	probe := core.Probe{}

	if len(composeHealthCheck.Test) > 1 {
		probe.Handler = core.Handler{
			Exec: &core.ExecAction{
				// docker/cli adds "CMD-SHELL" to the struct, hence we remove the first element of composeHealthCheck.Test
				Command: composeHealthCheck.Test[1:],
			},
		}
	} else {
		logrus.Warnf("Could not find command to execute in probe : %s", composeHealthCheck.Test)
	}
	if composeHealthCheck.Timeout != nil {
		parse, err := time.ParseDuration(composeHealthCheck.Timeout.String())
		if err != nil {
			return probe, errors.Wrap(err, "unable to parse health check timeout variable")
		}
		probe.TimeoutSeconds = int32(parse.Seconds())
	}
	if composeHealthCheck.Interval != nil {
		parse, err := time.ParseDuration(composeHealthCheck.Interval.String())
		if err != nil {
			return probe, errors.Wrap(err, "unable to parse health check interval variable")
		}
		probe.PeriodSeconds = int32(parse.Seconds())
	}
	if composeHealthCheck.Retries != nil {
		probe.FailureThreshold = int32(*composeHealthCheck.Retries)
	}
	if composeHealthCheck.StartPeriod != nil {
		parse, err := time.ParseDuration(composeHealthCheck.StartPeriod.String())
		if err != nil {
			return probe, errors.Wrap(err, "unable to parse health check startPeriod variable")
		}
		probe.InitialDelaySeconds = int32(parse.Seconds())
	}

	return probe, nil
}

func (c *v3Loader) getEnvs(composeServiceConfig types.ServiceConfig) (envs []core.EnvVar) {
	for name, value := range composeServiceConfig.Environment {
		var env core.EnvVar
		if value != nil {
			env = core.EnvVar{Name: name, Value: *value}
		} else {
			env = core.EnvVar{Name: name, Value: "unknown"}
		}
		envs = append(envs, env)
	}
	return envs
}

func (c *v3Loader) getAllDirContentAsMap(directoryPath string) (map[string][]byte, error) {
	fileList, err := os.ReadDir(directoryPath)
	if err != nil {
		return nil, err
	}
	dataMap := map[string][]byte{}
	count := 0
	for _, file := range fileList {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		logrus.Debugf("Reading file into the data map: [%s]", fileName)
		data, err := os.ReadFile(filepath.Join(directoryPath, fileName))
		if err != nil {
			logrus.Debugf("Unable to read file data : %s", fileName)
			continue
		}
		dataMap[fileName] = data
		count = count + 1
	}
	logrus.Debugf("Read %d files into the data map", count)
	return dataMap, nil
}
