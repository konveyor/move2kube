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

package common

import (
	"os"
	"regexp"

	"github.com/konveyor/move2kube/types"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// DisableLocalExecutionFlag is the name of the flag that tells us whether to use allow execution of executables locally
	DisableLocalExecutionFlag = "disable-local-execution"
	// FailOnEmptyPlan is the name of the flag that lets the user fail when the plan is empty (zero services, zero default transformers).
	FailOnEmptyPlan = "fail-on-empty-plan"
)

const (
	// DefaultProjectName represents the short app name
	DefaultProjectName = "myproject"
	// VolumePrefix defines the prefix to be used for volumes
	VolumePrefix = "vol"
	// DefaultDirectoryPermission defines the default permission used when a directory is created
	DefaultDirectoryPermission os.FileMode = 0755
	// DefaultExecutablePermission defines the default permission used when an executable file is created
	DefaultExecutablePermission os.FileMode = 0744
	// DefaultFilePermission defines the default permission used when a non-executable file is created
	DefaultFilePermission os.FileMode = 0644
	// QACacheFile defines the location of the QA cache file
	QACacheFile = types.AppNameShort + "qacache.yaml"
	// ConfigFile defines the location of the config file
	ConfigFile = types.AppNameShort + "config.yaml"
	// IgnoreFilename is the name of the file containing the ignore rules and exceptions
	IgnoreFilename = "." + types.AppNameShort + "ignore"
	// WindowsAnnotation tag is used tag a service to run on windows nodes
	WindowsAnnotation = types.GroupName + "/containertype.windows"
	// AnnotationLabelValue represents the value when an annotation is valid
	AnnotationLabelValue = "true"
	// DefaultServicePort is the default port that will be added to a service.
	DefaultServicePort int32 = 8080
	// TODOAnnotation is used to annotate with TODO tasks
	TODOAnnotation = types.GroupName + "/todo."
	// ShExt is the extension of sh file
	ShExt = ".sh"
	// BatExt is the extension of bat file
	BatExt = ".bat"
	// RemoteSourcesFolder stores remote sources
	RemoteSourcesFolder = "m2ksources"
	// RemoteCustomizationsFolder stores remote customizations
	RemoteCustomizationsFolder = "m2kcustomizations"
	// RemoteOutputsFolder stores remote outputs
	RemoteOutputsFolder = "m2koutputs"
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
	//TransformerSelectorKey represents transformers selector Key
	TransformerSelectorKey = BaseKey + d + "transformerselector"
	//ConfigServicesKey represents Services Key
	ConfigServicesKey = BaseKey + d + "services"
	//ConfigStoragesKey represents Storages Key
	ConfigStoragesKey = BaseKey + d + "storages"
	//ConfigMinReplicasKey represents Ingress host Key
	ConfigMinReplicasKey = BaseKey + d + "minreplicas"
	//ConfigStatefulSetKey represents whether the IR should generate a StatefulSet
	ConfigStatefulSetKey = "statefulset"
	//ConfigPortsForServiceKeySegment represents the ports used for service
	ConfigPortsForServiceKeySegment = "ports"
	//ConfigPortForServiceKeySegment represents the port used for service
	ConfigPortForServiceKeySegment = "port"
	//ConfigMainPythonFileForServiceKeySegment represents the main file used for service
	ConfigMainPythonFileForServiceKeySegment = "pythonmainfile"
	//ConfigStartingPythonFileForServiceKeySegment represents the starting python file used for service
	ConfigStartingPythonFileForServiceKeySegment = "pythonstartingfile"
	//ConfigCsprojFileForServiceKeySegment represents the csproj file used for service
	ConfigCsprojFileForServiceKeySegment = "csprojfile"
	//ConfigPublishProfileForServiceKeySegment represents the publish profile used for service
	ConfigPublishProfileForServiceKeySegment = "publishprofile"
	//ConfigContainerizationOptionServiceKeySegment represents containerization option to use
	ConfigContainerizationOptionServiceKeySegment = "containerizationoption"
	//ConfigApacheConfFileForServiceKeySegment represents the conf file used for service
	ConfigApacheConfFileForServiceKeySegment = "apacheconfig"
	//ConfigSpawnContainersKey represents spwan containers option Key
	ConfigSpawnContainersKey = BaseKey + d + "spawncontainers"
	//ConfigTransformersKey represents transformers Key
	ConfigTransformersKey = BaseKey + d + "transformers"
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
	//ConfigTransformerTypesKey represents Transformers type Key
	ConfigTransformerTypesKey = ConfigTransformersKey + d + "types"
	//VolQaPrefixKey represents the storage QA
	VolQaPrefixKey = BaseKey + d + "storage.type"
	//IngressKey represents ingress keyword
	IngressKey = "ingress"
	// ConfigIngressClassNameKeySuffix represents the ingress class name
	ConfigIngressClassNameKeySuffix = IngressKey + d + "ingressclassname"
	//ConfigIngressHostKeySuffix represents Ingress host Key
	ConfigIngressHostKeySuffix = IngressKey + d + "host"
	//ConfigIngressTLSKeySuffix represents ingress tls Key
	ConfigIngressTLSKeySuffix = IngressKey + d + "tls"
	//ConfigTargetClusterTypeKey represents target cluster type key
	ConfigTargetClusterTypeKey = ConfigTargetKey + d + "clustertype"
	//ConfigImageRegistryKey represents image registry Key
	ConfigImageRegistryKey = ConfigTargetKey + d + "imageregistry"
	// ConfigCICDKey is for CICD related questions
	ConfigCICDKey = ConfigTargetKey + d + "cicd"
	// ConfigCICDTektonKey is for CICD Tekton pipelines
	ConfigCICDTektonKey = ConfigCICDKey + d + "tekton"
	// ConfigCICDTektonGitRepoSSHSecretNameKey is for Tekton git-clone ssh
	ConfigCICDTektonGitRepoSSHSecretNameKey = ConfigCICDTektonKey + d + "gitreposshsecret"
	// ConfigCICDTektonGitRepoBasicAuthSecretNameKey is for Tekton git-clone basic auth
	ConfigCICDTektonGitRepoBasicAuthSecretNameKey = ConfigCICDTektonKey + d + "gitrepobasicauthsecret"
	// ConfigCICDTektonRegistryPushSecretNameKey is for Tekton push image to registry credentials
	ConfigCICDTektonRegistryPushSecretNameKey = ConfigCICDTektonKey + d + "registrypushsecret"
	//ConfigTargetExistingVersionUpdate represents key which how to update versions
	ConfigTargetExistingVersionUpdate = ConfigTargetKey + d + "existingversionupdate"
	//ConfigImageRegistryURLKey represents image registry url Key
	ConfigImageRegistryURLKey = ConfigImageRegistryKey + d + "url"
	//ConfigImageRegistryNamespaceKey represents image registry namespace Key
	ConfigImageRegistryNamespaceKey = ConfigImageRegistryKey + d + "namespace"
	//ConfigImageRegistryLoginTypeKey represents image registry login type Key
	ConfigImageRegistryLoginTypeKey = ConfigImageRegistryKey + d + "%s" + d + "logintype"
	//ConfigImageRegistryPullSecretKey represents image registry pull secret Key
	ConfigImageRegistryPullSecretKey = ConfigImageRegistryKey + d + "%s" + d + "pullsecret"
	//ConfigImageRegistryUserNameKey represents image registry login Username Key
	ConfigImageRegistryUserNameKey = ConfigImageRegistryKey + d + "%s" + d + "username"
	//ConfigImageRegistryPasswordKey represents image registry login Password Key
	ConfigImageRegistryPasswordKey = ConfigImageRegistryKey + d + "%s" + d + "password"
	//ConfigStoragesPVCForHostPathKey represents key for PVC for Host Path
	ConfigStoragesPVCForHostPathKey = ConfigStoragesKey + d + "pvcforhostpath"
	//ConfigStoragesPerClaimStorageClassKey represents key for having different storage class for claim
	ConfigStoragesPerClaimStorageClassKey = ConfigStoragesKey + d + "perclaimstorageclass"
	//ConfigServicesNamesKey is true if a detected service is enabled for transformation
	ConfigServicesNamesKey = ConfigServicesKey + d + Special + d + "enable"
	//ConfigContainerizationTypesKey represents source type Key
	ConfigContainerizationTypesKey = ConfigContainerizationKeySegment + d + "types"
	//ConfigServicesExposeKey represents Services Expose Key
	ConfigServicesExposeKey = ConfigServicesKey + d + Special + d + "expose"
	// ConfigActiveMavenProfilesForServiceKeySegment represents the maven profiles used for service
	ConfigActiveMavenProfilesForServiceKeySegment = "activemavenprofiles"
	// ConfigActiveSpringBootProfilesForServiceKeySegment represent the springboot profiles used for service
	ConfigActiveSpringBootProfilesForServiceKeySegment = "activespringbootprofiles"
	// ConfigServicesChildModulesNamesKey is true if a detected child module/sub-project of a service is enabled for transformation
	ConfigServicesChildModulesNamesKey = ConfigServicesKey + d + "%s" + d + "childModules" + d + Special + d + "enable"
	// ConfigServicesDotNetChildProjectsNamesKey is true if a detected child-project of a dot net service is enabled for transformation
	ConfigServicesDotNetChildProjectsNamesKey = ConfigServicesKey + d + "%s" + d + "childProjects" + d + Special + d + "enable"
	// ConfigServicesChildModulesSpringProfilesKey is the list of spring profiles for this child module. 1st arg is service name and 2nd is child module name.
	ConfigServicesChildModulesSpringProfilesKey = ConfigServicesKey + d + "%s" + d + "childModules" + d + "%s" + d + "springBootProfiles"
	// ConfigTransformersKubernetesArgoCDNamespaceKey represents namespace key for argocd transformer
	ConfigTransformersKubernetesArgoCDNamespaceKey = ConfigTransformersKey + d + "kubernetes" + d + "argocd" + d + "namespace"
	//VCSKey represents version control system key
	VCSKey = BaseKey + d + "vcs"
	//GitKey represents git qa key
	GitKey = VCSKey + d + "git"
)

