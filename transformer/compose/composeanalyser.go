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
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	// ComposeServiceConfigType represents the Compose service config type
	ComposeServiceConfigType transformertypes.ConfigType = "ComposeService"
)

const (
	// composeFilePathType defines the source artifact type of Docker compose
	composeFilePathType transformertypes.PathType = "DockerCompose"
	// imageInfoPathType defines the source artifact type of image info
	imageInfoPathType transformertypes.PathType = "ImageInfo"
)

// ComposeAnalyser implements Transformer interface
type ComposeAnalyser struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// ComposeConfig stores the config for compose service
type ComposeConfig struct {
	ServiceName string `yaml:"serviceName,omitempty"`
}

// Init Initializes the transformer
func (t *ComposeAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the config
func (t *ComposeAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect detects docker compose files
func (t *ComposeAnalyser) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	yamlpaths, err := common.GetFilesByExt(dir, []string{".yaml", ".yml"})
	if err != nil {
		logrus.Errorf("Unable to fetch yaml files at path %s Error: %q", dir, err)
		return nil, err
	}
	imageMetadataPaths := map[string]string{}
	for _, path := range yamlpaths {
		im := collecttypes.ImageInfo{}
		if err := common.ReadMove2KubeYaml(path, &im); err != nil || im.Kind != string(collecttypes.ImageMetadataKind) {
			continue
		}
		for _, imagetag := range im.Spec.Tags {
			imageMetadataPaths[imagetag] = path
		}
	}
	services = map[string][]transformertypes.Artifact{}
	for _, path := range yamlpaths {
		currServices := t.getServicesFromComposeFile(path, imageMetadataPaths)
		services = plantypes.MergeServicesT(services, currServices)
	}
	logrus.Debugf("Docker compose services : %+v", services)
	return services, nil
}

// Transform transforms the artifacts
func (t *ComposeAnalyser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		var config ComposeConfig
		err := a.GetConfig(ComposeServiceConfigType, &config)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", config, err)
			continue
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		ir := irtypes.NewIR()
		for _, path := range a.Paths[composeFilePathType] {
			logrus.Debugf("File %s being loaded from compose service : %s", path, config.ServiceName)
			// Try v3 first and if it fails try v1v2
			if cir, errV3 := new(v3Loader).ConvertToIR(path, config.ServiceName); errV3 == nil {
				ir.Merge(cir)
				logrus.Debugf("compose v3 transformer returned %d services", len(ir.Services))
			} else if cir, errV1V2 := new(v1v2Loader).ConvertToIR(path, config.ServiceName); errV1V2 == nil {
				ir.Merge(cir)
				logrus.Debugf("compose v1v2 transformer returned %d services", len(ir.Services))
			} else {
				logrus.Errorf("Unable to parse the docker compose file at path %s Error V3: %q Error V1V2: %q", path, errV3, errV1V2)
			}
		}
		for _, path := range a.Paths[imageInfoPathType] {
			imgMD := collecttypes.ImageInfo{}
			if err := common.ReadMove2KubeYaml(path, &imgMD); err != nil {
				logrus.Errorf("Failed to read image info yaml at path %s Error: %q", path, err)
				continue
			}
			ir.AddContainer(sImageName.ImageName, newContainerFromImageInfo(imgMD))
		}
		if sImageName.ImageName == "" {
			if len(ir.ContainerImages) != 0 {
				for name := range ir.ContainerImages {
					sImageName.ImageName = name
					break
				}
			}
		} else {
			if len(ir.ContainerImages) > 1 {
				logrus.Errorf("Expected only one image in %s. Resetting only the first image name.", sConfig.ServiceName)
			}
			for name, ci := range ir.ContainerImages {
				delete(ir.ContainerImages, name)
				ir.ContainerImages[sImageName.ImageName] = ci
				break
			}
			for sn, s := range ir.Services {
				if len(s.Containers) > 1 {
					logrus.Errorf("Expected only one container. Finding more than one contaienr for service %s.", sn)
				}
				if len(s.Containers) != 0 {
					s.Containers[0].Image = sImageName.ImageName
					break
				}
			}
		}
		for name, s := range ir.Services {
			delete(ir.Services, name)
			ir.Services[sConfig.ServiceName] = s
			break
		}
		if len(ir.ContainerImages) > 0 {
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.SourcePathMappingType,
				DestPath: common.DefaultSourceDir,
			})
		}
		for name, ci := range ir.ContainerImages {
			contextPath := ci.Build.ContextPath
			dockerfilePath := filepath.Join(ci.Build.ContextPath, common.DefaultDockerfileName)
			if len(ci.Build.Artifacts) != 0 && len(ci.Build.Artifacts[irtypes.DockerfileContainerBuildArtifactTypeValue]) != 0 {
				dockerfilePath = ci.Build.Artifacts[irtypes.DockerfileContainerBuildArtifactTypeValue][0]
			}
			if contextPath == "" && dockerfilePath != common.DefaultDockerfileName {
				contextPath = filepath.Dir(dockerfilePath)
			}
			artifactsCreated = append(artifactsCreated, transformertypes.Artifact{
				Name: name,
				Type: artifacts.DockerfileArtifactType,
				Paths: map[transformertypes.PathType][]string{artifacts.DockerfilePathType: {dockerfilePath},
					artifacts.DockerfileContextPathType: {contextPath},
				},
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.ImageNameConfigType: artifacts.ImageName{
						ImageName: name,
					},
				},
			})
		}
		ac := transformertypes.Artifact{
			Name: t.Env.GetProjectName(),
			Type: irtypes.IRArtifactType,
			Configs: map[transformertypes.ConfigType]interface{}{
				irtypes.IRConfigType: ir,
			},
		}
		artifactsCreated = append(artifactsCreated, ac)
	}
	return pathMappings, artifactsCreated, nil
}

