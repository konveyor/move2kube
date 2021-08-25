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

package analysers

import (
	"os"
	"strconv"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	dockerparser "github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/sirupsen/logrus"
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

// BaseDirectoryDetect runs detect in base directory
func (t *DockerfileParser) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *DockerfileParser) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// Transform transforms the artifacts
func (t *DockerfileParser) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	nartifacts := []transformertypes.Artifact{}
	processedImages := map[string]bool{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.DockerfileForServiceArtifactType {
			continue
		}
		sConfig := artifacts.ServiceConfig{}
		err := a.GetConfig(artifacts.ServiceArtifactType, &sConfig)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sConfig, err)
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(a.Name)
		}
		if processedImages[sImageName.ImageName] {
			continue
		}
		processedImages[sImageName.ImageName] = true
		for _, path := range a.Paths[artifacts.DockerfilePathType] {
			na := t.getIRFromDockerfile(path, sImageName.ImageName, sConfig.ServiceName)
			if na != nil {
				nartifacts = append(nartifacts, *na)
			}
		}
	}
	return nil, nartifacts, nil
}

func (t *DockerfileParser) getIRFromDockerfile(dockerfilepath, imageName, serviceName string) *transformertypes.Artifact {
	df, err := t.getDockerFileAST(dockerfilepath)
	if err != nil {
		logrus.Errorf("Unable to parse dockerfile : %s", err)
		return nil
	}
	ir := irtypes.NewIR()
	ir.Name = t.Env.GetProjectName()
	container := irtypes.NewContainer()
	for _, dfchild := range df.AST.Children {
		if dfchild.Value == "expose" {
			for {
				dfchild = dfchild.Next
				if dfchild == nil {
					break
				}
				p, err := strconv.Atoi(dfchild.Value)
				if err != nil {
					logrus.Errorf("Unable to parse port %s as int in %s", dfchild.Value, dockerfilepath)
					continue
				}
				container.AddExposedPort(int32(p))
			}
		}
	}
	container.Build.ContextPath = dockerfilepath
	container.Build.ContainerBuildType = irtypes.DockerfileContainerBuildType
	if len(container.ExposedPorts) == 0 {
		logrus.Warnf("Unable to find ports in Dockerfile : %s. Using default port", dockerfilepath)
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
	ir.Services[serviceName] = irService
	return &transformertypes.Artifact{
		Name:     t.Env.GetProjectName(),
		Artifact: irtypes.IRArtifactType,
		Configs: map[string]interface{}{
			irtypes.IRConfigType: ir,
		}}
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
		if dfchild.Value == "from" {
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
