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

	log "github.com/sirupsen/logrus"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// ReuseDockerfileContainerizer uses its own containerization interface
type ReuseDockerfileContainerizer struct {
}

// GetContainerBuildStrategy returns the containerization build strategy
func (d *ReuseDockerfileContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.ReuseDockerFileContainerBuildTypeValue
}

// GetContainer returns the container for the service
func (d *ReuseDockerfileContainerizer) GetContainer(path string, serviceName string, imageName string, dockerfile string, context string) (irtypes.Container, error) {
	container := irtypes.NewContainer(d.GetContainerBuildStrategy(), imageName, true)
	_, err := os.Stat(filepath.Join(path, dockerfile))
	if os.IsNotExist(err) {
		log.Errorf("Unable to find docker file %s : %s", dockerfile, err)
		log.Errorf("Will assume the dockerfile will be copied and will proceed.")
	}
	if context == "" {
		context = "."
	}
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}
	dockerbuildscript, err := common.GetStringFromTemplate(scripts.Dockerbuild_sh, struct {
		Dockerfilename string
		ImageName      string
		Context        string
	}{
		Dockerfilename: dockerfile,
		ImageName:      imageName,
		Context:        context,
	})
	if err != nil {
		log.Warnf("Unable to translate template to string : %s", scripts.Dockerbuild_sh)
	} else {
		container.AddFile(filepath.Join(path, serviceName+"dockerbuild.sh"), dockerbuildscript)
	}

	return container, nil
}
