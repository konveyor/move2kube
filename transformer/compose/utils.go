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

package compose

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/opts"
	"github.com/konveyor/move2kube/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

const (
	defaultEnvFile        string = ".env"
	modeReadOnly          string = "ro"
	tmpFsPath             string = "tmpfs"
	defaultSecretBasePath string = "/var/secrets"
	envFile               string = "env_file"
	maxConfigMapSizeLimit int    = 1024 * 1024
	// cfgMapPrefix defines the prefix to be used for config maps
	cfgMapPrefix = "cfgmap"
)

/*
// IsV3 returns if the docker-compose yaml is version 3
func IsV3(path string) (bool, error) {
	dcfile := sourcetypes.DockerCompose{}
	if err := common.ReadYaml(path, &dcfile); err != nil {
		logrus.Debugf("Unable to read docker compose yaml at path %s Error: %q", path, err)
		return false, err
	}
	logrus.Debugf("Docker Compose version: %s", dcfile.Version)
	switch dcfile.Version {
	case "", "1", "1.0", "2", "2.0", "2.1":
		return false, nil
	case "3", "3.0", "3.1", "3.2", "3.3", "3.4", "3.5", "3.6", "3.7", "3.8":
		return true, nil
	default:
		err := fmt.Errorf("The compose file at path %s uses Docker Compose version %s which is not supported. Please use version 1, 2 or 3", path, dcfile.Version)
		logrus.Error(err)
		return false, err
	}
}
*/

func createConfigMapName(sourcePath string) string {
	cfgMapHashID := getHash([]byte(sourcePath))
	cfgName := fmt.Sprintf("%s-%d", cfgMapPrefix, cfgMapHashID)
	cfgName = common.MakeStringK8sServiceNameCompliant(cfgName)
	return cfgName
}

func createVolumeName(sourcePath string, serviceName string) string {
	hashID := getHash([]byte(sourcePath))
	volumeName := fmt.Sprintf("%s-%s-%d", common.VolumePrefix, serviceName, hashID)
	volumeName = common.MakeStringK8sServiceNameCompliant(volumeName)
	return volumeName
}

func loadDataAsConfigMap(filePath string, cfgName string) (irtypes.Storage, error) {
	storage := irtypes.Storage{
		Name:        cfgName,
		StorageType: irtypes.ConfigMapKind,
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		logrus.Warnf("Could not identify the type of config map artifact [%s]. Encountered [%s]", filePath, err)
	} else {
		if !fileInfo.IsDir() {
			content, err := os.ReadFile(filePath)
			if err != nil {
				logrus.Warnf("Could not read the secret file [%s]. Encountered [%s]", filePath, err)
			} else {
				if len(content) > maxConfigMapSizeLimit {
					return irtypes.Storage{}, fmt.Errorf("config map could not be created from file. Size limit of 1M exceeded")
				}
				storage.Content = map[string][]byte{cfgName: content}
			}
		} else {
			dataMap, err := getAllDirContentAsMap(filePath)
			if err != nil {
				logrus.Warnf("Could not read the config map directory [%s]. Encountered [%s]", filePath, err)
			} else {
				size := 0
				for _, data := range dataMap {
					size += len(data)
				}
				if size > maxConfigMapSizeLimit {
					return irtypes.Storage{}, fmt.Errorf("config map could not be created from file. Size limit of 1M exceeded")
				}
				storage.Content = dataMap
			}
		}
	}

	return storage, nil
}

func getAllDirContentAsMap(directoryPath string) (map[string][]byte, error) {
	fileList, err := os.ReadDir(directoryPath)
	if err != nil {
		return nil, err
	}
	dataMap := map[string][]byte{}
	count := 0
	for _, file := range fileList {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		logrus.Debugf("Reading file into the data map: [%s]", fileName)
		data, err := os.ReadFile(filepath.Join(directoryPath, fileName))
		if err != nil {
			logrus.Debugf("Unable to read file data : %s", fileName)
			continue
		}
		dataMap[fileName] = data
		count = count + 1
	}
	logrus.Debugf("Read %d files into the data map", count)
	return dataMap, nil
}

func getEnvironmentVariables(envFile string) map[string]string {
	result := map[string]string{}
	if len(envFile) > 0 {
		envs, err := opts.ParseEnvFile(envFile)
		if err != nil {
			logrus.Debugf("Environment file %s could not be read: %v", envFile, err)
		} else {
			for _, s := range envs {
				if !strings.Contains(s, "=") {
					logrus.Debugf("unexpected environment %q", s)
					continue
				}
				kv := strings.SplitN(s, "=", 2)
				result[kv[0]] = strings.Trim(kv[1], "\"")
			}
		}
	}
	//TODO: Check if any variable is mandatory and fill it with dummy value
	if common.IgnoreEnvironment {
		return result
	}
	env := os.Environ()
	for _, s := range env {
		if !strings.Contains(s, "=") {
			logrus.Debugf("unexpected environment %q", s)
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
