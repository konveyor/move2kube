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

package containerizer

import (
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

// ReuseDockerfileContainerizer uses its own containerization interface
type ReuseDockerfileContainerizer struct {
}

// GetContainerBuildStrategy returns the containerization build strategy
func (d *ReuseDockerfileContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.ReuseDockerFileContainerBuildTypeValue
}

// GetContainer returns the container for the service
func (d *ReuseDockerfileContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	container := irtypes.NewContainer(d.GetContainerBuildStrategy(), service.Image, true)

	dockerfilePath := service.ContainerizationTargetOptions[0]
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		log.Errorf("Unable to find the Dockerfile at path %q Error: %q", dockerfilePath, err)
		log.Errorf("Will assume the dockerfile will be copied and will proceed.")
	}

	dockerfileDir := filepath.Dir(dockerfilePath)
	dockerfileName := filepath.Base(dockerfilePath)
	context := "."
	if sc, ok := service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType]; ok {
		if len(sc) > 0 {
			var err error
			rootDir := plan.Spec.Inputs.RootDir
			sourceCodeDir := sc[0]
			context, err = filepath.Rel(rootDir, sourceCodeDir)
			if err != nil {
				log.Errorf("Failed to make the context path %q relative to the root directory %q Error: %q", sourceCodeDir, rootDir, err)
				return container, err
			}
		}
	}

	dockerBuildScript, err := common.GetStringFromTemplate(scripts.Dockerbuild_sh, struct {
		Dockerfilename string
		ImageName      string
		Context        string
	}{
		Dockerfilename: dockerfileName,
		ImageName:      service.Image,
		Context:        context,
	})
	if err != nil {
		log.Warnf("Unable to translate template to string : %s", scripts.Dockerbuild_sh)
	}
	relOutputPath, err := filepath.Rel(plan.Spec.Inputs.RootDir, dockerfileDir)
	if err != nil {
		log.Errorf("Failed to make the Dockerfile directory %q relative to the root directory %q Error: %q", dockerfileDir, plan.Spec.Inputs.RootDir, err)
		return container, err
	}
	container.AddFile(filepath.Join(relOutputPath, service.ServiceName+"-docker-build.sh"), dockerBuildScript)

	return container, nil
}
