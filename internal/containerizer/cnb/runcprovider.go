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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containers/skopeo/cmd/skopeo/inspect"
	ocispec "github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"

	"github.com/konveyor/move2kube/internal/common"
)

var (
	// CNBContainersPath defines the location of the cnb container cache used by runc
	cnbContainersPath string = filepath.Join(common.AssetsPath, "cnb")
	runcImagesPath           = filepath.Join(cnbContainersPath, "images")
	runcBundlesPath          = filepath.Join(cnbContainersPath, "bundles")
)

type runcProvider struct {
}

func (r *runcProvider) getAllBuildpacks(builders []string) (map[string][]string, error) { //[Containerization target option value] buildpacks
	buildpacks := map[string][]string{}
	if !r.isAvailable() {
		return buildpacks, errors.New("Runc not supported in this instance")
	}
	log.Debugf("Getting data of all builders %s", builders)
	for _, builder := range builders {
		cmd := exec.Command("skopeo", "inspect", "docker://"+string(builder))
		output, err := cmd.CombinedOutput()
		log.Debugf("Builder %s data :%s", builder, output)
		if err != nil {
			log.Warnf("Error while getting supported buildpacks for builder %s : %s", builder, err)
			continue
		}
		sio := inspect.Output{}
		err = json.Unmarshal(output, &sio)
		if err != nil {
			log.Warnf("Unable to seriablize inspect output for builder %s : %s", builder, err)
			continue
		}
		o, found := sio.Labels[orderLabel]
		if !found {
			log.Warnf("%s missing in builder %s : %s", orderLabel, builder, err)
			continue
		}
		buildpacks[builder] = getBuildersFromLabel(o)
	}
	return buildpacks, nil
}

func (r *runcProvider) isAvailable() bool {
	_, err := exec.LookPath("runc")
	if err != nil {
		log.Debugf("Unable to find runc, ignoring runc based cnb check : %s", err)
		return false
	}
	_, err = exec.LookPath("skopeo")
	if err != nil {
		log.Debugf("Unable to find skopeo, ignoring runc based cnb check : %s", err)
		return false
	}
	_, err = exec.LookPath("umoci")
	if err != nil {
		log.Debugf("Unable to find umoci, ignoring runc based cnb check : %s", err)
		return false
	}
	return true
}

func (r *runcProvider) isBuilderAvailable(builder string) bool {
	if !r.isAvailable() {
		return false
	}
	r.init([]string{builder})
	image, _ := common.GetImageNameAndTag(builder)
	_, err := os.Stat(filepath.Join(runcBundlesPath, image))
	if os.IsNotExist(err) {
		log.Debugf("Unable to find pack builder oci bundle, ignoring builder : %s", err)
		return false
	}
	return true
}

func (r *runcProvider) isBuilderSupported(path string, builder string) (bool, error) {
	if !r.isBuilderAvailable(builder) {
		return false, fmt.Errorf("Runc Builder image not available : %s", builder)
	}
	image, _ := common.GetImageNameAndTag(builder)
	ociimagespec := ocispec.Spec{}
	configfilepath := filepath.Join(runcBundlesPath, image, "config.json")
	err := common.ReadJSON(configfilepath, &ociimagespec)
	if err != nil {
		log.Errorf("Unable to read config for image %s : %s", builder, err)
		return false, err
	}

	mount := ocispec.Mount{}
	mount.Source, _ = filepath.Abs(path)
	mount.Destination = "/workspace"
	mount.Type = "bind"
	mount.Options = []string{"rbind", "ro"}
	found := false
	for i, m := range ociimagespec.Mounts {
		if m.Destination == mount.Destination {
			mounts := ociimagespec.Mounts
			mounts[i] = mount
			ociimagespec.Mounts = mounts
			found = true
		}
	}
	if !found {
		ociimagespec.Mounts = append(ociimagespec.Mounts, mount)
	}
	ociimagespec.Process.Args = []string{"/cnb/lifecycle/detector"}
	ociimagespec.Process.Terminal = false
	err = common.WriteJSON(configfilepath, ociimagespec)
	if err != nil {
		log.Errorf("Unable to write config json %s : %s", configfilepath, err)
	}

	//TODO: Check if two instances of runc can be spawned by two processes with same container name without errors
	cmd := exec.Command("runc", "run", "cnbbuilder")
	cmd.Dir = filepath.Join(runcBundlesPath, image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debugf("Error while executing runc %+v at %s : %s, %s, %s", cmd, cmd.Dir, path, output, err)
		return false, err
	}

	if strings.Contains(string(output), "ERROR: No buildpack groups passed detection.") {
		log.Debugf("No compatible cnb for %s", path)
		return false, nil
	}
	return true, nil
}

func (r *runcProvider) init(builders []string) {
	if !r.isAvailable() {
		return
	}

	err := os.MkdirAll(runcImagesPath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Debugf("Unable to create cnb directory ignoring runc based cnb check : %s", err)
		return
	}

	err = os.MkdirAll(runcBundlesPath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Debugf("Unable to create cnb directory ignoring runc based cnb check : %s", err)
		return
	}

	for _, builder := range builders {
		image, tag := common.GetImageNameAndTag(builder)
		if _, err := os.Stat(filepath.Join(runcImagesPath, image)); !os.IsNotExist(err) {
			continue
		}
		skopeocmd := exec.Command("skopeo", "copy", "docker://"+builder, "oci:"+image+":"+tag)
		skopeocmd.Dir = runcImagesPath
		log.Debugf("Pulling %s", builder)
		output, err := skopeocmd.CombinedOutput()
		if err != nil {
			log.Debugf("Unable to copy image %s : %s, %s", image, err, output)
			continue
		} else {
			log.Debugf("Image pull done : %s", output)
		}
		fullbundlepath, err := filepath.Abs(filepath.Join(runcBundlesPath, image))
		if err != nil {
			log.Errorf("Unable to resolve full path of directory %s : %s", fullbundlepath, err)
		}
		umocicmd := exec.Command("umoci", "unpack", "--image", image+":"+tag, fullbundlepath)
		umocicmd.Dir = runcImagesPath
		log.Debugf("Creating OCI image %s", builder)
		output, err = umocicmd.CombinedOutput()
		if err != nil {
			log.Debugf("Unable to copy image %s : %s, %s", image, err, output)
			continue
		} else {
			log.Debugf("Image extract done : %s", output)
		}
	}
}
