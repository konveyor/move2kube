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
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	modeReadOnly          string = "ro"
	tmpFsPath             string = "tmpfs"
	defaultSecretBasePath string = "/var/secrets"
	envFile               string = "env_file"
)

func makeVolumesFromTmpFS(serviceName string, tfsList []string) ([]corev1.VolumeMount, []corev1.Volume) {
	var vmList []corev1.VolumeMount
	var vList []corev1.Volume

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