const (
	// DefaultDockerfileName refers to the default Dockerfile name
	DefaultDockerfileName = "Dockerfile"
	// VcapServiceEnvName refers to the VCAP_SERVICES environment variable DefaultDockerfileName
	VcapServiceEnvName = "VCAP_SERVICES"
	// VcapApplicationEnvName refers to the VCAP_APPLICATION environment variable DefaultDockerfileName
	VcapApplicationEnvName = "VCAP_APPLICATION"
	// VcapSpringBootSecretSuffix refers to VCAP springboot secret suffix
	VcapSpringBootSecretSuffix = "-vcapasspringbootproperties"
	// VcapCfSecretSuffix refers to VCAP secret suffix
	VcapCfSecretSuffix = "-vcapasenv"
)

const (
	// ProjectNameTemplatizedStringKey is the key for denoting project name in a templatized string
	ProjectNameTemplatizedStringKey = "ProjectName"
	// ArtifactNameTemplatizedStringKey is the key for denoting artifact name in a templatized string
	ArtifactNameTemplatizedStringKey = "ArtifactName"
	// ServiceNameTemplatizedStringKey is the key for denoting service name in a templatized string
	ServiceNameTemplatizedStringKey = "ServiceName"
	// ArtifactTypeTemplatizedStringKey is the key for denoting artifact type in a templatized string
	ArtifactTypeTemplatizedStringKey = "ArtifactType"
)

