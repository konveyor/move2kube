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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/opts"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

const (
	defaultEnvFile        string = ".env"
	modeReadOnly          string = "ro"
	modeReadWrite         string = "rw"
	tmpFsPath             string = "tmpfs"
	defaultSecretBasePath string = "/var/secrets"
	envFile               string = "env_file"
	maxConfigMapSizeLimit int    = 1024 * 1024
	// cfgMapPrefix defines the prefix to be used for config maps
	cfgMapPrefix   = "configmap"
	secretPrefix   = "secret"
	volQaPrefixKey = "move2kube.storage.type"
	ignoreOpt      = "Ignore"
	configMapOpt   = "ConfigMap"
	secretOpt      = "Secret"
	hostPathOpt    = "HostPath"
	pvcOpt         = "PVC"
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

func createSecretName(sourcePath string) string {
	cfgMapHashID := getHash([]byte(sourcePath))
	cfgName := fmt.Sprintf("%s-%d", secretPrefix, cfgMapHashID)
	cfgName = common.MakeStringK8sServiceNameCompliant(cfgName)
	return cfgName
}

func createVolumeName(sourcePath string, serviceName string) string {
	hashID := getHash([]byte(sourcePath))
	volumeName := fmt.Sprintf("%s-%s-%d", common.VolumePrefix, serviceName, hashID)
	volumeName = common.MakeStringK8sServiceNameCompliant(volumeName)
	return volumeName
}

func withinK8sConfigSizeLimit(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	if !fileInfo.IsDir() {
		if fileInfo.Size() > int64(maxConfigMapSizeLimit) {
			return false, fmt.Errorf("config map could not be created from file. Size limit of 1M exceeded")
		}
	} else {
		var totalSize int64 = 0
		filepath.Walk(filePath, filepath.WalkFunc(func(dir string, fileHandle os.FileInfo, err error) error {
			if !fileHandle.IsDir() {
				totalSize += fileHandle.Size()
				if totalSize > int64(maxConfigMapSizeLimit) {
					return io.EOF
				}
			}
			return nil
		}))
		if totalSize > int64(maxConfigMapSizeLimit) {
			return false, fmt.Errorf("config map could not be created from directory. Size limit of 1M exceeded")
		}
	}

	return true, nil
}

func applyVolumePolicy(filedir string, serviceName string, volSource string, volTarget string, volAccessMode string, storageMap map[string]bool) (*core.VolumeMount, *core.Volume, *irtypes.Storage, error) {
	volume := core.Volume{}
	volumeMount := core.VolumeMount{}
	storage := irtypes.Storage{}
	storageOpt := ""
	volumeName := ""
	hPath := ""
	if isPath(volSource) {
		hPath = volSource
		if filepath.IsAbs(hPath) {
			relPath, err := filepath.Rel(filedir, hPath)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("could not extract relative path for [%s] due to <%s>", hPath, err)
			}
			volSource = relPath
		} else {
			hPath = filepath.Join(filedir, volSource)
		}
		opt, err := getUserInputsOnStorageType(hPath, serviceName)
		if err != nil {
			return nil, nil, nil, err
		}
		storageOpt = opt
		// Generate a hash Id for the given source file path to be mounted.
		volumeName = createVolumeName(volSource, serviceName)
	} else {
		// Generate a hash Id for the given source file path to be mounted.
		hPath = volSource
		opt, err := getUserInputsOnStorageType(hPath, serviceName)
		if err != nil {
			return nil, nil, nil, err
		}
		storageOpt = opt
		if hPath == "" {
			hPath = volTarget
		}
		hashID := getHash([]byte(hPath))
		volumeName = fmt.Sprintf("%s%d", common.VolumePrefix, hashID)
	}
	logrus.Warnf("====> Storage option selected: [%s]", storageOpt)
	switch storageOpt {
	case secretOpt:
		secretName := createSecretName(volSource)
		if _, ok := storageMap[secretName]; !ok {
			st, err := createStorage(hPath, secretName, irtypes.SecretKind)
			if err != nil {
				return nil, nil, nil, err
			}
			storage = st
		}
		secretVolSrc := core.SecretVolumeSource{}
		secretVolSrc.SecretName = secretName
		volume = core.Volume{
			Name: volumeName,
			VolumeSource: core.VolumeSource{
				Secret: &secretVolSrc,
			},
		}
	case configMapOpt:
		configMapName := createConfigMapName(volSource)
		if _, ok := storageMap[configMapName]; !ok {
			st, err := createStorage(hPath, configMapName, irtypes.ConfigMapKind)
			if err != nil {
				return nil, nil, nil, err
			}
			storage = st
		}
		cfgMapVolSrc := core.ConfigMapVolumeSource{}
		cfgMapVolSrc.Name = configMapName
		volume = core.Volume{
			Name: volumeName,
			VolumeSource: core.VolumeSource{
				ConfigMap: &cfgMapVolSrc,
			},
		}
	case hostPathOpt:
		volume = core.Volume{
			Name: volumeName,
			VolumeSource: core.VolumeSource{
				HostPath: &core.HostPathVolumeSource{Path: volSource},
			},
		}
	case pvcOpt:
		accessMode := core.ReadWriteMany
		if volAccessMode == modeReadOnly {
			accessMode = core.ReadOnlyMany
		}
		storage = irtypes.Storage{StorageType: irtypes.PVCKind, Name: volumeName, Content: nil}
		storage.PersistentVolumeClaimSpec = core.PersistentVolumeClaimSpec{
			AccessModes: []core.PersistentVolumeAccessMode{accessMode},
		}
		volume = core.Volume{
			Name: volumeName,
			VolumeSource: core.VolumeSource{
				PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
					ClaimName: volumeName,
					ReadOnly:  volAccessMode == modeReadOnly,
				},
			},
		}
	case ignoreOpt:
		return nil, nil, nil, nil
	}
	volumeMount = core.VolumeMount{
		Name:      volumeName,
		ReadOnly:  volAccessMode == modeReadOnly,
		MountPath: volTarget,
	}
	return &volumeMount, &volume, &storage, nil
}