func (t *ComposeAnalyser) getService(composeFilePath string, serviceName string, serviceImage string, relContextPath string, relDockerfilePath string, imageMetadataPaths map[string]string) transformertypes.Artifact {
	ct := transformertypes.Artifact{
		Configs: map[transformertypes.ConfigType]interface{}{
			ComposeServiceConfigType: ComposeConfig{
				ServiceName: serviceName,
			}},
		Paths: map[transformertypes.PathType][]string{
			composeFilePathType: {
				composeFilePath,
			},
		},
	}
	if imagepath, ok := imageMetadataPaths[serviceImage]; ok {
		ct.Paths[imageInfoPathType] = common.MergeStringSlices(ct.Paths[imageInfoPathType], imagepath)
	}
	logrus.Debugf("Found a docker compose service : %s", serviceName)
	if relContextPath != "" {
		composeFileDir := filepath.Dir(composeFilePath)
		contextPath := filepath.Join(composeFileDir, relContextPath)
		if filepath.IsAbs(relContextPath) {
			contextPath = relContextPath // this happens with v1v2 parser
		}
		dockerfilePath := filepath.Join(contextPath, common.DefaultDockerfileName)
		if relDockerfilePath != "" {
			dockerfilePath = filepath.Join(contextPath, relDockerfilePath)
			if filepath.IsAbs(relDockerfilePath) {
				dockerfilePath = relDockerfilePath // this happens with v1v2 parser
			}
		}
		// Add reuse Dockerfile containerization option
		ct.Paths[artifacts.DockerfilePathType] = common.MergeStringSlices(ct.Paths[artifacts.DockerfilePathType], dockerfilePath)
		ct.Paths[artifacts.ServiceDirPathType] = common.MergeStringSlices(ct.Paths[artifacts.ServiceDirPathType], contextPath)
	}
	return ct
}

func (t *ComposeAnalyser) getServicesFromComposeFile(composeFilePath string, imageMetadataPaths map[string]string) map[string][]transformertypes.Artifact {
	services := map[string][]transformertypes.Artifact{}
	interpolate := true
	// Try v3 first and if it fails try v1v2
	if dc, errV3 := parseV3(composeFilePath); errV3 == nil {
		logrus.Debugf("Found a docker compose file at path %s", composeFilePath)
		for _, service := range dc.Services {
			services[service.Name] = []transformertypes.Artifact{t.getService(composeFilePath, service.Name, service.Image, service.Build.Context, service.Build.Dockerfile, imageMetadataPaths)}
		}
	} else {
		if strings.HasPrefix(errV3.Error(), "invalid interpolation format") {
			// With interpolation error v2 parser panics. This prevents the panic
			interpolate = false
		}
		if dc, errV1V2 := parseV2(composeFilePath, interpolate); errV1V2 == nil {
			logrus.Debugf("Found a docker compose file at path %s", composeFilePath)
			servicesMap := dc.ServiceConfigs.All()
			for serviceName, service := range servicesMap {
				services[serviceName] = []transformertypes.Artifact{t.getService(composeFilePath, serviceName, service.Image, service.Build.Context, service.Build.Dockerfile, imageMetadataPaths)}
			}
		} else {
			logrus.Debugf("Failed to parse file at path %s as a docker compose file. Error V3: %q Error V1V2: %q", composeFilePath, errV3, errV1V2)
		}
	}
	return services
}

// newContainerFromImageInfo creates a new container from image info
func newContainerFromImageInfo(i collecttypes.ImageInfo) irtypes.ContainerImage {
	c := irtypes.NewContainer()
	c.ExposedPorts = i.Spec.PortsToExpose
	c.UserID = i.Spec.UserID
	c.AccessedDirs = i.Spec.AccessedDirs
	return c
}
