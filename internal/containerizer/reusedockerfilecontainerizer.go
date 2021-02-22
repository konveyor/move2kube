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
	"fmt"
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

	if len(service.ContainerizationOptions) == 0 {
		err := fmt.Errorf("Failed to reuse the Dockerfile. The service %s doesn't have any containerization target options", service.ServiceName)
		log.Debug(err)
		return container, err
	}

	dockerfilePath := service.ContainerizationOptions[0]
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) { // TODO: What about other types of errors?
		log.Errorf("Unable to find the Dockerfile at path %q Error: %q", dockerfilePath, err)
		log.Errorf("Will assume the dockerfile will be copied and will proceed.") // TODO: is this correct? shouldn't we return here?
	}

	dockerfileDir := filepath.Dir(dockerfilePath)
	buildScriptFilename := service.ServiceName + "-docker-build.sh"
	buildScriptPath := filepath.Join(dockerfileDir, buildScriptFilename)

	relContextPath := "."
	if sourceCodeDirs, ok := service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType]; ok {
		if len(sourceCodeDirs) > 0 {
			sourceCodeDir := sourceCodeDirs[0]
			log.Debugf("Using %q as the context path for Dockerfile at path %q", sourceCodeDir, dockerfilePath)
			newRelContextPath, err := filepath.Rel(dockerfileDir, sourceCodeDir)
			if err != nil {
				log.Errorf("Failed to make the context path %q relative to the directory containing the Dockerfile %q Error: %q", sourceCodeDir, dockerfileDir, err)
				return container, err
			}
			relContextPath = newRelContextPath
		}
	}

	dockerBuildScript, err := common.GetStringFromTemplate(scripts.Dockerbuild_sh, struct {
		Dockerfilename string
		ImageName      string
		Context        string
	}{
		Dockerfilename: filepath.Base(dockerfilePath),
		ImageName:      service.Image,
		Context:        relContextPath,
	})
	if err != nil {
		log.Warnf("Unable to translate template to string : %s", scripts.Dockerbuild_sh)
	}

	relBuildScriptPath, err := plan.GetRelativePath(buildScriptPath)
	if err != nil {
		log.Errorf("Failed to make the build script path %q relative to the root directory %q Error: %q", buildScriptPath, plan.Spec.Inputs.RootDir, err)
		return container, err
	}

	container.AddFile(relBuildScriptPath, dockerBuildScript)
	return container, nil
}