func getUserInputsOnStorageType(filePath string, serviceName string) (string, error) {
	selectedOption := ignoreOpt
	ignoreDataAnswer := "Ignore the data source"
	defAnswer := ignoreDataAnswer
	desc := "Select the storage type to create"
	hints := []string{"By default, no storage type will be created. Data source will be ignored"}
	volQaKey := common.JoinQASubKeys(volQaPrefixKey, `"`+serviceName+`"`, ".options")
	options := []string{pvcOpt, ignoreDataAnswer}
	if isPath(filePath) {
		isWithinLimits, err := withinK8sConfigSizeLimit(filePath)
		if err != nil {
			options = []string{pvcOpt, hostPathOpt, ignoreDataAnswer}
			logrus.Warnf("(getUserInputsOnStorageType) filePath could not be read: %s", filePath)
		} else {
			if isWithinLimits {
				defAnswer = secretOpt
				options = []string{configMapOpt, hostPathOpt, pvcOpt, ignoreDataAnswer, defAnswer}
				hints = []string{fmt.Sprintf("By default, %s will be created", defAnswer)}
				logrus.Warnf("(getUserInputsOnStorageType) artifacts within path within limits")
			} else {
				options = []string{hostPathOpt, pvcOpt, defAnswer}
				logrus.Warnf("(getUserInputsOnStorageType) artifacts outside path within limits")
			}
		}
	} else {
		logrus.Warnf("(getUserInputsOnStorageType) [%s] is not a path", filePath)
	}
	selectedOption = qaengine.FetchSelectAnswer(volQaKey, desc, hints, defAnswer, options, nil)
	if selectedOption == ignoreDataAnswer {
		selectedOption = ignoreOpt
		logrus.Warnf("User has ignored data in path [%s]. No storage type created", filePath)
	}

	return selectedOption, nil
}

func createStorage(filePath string, storageName string, storageType irtypes.StorageKindType) (irtypes.Storage, error) {
	storage := irtypes.Storage{
		Name:        storageName,
		StorageType: storageType,
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return irtypes.Storage{}, fmt.Errorf("could not identify the volume source path (%s) because <%s>", filePath, err)
	}
	if !fileInfo.IsDir() {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return irtypes.Storage{}, fmt.Errorf("could not read the file [%s]. Encountered [%s]", filePath, err)
		}
		storage.Content = map[string][]byte{storageName: content}
	} else {
		dataMap, err := getAllDirContentAsMap(filePath)
		if err != nil {
			return irtypes.Storage{}, fmt.Errorf("could not read the volume source directory [%s]. Encountered [%s]", filePath, err)
		}
		storage.Content = dataMap
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
