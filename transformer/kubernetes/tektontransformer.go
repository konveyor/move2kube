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

package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/common/knownhosts"
	"github.com/konveyor/move2kube/common/sshkeys"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer/kubernetes/apiresource"
	"github.com/konveyor/move2kube/transformer/kubernetes/irpreprocessor"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	core "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/networking"
)

const (
	gitDomainPlaceholder                   = "<TODO: insert git repo domain>"
	knownHostsPlaceholder                  = "<TODO: insert the known host keys for your git repo>"
	gitPrivateKeyPlaceholder               = "<TODO: insert the private ssh key for your git repo>"
	dockerConfigJSONPlaceholder            = "<TODO: insert your docker config json>"
	baseGitSecretName                      = "git-repo"
	baseWorkspaceName                      = "shared-data"
	baseTektonTriggersServiceAccountName   = "tekton-triggers-admin"
	baseTriggerBindingName                 = "git-event"
	baseTriggerTemplateName                = "run-clone-build-push"
	basePipelineName                       = "clone-build-push"
	baseClonePushServiceAccountName        = "clone-push"
	baseRegistrySecretName                 = "image-registry"
	baseGitEventListenerName               = "git-repo"
	baseGitEventIngressName                = "git-repo"
	baseTektonTriggersAdminRoleName        = "tekton-triggers-admin"
	baseTektonTriggersAdminRoleBindingName = "tekton-triggers-admin"
	defaultTektonYamlsOutputPath           = common.DeployDir + string(os.PathSeparator) + common.CICDDir + string(os.PathSeparator) + "tekton"
)

// Tekton implements Transformer interface
type Tekton struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	TektonConfig *TektonYamlConfig
}

// TektonYamlConfig stores the Tekton related information
type TektonYamlConfig struct {
	OutputPath              string `yaml:"outputPath"`
	SetDefaultValuesInYamls bool   `yaml:"setDefaultValuesInYamls"`
}

// Init Initializes the transformer
func (t *Tekton) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.Env = env
	t.TektonConfig = &TektonYamlConfig{}
	err := common.GetObjFromInterface(t.Config.Spec.Config, t.TektonConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.TektonConfig, err)
		return err
	}
	if t.TektonConfig.OutputPath == "" {
		t.TektonConfig.OutputPath = defaultTektonYamlsOutputPath
	}
	if !t.TektonConfig.SetDefaultValuesInYamls {
		t.TektonConfig.SetDefaultValuesInYamls = setDefaultValuesInYamls
	}
	return nil
}

// GetConfig returns the configuration
func (t *Tekton) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each subdirectory
func (t *Tekton) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms artifacts understood by the transformer
func (t *Tekton) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	logrus.Debugf("Translating IR using Kubernetes transformer")
	pathMappings = []transformertypes.PathMapping{}
	createdArtifacts = []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if newArtifact.Type != irtypes.IRArtifactType {
			continue
		}
		var ir irtypes.IR
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", ir, err)
			continue
		}
		var clusterConfig collecttypes.ClusterMetadata
		if err := newArtifact.GetConfig(ClusterMetadata, &clusterConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", clusterConfig, err)
			continue
		}
		ir.Name = newArtifact.Name
		preprocessedIR, err := irpreprocessor.Preprocess(ir)
		if err != nil {
			logrus.Errorf("Unable to prepreocess IR : %s", err)
		} else {
			ir = preprocessedIR
		}
		resources := []apiresource.IAPIResource{
			new(apiresource.Service),
			new(apiresource.ServiceAccount),
			new(apiresource.RoleBinding),
			new(apiresource.Role),
			new(apiresource.Storage),
			new(apiresource.EventListener),
			new(apiresource.TriggerBinding),
			new(apiresource.TriggerTemplate),
			new(apiresource.Pipeline),
		}
		deployCICDDir := t.TektonConfig.OutputPath
		tempDest := filepath.Join(t.Env.TempPath, deployCICDDir)
		logrus.Debugf("Generating Tekton pipeline for CI/CD")
		enhancedIR := t.setupEnhancedIR(ir, t.Env.GetProjectName())
		files, err := apiresource.TransformIRAndPersist(enhancedIR, tempDest, resources, clusterConfig, t.TektonConfig.SetDefaultValuesInYamls)
		if err != nil {
			logrus.Errorf("Unable to transform and persist IR : %s", err)
			return nil, nil, err
		}
		for _, f := range files {
			destPath, err := filepath.Rel(t.Env.TempPath, f)
			if err != nil {
				logrus.Errorf("Invalid yaml path : %s", destPath)
				continue
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.DefaultPathMappingType,
				SrcPath:  f,
				DestPath: destPath,
			})
		}
		createdArtifact := transformertypes.Artifact{
			Name: t.Config.Name,
			Type: artifacts.KubernetesYamlsArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.KubernetesYamlsPathType: {deployCICDDir},
			},
		}
		createdArtifacts = append(createdArtifacts, createdArtifact)
		logrus.Debugf("Total transformed objects : %d", len(files))
	}
	return pathMappings, createdArtifacts, nil
}

