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

package generators

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/knownhosts"
	"github.com/konveyor/move2kube/internal/common/sshkeys"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	core "k8s.io/kubernetes/pkg/apis/core"
)

const (
	gitDomainPlaceholder                   = "<TODO: insert git repo domain>"
	knownHostsPlaceholder                  = "<TODO: insert the known host keys for your git repo>"
	gitPrivateKeyPlaceholder               = "<TODO: insert the private ssh key for your git repo>"
	registryURLPlaceholder                 = "<TODO: insert the image registry URL>"
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
)

const (
	TektonArtifacts = "TektonYamls"
)

// Tekton implements Transformer interface
type Tekton struct {
	Config transformertypes.Transformer
	Env    environment.Environment
}

type TektonConfig struct {
}

func (t *Tekton) Init(tc transformertypes.Transformer, env environment.Environment) error {
	t.Config = tc
	t.Env = env
	return nil
}

func (t *Tekton) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.Config, t.Env
}

func (t *Tekton) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

func (t *Tekton) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

func (t *Tekton) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	logrus.Debugf("Translating IR using Kubernetes transformer")
	pathMappings = []transformertypes.PathMapping{}
	for _, a := range newArtifacts {
		if a.Artifact != irtypes.IRArtifactType {
			continue
		}
		var ir irtypes.IR
		err := a.GetConfig(irtypes.IRConfigType, &ir)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", ir, err)
			continue
		}
		var pC artifacts.PlanConfig
		err = a.GetConfig(artifacts.PlanConfigType, &pC)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", pC, err)
			continue
		}
		apis := []apiresource.IAPIResource{&apiresource.Service{},
			&apiresource.ServiceAccount{},
			&apiresource.RoleBinding{},
			&apiresource.Role{},
			&apiresource.Storage{},
			&apiresource.EventListener{},
			&apiresource.TriggerBinding{},
			&apiresource.TriggerTemplate{},
			&apiresource.Pipeline{},
		}
		tempDest := filepath.Join(t.Env.TempPath, common.DeployDir, common.CICDDir, "tekton")
		logrus.Infof("Generating Tekton pipeline for CI/CD")
		enhancedIR := t.SetupEnhancedIR(ir, pC.PlanName)
		if files, err := apiresource.TransformAndPersist(enhancedIR, tempDest, apis, pC.TargetCluster); err == nil {
			for _, f := range files {
				if destPath, err := filepath.Rel(t.Env.TempPath, f); err != nil {
					logrus.Errorf("Invalid yaml path : %s", destPath)
				} else {
					pathMappings = append(pathMappings, transformertypes.PathMapping{
						Type:     transformertypes.DefaultPathMappingType,
						SrcPath:  f,
						DestPath: destPath,
					})
				}
			}
			logrus.Debugf("Total transformed objects : %d", len(files))
		} else {
			logrus.Errorf("Unable to transform and persist IR : %s", err)
			return nil, nil, err
		}
	}
	return pathMappings, nil, nil
}

// SetupEnhancedIR returns EnhancedIR containing Tekton components
func (t *Tekton) SetupEnhancedIR(oldir irtypes.IR, name string) irtypes.EnhancedIR {
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
		StorageClassName:   common.DefaultStorageClassName,
	}}
	res.Pipelines = []irtypes.Pipeline{{
		Name:          pipelineName,
		WorkspaceName: workspaceName,
	}}
	ir.TektonResources = res
	ir.Services = map[string]irtypes.Service{gitEventIngressName: {
		Name:               gitEventIngressName,
		BackendServiceName: gitEventListenerServiceName,
		OnlyIngress:        true,
		ServiceRelPath:     "/" + gitEventListenerServiceName, // this has to be an absolute path otherwise k8s will complain
		PodSpec: core.PodSpec{
			Containers: []core.Container{{
				Ports: []core.ContainerPort{{ContainerPort: int32(8080)}},
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
		if container.Build.ContextPath != "" {
			_, _, gitRepoHostName, gitRepoURL, _, err := common.GatherGitInfo(container.Build.ContextPath)
			if err != nil {
				logrus.Warnf("Failed to parse git repo url %q Error: %q", gitRepoURL, err)
				continue
			}
			if gitRepoHostName == "" {
				continue
			}
			gitDomains = append(gitDomains, gitRepoHostName)
		}
	}
	gitDomains = common.UniqueStrings(gitDomains)
	if len(gitDomains) == 0 {
		logrus.Info("No remote git repos detected. You might want to configure the git repository links manually.")
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
			qaKey := common.ConfigRepoLoadPubDomainsKey + common.Delim + `"` + gitRepoDomain + `"` + common.Delim + "pubkey"
			knownHosts = qaengine.FetchStringAnswer(qaKey, problemDesc, hints, knownHostsPlaceholder)
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
