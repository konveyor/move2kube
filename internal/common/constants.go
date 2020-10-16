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

package common

import (
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/types"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// DefaultProjectName represents the short app name
	DefaultProjectName string = "myproject"
	// DefaultPlanFile defines default name for plan file
	DefaultPlanFile string = types.AppNameShort + ".plan"
	// TempDirPrefix defines the prefix of the temp directory
	TempDirPrefix string = types.AppNameShort + "-"
	// AssetsDir defines the dir of the assets temp directory
	AssetsDir string = types.AppNameShort + "assets"
	// VolumePrefix defines the prefix to be used for volumes
	VolumePrefix string = "vol"
	// DefaultStorageClassName defines the default storage class to be used
	DefaultStorageClassName string = "default"
	// DefaultDirectoryPermission defines the default permission used when a directory is created
	DefaultDirectoryPermission os.FileMode = 0755
	// DefaultExecutablePermission defines the default permission used when an executable file is created
	DefaultExecutablePermission os.FileMode = 0744
	// DefaultFilePermission defines the default permission used when a non-executable file is created
	DefaultFilePermission os.FileMode = 0644
	// DefaultRegistryURL points to the default registry url that will be used
	DefaultRegistryURL string = "docker.io"
	// ImagePullSecretPrefix is the prefix that will be prepended to pull secret name
	ImagePullSecretPrefix string = "imagepullsecret"
	// QACacheFile defines the location of the QA cache file
	QACacheFile string = types.AppNameShort + "qacache.yaml"
	// DefaultClusterType defines the default cluster type chosen by plan
	DefaultClusterType string = "Kubernetes"
	// IgnoreFilename is the name of the file containing the ignore rules and exceptions
	IgnoreFilename string = "." + types.AppNameShort + "ignore"
)

var (
	// DefaultPVCSize stores the default PVC size
	DefaultPVCSize, _ = resource.ParseQuantity("100Mi")
	// IgnoreEnvironment indicates whether to ignore the current environment or not
	IgnoreEnvironment = false
	// TempPath defines where all app data get stored during execution
	TempPath = TempDirPrefix + "temp"
	// AssetsPath defines where all assets get stored during execution
	AssetsPath = filepath.Join(TempPath, AssetsDir)
)