// setupEnhancedIR returns EnhancedIR containing Tekton components
func (t *Tekton) setupEnhancedIR(oldir irtypes.IR, name string) irtypes.EnhancedIR {
	ir := irtypes.NewEnhancedIRFromIR(oldir)

	// Prefix the project name and make the name a valid k8s name.
	projectName := name
	p := func(name string) string {
		name = fmt.Sprintf("%s-%s", projectName, name)
		return common.MakeStringDNSSubdomainNameCompliant(name)
	}
	pipelineName := p(basePipelineName)
	gitSecretNamePrefix := p(baseGitSecretName)
	clonePushServiceAccountName := p(baseClonePushServiceAccountName)
	registrySecretName := p(baseRegistrySecretName)
	gitEventListenerName := p(baseGitEventListenerName)
	triggerBindingName := p(baseTriggerBindingName)
	tektonTriggersAdminServiceAccountName := p(baseTektonTriggersServiceAccountName)
	triggerTemplateName := p(baseTriggerTemplateName)
	workspaceName := p(baseWorkspaceName)
	tektonTriggersAdminRoleName := p(baseTektonTriggersAdminRoleName)
	tektonTriggersAdminRoleBindingName := p(baseTektonTriggersAdminRoleBindingName)
	gitEventIngressName := p(baseGitEventIngressName)
	// https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#how-does-the-eventlistener-work
	gitEventListenerServiceName := "el-" + gitEventListenerName

	res := irtypes.TektonResources{}
	res.EventListeners = []irtypes.EventListener{{
		Name:                gitEventListenerName,
		ServiceAccountName:  tektonTriggersAdminServiceAccountName,
		TriggerBindingName:  triggerBindingName,
		TriggerTemplateName: triggerTemplateName,
	}}
	res.TriggerBindings = []irtypes.TriggerBinding{{Name: triggerBindingName}}
	res.TriggerTemplates = []irtypes.TriggerTemplate{{
		Name:               triggerTemplateName,
		PipelineName:       pipelineName,
		PipelineRunName:    pipelineName + "-$(uid)", // appends a random string to the name to make it unique
		ServiceAccountName: clonePushServiceAccountName,
		WorkspaceName:      workspaceName,
		StorageClassName:   defaultStorageClassName,
	}}
	res.Pipelines = []irtypes.Pipeline{{
		Name:          pipelineName,
		WorkspaceName: workspaceName,
	}}
	ir.TektonResources = res
	var port int32 = common.DefaultServicePort
	ir.Services = map[string]irtypes.Service{gitEventIngressName: {
		Name:               gitEventIngressName,
		BackendServiceName: gitEventListenerServiceName,
		OnlyIngress:        true,
		ServiceToPodPortForwardings: []irtypes.ServiceToPodPortForwarding{{
			ServicePort:    networking.ServiceBackendPort{Number: port},
			PodPort:        networking.ServiceBackendPort{Number: port},
			ServiceRelPath: "/" + gitEventListenerServiceName, // this has to be an absolute path otherwise k8s will complain
			ServiceType:    core.ServiceTypeClusterIP,
		}},
		PodSpec: irtypes.PodSpec{
			Containers: []core.Container{{
				Ports: []core.ContainerPort{{ContainerPort: port}},
			}}},
	}}

	ir.RoleBindings = append(ir.RoleBindings, irtypes.RoleBinding{
		Name:               tektonTriggersAdminRoleBindingName,
		RoleName:           tektonTriggersAdminRoleName,
		ServiceAccountName: tektonTriggersAdminServiceAccountName,
	})
	ir.Roles = append(ir.Roles, irtypes.Role{
		Name: tektonTriggersAdminRoleName,
		PolicyRules: []irtypes.PolicyRule{
			{APIGroups: []string{triggersv1alpha1.SchemeGroupVersion.Group}, Resources: []string{"eventlisteners", "triggerbindings", "triggertemplates"}, Verbs: []string{"get"}},
			{APIGroups: []string{v1beta1.SchemeGroupVersion.Group}, Resources: []string{"pipelineruns"}, Verbs: []string{"create"}},
			{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get", "list", "watch"}},
		},
	})

	dockerConfigJSON := dockerConfigJSONPlaceholder
	imageRegistrySecret := irtypes.Storage{
		StorageType: irtypes.SecretKind,
		Name:        registrySecretName,
		SecretType:  core.SecretTypeDockerConfigJSON,
		// TODO: not sure if this annotation is specific to docker hub
		// TOFIX: Fix registry url
		Annotations: map[string]string{"tekton.dev/docker-0": ""},
		Content:     map[string][]byte{core.DockerConfigJSONKey: []byte(dockerConfigJSON)},
	}

	secrets := []irtypes.Storage{imageRegistrySecret}
	gitDomains := []string{}
	for _, container := range ir.ContainerImages {
		if container.Build.ContextPath == "" {
			continue
		}
		_, _, gitRepoHostName, gitRepoURL, _, err := common.GatherGitInfo(container.Build.ContextPath)
		if err != nil {
			logrus.Debugf("failed to gather git info. Error: %q", err)
			if gitRepoURL != "" {
				logrus.Warnf("Failed to parse git repo url %q Error: %q", gitRepoURL, err)
			}
			continue
		}
		if gitRepoHostName == "" {
			continue
		}
		gitDomains = append(gitDomains, gitRepoHostName)
	}
	gitDomains = common.UniqueStrings(gitDomains)
	if len(gitDomains) == 0 {
		logrus.Debug("No remote git repos detected. You might want to configure the git repository links manually.")
	} else {
		for _, gitDomain := range gitDomains {
			// This name is also used by tekton to create a volume to hold secrets. If there is a dot k8s will complain.
			normalizedGitDomain := strings.Replace(gitDomain, ".", "-", -1)
			gitSecretName := fmt.Sprintf("%s-%s", gitSecretNamePrefix, normalizedGitDomain)
			gitSecretName = common.MakeStringDNSSubdomainNameCompliant(gitSecretName)
			secrets = append(secrets, t.createGitSecret(gitSecretName, gitDomain))
		}
	}
	ir.Storages = append(ir.Storages, secrets...)

	secretNames := []string{}
	for _, secret := range secrets {
		secretNames = append(secretNames, secret.Name)
	}
	ir.ServiceAccounts = append(
		ir.ServiceAccounts,
		irtypes.ServiceAccount{Name: tektonTriggersAdminServiceAccountName},
		irtypes.ServiceAccount{Name: clonePushServiceAccountName, SecretNames: secretNames},
	)
	return ir
}

func (t *Tekton) createGitSecret(name, gitRepoDomain string) irtypes.Storage {
	gitPrivateKey := gitPrivateKeyPlaceholder
	knownHosts := knownHostsPlaceholder
	if gitRepoDomain == "" {
		gitRepoDomain = gitDomainPlaceholder
	} else {
		sshkeys.LoadKnownHostsOfCurrentUser()
		if pubKeys, ok := sshkeys.DomainToPublicKeys[gitRepoDomain]; ok { // Check in our hardcoded set of keys and their ~/.ssh/known_hosts file.
			knownHosts = strings.Join(pubKeys, "\n")
		} else if pubKeyLine, err := knownhosts.GetKnownHostsLine(gitRepoDomain); err == nil { // Check online by connecting to the host.
			knownHosts = pubKeyLine
		} else {
			problemDesc := fmt.Sprintf("Unable to find the public key for the domain %s from known_hosts, please enter it. If don't know the public key, just leave this empty and you will be able to add it later: ", gitRepoDomain)
			hints := []string{"Ex : " + sshkeys.DomainToPublicKeys["github.com"][0]}
			qaKey := common.JoinQASubKeys(common.ConfigRepoLoadPubDomainsKey, `"`+gitRepoDomain+`"`, "pubkey")
			knownHosts = qaengine.FetchStringAnswer(qaKey, problemDesc, hints, knownHostsPlaceholder, nil)
		}

		if key, ok := sshkeys.GetSSHKey(gitRepoDomain); ok {
			gitPrivateKey = key
		}
	}

	return irtypes.Storage{
		StorageType: irtypes.SecretKind,
		Name:        name,
		SecretType:  core.SecretTypeSSHAuth,
		Annotations: map[string]string{"tekton.dev/git-0": gitRepoDomain},
		Content: map[string][]byte{
			core.SSHAuthPrivateKey: []byte(gitPrivateKey),
			"known_hosts":          []byte(knownHosts),
		},
	}
}
