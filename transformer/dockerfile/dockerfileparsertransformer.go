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

package dockerfile

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	dockerparser "github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	core "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/networking"
)

// DockerfileParser implements Transformer interface
type DockerfileParser struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *DockerfileParser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *DockerfileParser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *DockerfileParser) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *DockerfileParser) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	createdArtifacts := []transformertypes.Artifact{}
	processedImages := map[string]bool{}
	for _, newArtifact := range newArtifacts {
		serviceConfig := artifacts.ServiceConfig{}
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &serviceConfig); err != nil {
			logrus.Errorf("unable to load the service config from the artifact %+v . Error: %q", newArtifact, err)
		}
		if serviceConfig.ServiceName == "" {
			serviceConfig.ServiceName = common.MakeStringK8sServiceNameCompliant(newArtifact.Name)
		}
		imageName := artifacts.ImageName{}
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Errorf("unable to load the imagename config from the artifact %+v . Error: %q", newArtifact, err)
		}
		if imageName.ImageName == "" {
			imageName.ImageName = common.MakeStringContainerImageNameCompliant(newArtifact.Name)
		}
		ir := irtypes.NewIR()
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			ir = irtypes.NewIR()
		}
		if processedImages[imageName.ImageName] {
			continue
		}
		processedImages[imageName.ImageName] = true
		if paths, ok := newArtifact.Paths[artifacts.DockerfilePathType]; ok && len(paths) > 0 {
			serviceFsPath := filepath.Dir(paths[0])
			if serviceFsPaths, ok := newArtifact.Paths[artifacts.ServiceDirPathType]; ok && len(serviceFsPaths) > 0 {
				serviceFsPath = serviceFsPaths[0]
			}
			contextPath := filepath.Dir(paths[0])
			if contextPaths, ok := newArtifact.Paths[artifacts.DockerfileContextPathType]; ok && len(contextPaths) > 0 {
				contextPath = contextPaths[0]
			}
			createdArtifact, err := t.getIRFromDockerfile(paths[0], contextPath, imageName.ImageName, serviceConfig.ServiceName, serviceFsPath, ir)
			if err != nil {
				logrus.Errorf("failed to convert the Dockerfile to IR. Error: %q", err)
				continue
			}
			createdArtifacts = append(createdArtifacts, createdArtifact)
		}
	}
	return nil, createdArtifacts, nil
}

func (t *DockerfileParser) getIRFromDockerfile(dockerfilepath, contextPath, imageName, serviceName, serviceFsPath string, ir irtypes.IR) (transformertypes.Artifact, error) {
	df, err := t.getDockerFileAST(dockerfilepath)
	if err != nil {
		logrus.Errorf("Unable to parse dockerfile : %s", err)
		return transformertypes.Artifact{}, err
	}
	ir.Name = t.Env.GetProjectName()
	container := irtypes.NewContainer()
	for _, dfchild := range df.AST.Children {
		if strings.EqualFold(dfchild.Value, "EXPOSE") {
			for {
				dfchild = dfchild.Next
				if dfchild == nil {
					break
				}
				p, err := cast.ToIntE(dfchild.Value)
				if err != nil {
					logrus.Errorf("Unable to parse port %s as int in %s", dfchild.Value, dockerfilepath)
					continue
				}
				container.AddExposedPort(int32(p))
			}
		}
	}
	container.Build.ContainerBuildType = irtypes.DockerfileContainerBuildType
	container.Build.ContextPath = contextPath

	t111 := map[irtypes.ContainerBuildArtifactTypeValue][]string{
		irtypes.DockerfileContainerBuildArtifactTypeValue: {dockerfilepath},
	}
	currEnvOutputDir := t.Env.GetEnvironmentOutput()
	logrus.Debugf("making Dockerfile paths relative to env output dir: '%s'", currEnvOutputDir)
	if relDockerfilePath, err := filepath.Rel(currEnvOutputDir, dockerfilepath); err == nil {
		t111[irtypes.RelDockerfileContainerBuildArtifactTypeValue] = []string{relDockerfilePath}
	} else {
		logrus.Errorf("failed to make the Dockerfile path '%s' relative to the env output dir '%s' . Error: %q", dockerfilepath, currEnvOutputDir, err)
	}
	if relDockerfileContextPath, err := filepath.Rel(currEnvOutputDir, contextPath); err == nil {
		t111[irtypes.RelDockerfileContextContainerBuildArtifactTypeValue] = []string{relDockerfileContextPath}
	} else {
		logrus.Errorf("failed to make the Dockerfile context path '%s' relative to the env output dir '%s' . Error: %q", contextPath, currEnvOutputDir, err)
	}
	container.Build.Artifacts = t111

	if len(container.ExposedPorts) == 0 {
		logrus.Warnf("Unable to find ports in Dockerfile : %s. Using default port %d", dockerfilepath, common.DefaultServicePort)
		container.AddExposedPort(common.DefaultServicePort)
	}
	ir.AddContainer(imageName, container)
	serviceContainer := core.Container{Name: serviceName}
	serviceContainer.Image = imageName
	irService := irtypes.NewServiceWithName(serviceName)
	serviceContainerPorts := []core.ContainerPort{}
	for _, port := range container.ExposedPorts {
		// Add the port to the k8s pod.
		serviceContainerPort := core.ContainerPort{ContainerPort: port}
		serviceContainerPorts = append(serviceContainerPorts, serviceContainerPort)
		// Forward the port on the k8s service to the k8s pod.
		podPort := networking.ServiceBackendPort{Number: port}
		servicePort := podPort
		irService.AddPortForwarding(servicePort, podPort, "")
	}
	serviceContainer.Ports = serviceContainerPorts
	irService.Containers = []core.Container{serviceContainer}
	if t.isWindowsContainer(df) {
		irService.Annotations = map[string]string{common.WindowsAnnotation: common.AnnotationLabelValue}
		irService.NodeSelector = map[string]string{"kubernetes.io/os": "windows"}
		irService.Tolerations = []core.Toleration{{
			Effect: core.TaintEffectNoSchedule,
			Key:    "os",
			Value:  "Windows",
		}}
	}
	ir.AddService(irService)
	return transformertypes.Artifact{
		Name: t.Env.GetProjectName(),
		Type: irtypes.IRArtifactType,
		Paths: map[transformertypes.PathType][]string{
			artifacts.ServiceDirPathType: {serviceFsPath},
		},
		Configs: map[transformertypes.ConfigType]interface{}{
			irtypes.IRConfigType: ir,
		}}, nil
}

func (t *DockerfileParser) getDockerFileAST(path string) (*dockerparser.Result, error) {
	f, err := os.Open(path)
	if err != nil {
		logrus.Debugf("Unable to open file %s : %s", path, err)
		return nil, err
	}
	defer f.Close()
	res, err := dockerparser.Parse(f)
	if err != nil {
		logrus.Debugf("Unable to parse file %s as Docker files : %s", path, err)
	}
	return res, err
}

func (t *DockerfileParser) isWindowsContainer(df *dockerparser.Result) bool {
	for _, dfchild := range df.AST.Children {
		if strings.EqualFold(dfchild.Value, "FROM") {
			imageNameNode := dfchild.Next
			if imageNameNode == nil {
				continue
			}
			for _, flag := range dfchild.Flags {
				flag = strings.TrimPrefix(flag, "--platform=")
				if strings.HasPrefix(flag, "windows") {
					return true
				}
			}
		}
	}
	return false
}
