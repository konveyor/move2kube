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

package compose

import (
	"fmt"
	"hash/fnv"
	"os"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	log "github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

const (
	modeReadOnly          string = "ro"
	tmpFsPath             string = "tmpfs"
	defaultSecretBasePath string = "/var/secrets"
	envFile               string = "env_file"
)

/*
// IsV3 returns if the docker-compose yaml is version 3
func IsV3(path string) (bool, error) {
	dcfile := sourcetypes.DockerCompose{}
	if err := common.ReadYaml(path, &dcfile); err != nil {
		log.Debugf("Unable to read docker compose yaml at path %s Error: %q", path, err)
		return false, err
	}
	log.Debugf("Docker Compose version: %s", dcfile.Version)
	switch dcfile.Version {
	case "", "1", "1.0", "2", "2.0", "2.1":
		return false, nil
	case "3", "3.0", "3.1", "3.2", "3.3", "3.4", "3.5", "3.6", "3.7", "3.8":
		return true, nil
	default:
		err := fmt.Errorf("The compose file at path %s uses Docker Compose version %s which is not supported. Please use version 1, 2 or 3", path, dcfile.Version)
		log.Error(err)
		return false, err
	}
}
*/

func getEnvironmentVariables() map[string]string {
	result := map[string]string{}
	//TODO: Check if any variable is mandatory and fill it with dummy value
	if common.IgnoreEnvironment {
		return result
	}
	env := os.Environ()
	for _, s := range env {
		if !strings.Contains(s, "=") {
			log.Debugf("unexpected environment %q", s)
			continue
		}
		kv := strings.SplitN(s, "=", 2)
		result[kv[0]] = kv[1]
	}
	return result
}

func makeVolumesFromTmpFS(serviceName string, tfsList []string) ([]core.VolumeMount, []core.Volume) {
	vmList := []core.VolumeMount{}
	vList := []core.Volume{}

	for index, tfsObj := range tfsList {
		volumeName := fmt.Sprintf("%s-%s-%d", serviceName, tmpFsPath, index)

		vmList = append(vmList, core.VolumeMount{
			Name:      volumeName,
			MountPath: strings.Split(tfsObj, ":")[0],
		})

		vList = append(vList, core.Volume{
			Name: volumeName,
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{Medium: core.StorageMediumMemory},
			},
		})
	}

	return vmList, vList
}

func isPath(substring string) bool {
	return strings.Contains(substring, "/") || substring == "."
}

func getHash(data []byte) uint64 {
	hasher := fnv.New64a()
	hasher.Write(data)
	return hasher.Sum64()
}
