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
	// ScriptsDir defines the directory where the output scripts are placed
	ScriptsDir string = "scripts"
	// SourceDir defines the directory where the source files and folders are placed along with build scripts for each individual image
	SourceDir string = "source"
	// DeployDir defines the directory where the deployment artifacts are placed
	DeployDir string = "deploy"
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
	DefaultRegistryURL string = "quay.io"
	// ImagePullSecretPrefix is the prefix that will be prepended to pull secret name
	ImagePullSecretPrefix string = "imagepullsecret"
	// QACacheFile defines the location of the QA cache file
	QACacheFile string = types.AppNameShort + "qacache.yaml"
	// ConfigFile defines the location of the config file
	ConfigFile string = types.AppNameShort + "config.yaml"
	// DefaultClusterType defines the default cluster type chosen by plan
	DefaultClusterType string = "Kubernetes"
	// IgnoreFilename is the name of the file containing the ignore rules and exceptions
	IgnoreFilename string = "." + types.AppNameShort + "ignore"
	// ExposeSelector tag is used to annotate services that are externally exposed
	ExposeSelector string = types.GroupName + "/service.expose"
	// AnnotationLabelValue represents the value when an annotation is valid
	AnnotationLabelValue string = "true"
	// DefaultServicePort is the default port that will be added to a service.
	DefaultServicePort = 8080
	// TODOAnnotation is used to annotate with TODO tasks
	TODOAnnotation string = types.GroupName + "/todo."
)

const (
	// Delim is the delimiter used to separate key segments
	Delim = "."
	// Special is the special case indicator of the multi-select problems
	Special = "[]"
	// MatchAll is used to set the default for a set of keys. Example: move2kube.services.*.ports=[8080]
	MatchAll = "*"

	//Configuration keys

	d = Delim // for readability
	// BaseKey is the prefix for every key
	BaseKey = types.AppName
	//ConfigServicesKey represents Services Key
	ConfigServicesKey = BaseKey + d + "services"
	//ConfigStoragesKey represents Storages Key
	ConfigStoragesKey = BaseKey + d + "storages"
	//ConfigSourcesKey represents Sources Key
	ConfigSourcesKey = BaseKey + d + "sources"
	//ConfigTargetKey represents Target Key
	ConfigTargetKey = BaseKey + d + "target"
	//ConfigRepoKey represents Repo Key
	ConfigRepoKey = BaseKey + d + "repo"
	//ConfigContainerizationKeySegment represents Containerization Key segment
	ConfigContainerizationKeySegment = BaseKey + d + "containerization"
	//ConfigRepoKeysKey represents Repo Key
	ConfigRepoKeysKey = ConfigRepoKey + d + "keys"
	//ConfigRepoPubKey represents allow load of public key of repos Key
	ConfigRepoPubKey = ConfigRepoKeysKey + d + "pub"
	//ConfigRepoLoadPubDomainsKey represents allow load of public key per domain of repos Key
	ConfigRepoLoadPubDomainsKey = ConfigRepoPubKey + d + "domain"
	//ConfigRepoLoadPubKey represents allow load of public key of repos Key
	ConfigRepoLoadPubKey = ConfigRepoPubKey + d + "load"
	//ConfigRepoPrivKey represents allow load of private key of repos Key
	ConfigRepoPrivKey = ConfigRepoKeysKey + d + "priv"
	//ConfigRepoLoadPrivKey represents allow load of private key of repos Key
	ConfigRepoLoadPrivKey = ConfigRepoKeysKey + d + "load"
	//ConfigRepoKeyPathsKey represents paths of keyfiles
	ConfigRepoKeyPathsKey = ConfigRepoKeysKey + d + "paths"
	//ConfigSourceTypesKey represents source type Key
	ConfigSourceTypesKey = ConfigSourcesKey + d + "types"
	//ConfigIngressKey represents Ingress Key
	ConfigIngressKey = ConfigTargetKey + d + "ingress"
	//ConfigIngressHostKey represents Ingress host Key
	ConfigIngressHostKey = ConfigIngressKey + d + "host"
	//ConfigIngressTLSKey represents ingress tls Key
	ConfigIngressTLSKey = ConfigIngressKey + d + "tls"
	//ConfigTargetClusterTypeKey represents target cluster type key
	ConfigTargetClusterTypeKey = ConfigTargetKey + d + "clustertype"
	//ConfigImageRegistryKey represents image registry Key
	ConfigImageRegistryKey = ConfigTargetKey + d + "imageregistry"
	//ConfigImageRegistryURLKey represents image registry url Key
	ConfigImageRegistryURLKey = ConfigImageRegistryKey + d + "url"
	//ConfigImageRegistryNamespaceKey represents image registry namespace Key
	ConfigImageRegistryNamespaceKey = ConfigImageRegistryKey + d + "namespace"
	//ConfigImageRegistryLoginTypeKey represents image registry login type Key
	ConfigImageRegistryLoginTypeKey = ConfigImageRegistryKey + d + "logintype"
	//ConfigImageRegistryPullSecretKey represents image registry pull secret Key
	ConfigImageRegistryPullSecretKey = ConfigImageRegistryKey + d + "pullsecret"
	//ConfigImageRegistryUserNameKey represents image registry login Username Key
	ConfigImageRegistryUserNameKey = ConfigImageRegistryKey + d + "username"
	//ConfigImageRegistryPasswordKey represents image registry login Password Key
	ConfigImageRegistryPasswordKey = ConfigImageRegistryKey + d + "password"
	//ConfigStoragesPVCForHostPathKey represents key for PVC for Host Path
	ConfigStoragesPVCForHostPathKey = ConfigStoragesKey + d + "pvcforhostpath"
	//ConfigStoragesPerClaimStorageClassKey represents key for having different storage class for claim
	ConfigStoragesPerClaimStorageClassKey = ConfigStoragesKey + d + "perclaimstorageclass"
	//ConfigServicesNamesKey represents Storages Key
	ConfigServicesNamesKey = ConfigServicesKey + d + Special + d + "enable"
	//ConfigContainerizationTypesKey represents source type Key
	ConfigContainerizationTypesKey = ConfigContainerizationKeySegment + d + "types"
	//ConfigServicesExposeKey represents Services Expose Key
	ConfigServicesExposeKey = ConfigServicesKey + d + Special + d + "expose"
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