var (
	// DefaultPVCSize stores the default PVC size
	DefaultPVCSize, _ = resource.ParseQuantity("100Mi")
	// IgnoreEnvironment indicates whether to ignore the current environment or not
	IgnoreEnvironment = false
	// DisableLocalExecution indicates whether to allow execution of local executables
	DisableLocalExecution = false
	// DefaultIgnoreDirRegexps specifies directory name regexes that would be ignored
	DefaultIgnoreDirRegexps = []*regexp.Regexp{regexp.MustCompile("^[.].*")}
	// DisabledCategories is a list of QA categories that are disabled
	DisabledCategories = []string{}
	// QACategoryMap maps category names to problem IDs
	QACategoryMap = map[string][]string{}
	// disallowedDNSCharactersRegex provides pattern for characters not allowed in a DNS Name
	disallowedDNSCharactersRegex = regexp.MustCompile(`[^a-z0-9\-]`)
	// disallowedEnvironmentCharactersRegex provides pattern for characters not allowed in a DNS Name
	disallowedEnvironmentCharactersRegex = regexp.MustCompile(`[^A-Z0-9\_]`)
)

// PlanProgressNumBaseDetectTransformers keeps track of the number of transformers that finished base directory detect during planning
var PlanProgressNumBaseDetectTransformers = 0

// PlanProgressNumDirectories keeps track of the number of files/folders analyzed during planning
var PlanProgressNumDirectories = 0

// CompressionType refers to the compression type
type CompressionType = string

const (
	// GZipCompression allows archival using gzip compression format
	GZipCompression CompressionType = "GZip"
	// NoCompression allows archival without compression
	NoCompression CompressionType = "None"
)
