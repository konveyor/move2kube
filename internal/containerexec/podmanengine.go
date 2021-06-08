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

package containerexec

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
)

type podmanEngine struct {
	availableImages map[string]bool
}

func newPodmanEngine() *podmanEngine {
	return &podmanEngine{
		availableImages: map[string]bool{},
	}
}

// InspectImage returns inspect output for an image using Podman
func (e *podmanEngine) InspectImage(image string) (t types.ImageInspect, err error) {
	inspectcmd := exec.Command("podman", "inspect", image)
	logrus.Debugf("Inspecting image %s", image)
	output, err := inspectcmd.CombinedOutput()
	if err != nil {
		logrus.Debugf("Unable to inspect image %s : %s, %s", image, err, output)
		return t, err
	}
	t = types.ImageInspect{}
	err = json.Unmarshal(output, &t)
	if err != nil {
		logrus.Debugf("Error in unmarshalling json %s: %s.", output, err)
	}
	return t, err
}

func (e *podmanEngine) pullImage(image string) bool {
	if a, ok := e.availableImages[image]; ok {
		return a
	}
	pullcmd := exec.Command("podman", "pull", image)
	logrus.Debugf("Pulling image %s", image)
	output, err := pullcmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("Error while pulling builder %s : %s : %s", image, err, output)
		e.availableImages[image] = false
		return false
	}
	e.availableImages[image] = true
	return true
}

// RunContainer executes a container using podman
func (e *podmanEngine) RunContainer(image string, cmd string, volsrc string, voldest string) (output string, containerStarted bool, err error) {
	if !e.pullImage(image) {
		logrus.Debugf("Unable to pull image using podman : %s", image)
		return "", false, fmt.Errorf("unable to pull image")
	}
	args := []string{"run", "--rm"}
	if volsrc != "" && voldest != "" {
		args = append(args, "-v", volsrc+":"+voldest)
	}
	args = append(args, image)
	if cmd != "" {
		args = append(args, cmd)
	}
	detectcmd := exec.Command("podman", args...)
	logrus.Debugf("Running detect on image %s", image)
	o, err := detectcmd.CombinedOutput()
	if err != nil {
		logrus.Debugf("Detect failed %s : %s : %s", image, err, output)
		return string(o), false, err
	}
	return string(o), true, nil
}
