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

package cnb

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type containerRuntimeProvider struct {
}

var (
	containerRuntime  = ""
	availableBuilders = make(map[string]bool)
)

func (r *containerRuntimeProvider) getAllBuildpacks(builders []string) (map[string][]string, error) { //[Containerization target option value] buildpacks
	buildpacks := map[string][]string{}
	containerRuntime, available := r.getContainerRuntime()
	if !available {
		return buildpacks, errors.New("Container runtime not supported in this instance")
	}
	log.Debugf("Getting data of all builders %s", builders)
	for _, builder := range builders {
		inspectcmd := exec.Command(containerRuntime, "inspect", "--format", `{{ index .Config.Labels "`+orderLabel+`"}}`, builder)
		log.Debugf("Inspecting image %s", builder)
		output, err := inspectcmd.CombinedOutput()
		if err != nil {
			log.Debugf("Unable to inspect image %s : %s, %s", builder, err, output)
			continue
		}
		buildpacks[builder] = getBuildersFromLabel(string(output))
	}

	return buildpacks, nil
}

func (r *containerRuntimeProvider) getContainerRuntime() (runtime string, available bool) {
	if containerRuntime == "" {
		detectcmd := exec.Command("podman", "run", "--rm", "hello-world")
		output, err := detectcmd.CombinedOutput()
		if err != nil {
			log.Debugf("Podman not supported : %s : %s", err, output)
			containerRuntime = "none"
			return containerRuntime, false
		}
		containerRuntime = "podman"
		return containerRuntime, true
	} else if containerRuntime == "none" {
		return containerRuntime, false
	}
	return containerRuntime, true
}

func (r *containerRuntimeProvider) isBuilderAvailable(builder string) bool {
	containerRuntime, available := r.getContainerRuntime()
	if !available {
		return false
	}
	if status, ok := availableBuilders[builder]; ok {
		return status
	}
	// Check if the image exists locally
	existcmd := exec.Command(containerRuntime, "images", "-q", builder)
	log.Debugf("Checking if the image %s exists locally", builder)
	output, err := existcmd.Output()
	if err != nil {
		log.Warnf("Error while checking if the builder %s exists locally. Error: %q Output: %q", builder, err, output)
		availableBuilders[builder] = false
		return false
	}
	if len(output) > 0 {
		// Found the image in the local machine, no need to pull.
		availableBuilders[builder] = true
		return true
	}

	pullcmd := exec.Command(containerRuntime, "pull", builder)
	log.Debugf("Pulling image %s", builder)
	output, err = pullcmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error while pulling builder %s : %s : %s", builder, err, output)
		availableBuilders[builder] = false
		return false
	}
	availableBuilders[builder] = true
	return true
}

func (r *containerRuntimeProvider) isBuilderSupported(path string, builder string) (bool, error) {
	if !r.isBuilderAvailable(builder) {
		return false, fmt.Errorf("Builder image not available : %s", builder)
	}
	containerRuntime, _ := r.getContainerRuntime()
	p, err := filepath.Abs(path)
	if err != nil {
		log.Warnf("Unable to resolve to absolute path : %s", err)
	}
	detectcmd := exec.Command(containerRuntime, "run", "--rm", "-v", p+":/workspace", builder, "/cnb/lifecycle/detector")
	log.Debugf("Running detect on image %s", builder)
	output, err := detectcmd.CombinedOutput()
	if err != nil {
		log.Debugf("Detect failed %s : %s : %s", builder, err, output)
		return false, nil
	}
	return true, nil
}
