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

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	modeReadOnly          string = "ro"
	tmpFsPath             string = "tmpfs"
	defaultSecretBasePath string = "/var/secrets"
	envFile               string = "env_file"
)

func checkForDockerfile(path string) bool {
	finfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("There is no file at path %s Error: %q", path, err)
			return false
		}
		log.Errorf("There was an error accessing the file at path %s Error: %q", path, err)
		return false
	}
	if finfo.IsDir() {
		log.Errorf("The path %s points to a directory. Expected a Dockerfile.", path)
		return false
	}
	return true
}

func makeVolumesFromTmpFS(serviceName string, tfsList []string) ([]corev1.VolumeMount, []corev1.Volume) {
	vmList := []corev1.VolumeMount{}
	vList := []corev1.Volume{}

	for index, tfsObj := range tfsList {
		volumeName := fmt.Sprintf("%s-%s-%d", serviceName, tmpFsPath, index)

		vmList = append(vmList, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: strings.Split(tfsObj, ":")[0],
		})

		vList = append(vList, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumMemory},
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
