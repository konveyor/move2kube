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

package apiresourceset

import (
	"fmt"
	"strings"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/knownhosts"
	"github.com/konveyor/move2kube/internal/common/sshkeys"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/internal/types/tekton"
	plantypes "github.com/konveyor/move2kube/types/plan"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	giturls "github.com/whilp/git-urls"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	gitDomainPlaceholder                   = "<TODO: insert git repo domain>"
	knownHostsPlaceholder                  = "<TODO: insert the known host keys for your git repo>"
	gitPrivateKeyPlaceholder               = "<TODO: insert the private ssh key for your git repo>"
	ingressHostNamePlaceholder             = "<TODO: insert subdomain where you want to receive git events>"
	ingressTLSSecretNamePlaceholder        = "<TODO: insert name of TLS secret>"
	storageClassPlaceholder                = "<TODO: insert the storage class you want to use>"
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

// TektonAPIResourceSet is the set of CI/CD resources we generate.
type TektonAPIResourceSet struct {
}

func (*TektonAPIResourceSet) getAPIResources(ir irtypes.IR) []apiresource.APIResource {
	apiresources := []apiresource.APIResource{
		apiresource.APIResource{IAPIResource: &apiresource.Service{}},
		apiresource.APIResource{IAPIResource: &apiresource.EventListener{}},
		apiresource.APIResource{IAPIResource: &apiresource.TriggerBinding{}},
		apiresource.APIResource{IAPIResource: &apiresource.TriggerTemplate{}},
		apiresource.APIResource{IAPIResource: &apiresource.Pipeline{}},
		apiresource.APIResource{IAPIResource: &apiresource.ServiceAccount{}},
		apiresource.APIResource{IAPIResource: &apiresource.RoleBinding{}},
		apiresource.APIResource{IAPIResource: &apiresource.Role{}},
		apiresource.APIResource{IAPIResource: &apiresource.Storage{}},
	}
	return apiresources
}

// CreateAPIResources converts IR to runtime objects
func (tekSet *TektonAPIResourceSet) CreateAPIResources(ir irtypes.IR) []runtime.Object {
	ir = tekSet.setupIR(ir)
	targetobjs := []runtime.Object{}
	for _, a := range tekSet.getAPIResources(ir) {
		a.SetClusterContext(ir.TargetClusterSpec)
		objs := a.GetUpdatedResources(ir)
		targetobjs = append(targetobjs, objs...)
	}
	return targetobjs
}

func (*TektonAPIResourceSet) setupIR(oldir irtypes.IR) irtypes.IR {
	// Create a new ir containing just the subset we care about
	ir := irtypes.NewIR(plantypes.NewPlan())
	ir.Name = oldir.Name
	ir.TargetClusterSpec = oldir.TargetClusterSpec
	ir.Kubernetes = oldir.Kubernetes
	ir.Containers = oldir.Containers

	// Prefix the project name and make the name a valid k8s name.
	projectName := ir.Name
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
	// https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#how-does-the-eventlistener-work
	gitEventListenerServiceName := "el-" + gitEventListenerName

	res := tekton.Resources{}
	res.EventListeners = []tekton.EventListener{{
		Name:                gitEventListenerName,
		ServiceAccountName:  gitEventListenerName,
		TriggerBindingName:  triggerBindingName,
		TriggerTemplateName: triggerTemplateName,
	}}
	res.TriggerBindings = []tekton.TriggerBinding{{Name: triggerBindingName}}
	res.TriggerTemplates = []tekton.TriggerTemplate{{
		Name:               triggerTemplateName,
		PipelineName:       pipelineName,
		PipelineRunName:    pipelineName,
		ServiceAccountName: clonePushServiceAccountName,
		WorkspaceName:      workspaceName,
		StorageClassName:   storageClassPlaceholder,
	}}
	res.Pipelines = []tekton.Pipeline{{
		Name:          pipelineName,
		WorkspaceName: workspaceName,
	}}
	ir.TektonResources = res
	ir.Services = map[string]irtypes.Service{
		gitEventListenerServiceName: {
			Name:          gitEventListenerServiceName,
			ExposeService: true,
			OnlyIngress:   true,
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Ports: []corev1.ContainerPort{{ContainerPort: int32(8080)}},
				}},
			},
		},
	}

	ir.RoleBindings = append(ir.RoleBindings, irtypes.RoleBinding{
		Name:               tektonTriggersAdminRoleBindingName,
		RoleName:           tektonTriggersAdminRoleName,
		ServiceAccountName: tektonTriggersAdminServiceAccountName,
	})
	ir.Roles = append(ir.Roles, irtypes.Role{
		Name: tektonTriggersAdminRoleName,
		PolicyRules: []irtypes.PolicyRule{
			irtypes.PolicyRule{APIGroups: []string{triggersv1alpha1.SchemeGroupVersion.Group}, Resources: []string{"eventlisteners", "triggerbindings", "triggertemplates"}, Verbs: []string{"get"}},
			irtypes.PolicyRule{APIGroups: []string{v1beta1.SchemeGroupVersion.Group}, Resources: []string{"pipelineruns"}, Verbs: []string{"create"}},
			irtypes.PolicyRule{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get", "list", "watch"}},
		},
	})

	// TODO: this could also be common.DefaultRegistryURL
	registryURL := registryURLPlaceholder
	if ir.Kubernetes.RegistryURL != "" {
		registryURL = ir.Kubernetes.RegistryURL
		if registryURL == "docker.io" {
			// There's some bug with kaniko that requires this
			registryURL = "index.docker.io"
		}
	}

	dockerConfigJSON := dockerConfigJSONPlaceholder
	imageRegistrySecret := irtypes.Storage{
		StorageType: irtypes.SecretKind,
		Name:        registrySecretName,
		SecretType:  corev1.SecretTypeDockerConfigJson,
		// TODO: not sure if this annotation is specific to docker hub
		Annotations: map[string]string{"tekton.dev/docker-0": registryURL},
		StringData:  map[string]string{corev1.DockerConfigJsonKey: dockerConfigJSON},
	}

	secrets := []irtypes.Storage{imageRegistrySecret}
	gitDomains := []string{}
	for _, container := range ir.Containers {
		if !container.New {
			continue
		}
		gitRepoURL, err := giturls.Parse(container.RepoInfo.GitRepoURL)
		if err != nil {
			log.Warnf("Failed to parse git repo url %q Error: %q", container.RepoInfo.GitRepoURL, err)
			continue
		}
		if gitRepoURL.Hostname() == "" {
			continue
		}
		gitDomains = append(gitDomains, gitRepoURL.Hostname())
	}
	gitDomains = common.UniqueStrings(gitDomains)
	if len(gitDomains) == 0 {
		log.Warn("No remote git repos found. CI/CD pipeline requires a remote git repo to pull the source code from.")
	} else {
		for _, gitDomain := range gitDomains {
			gitSecretName := fmt.Sprintf("%s-%s", gitSecretNamePrefix, gitDomain)
			gitSecretName = common.MakeStringDNSSubdomainNameCompliant(gitSecretName)
			secrets = append(secrets, createGitSecret(gitSecretName, gitDomain))
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

func createGitSecret(name, gitRepoDomain string) irtypes.Storage {
	gitPrivateKey := gitPrivateKeyPlaceholder
	knownHosts := knownHostsPlaceholder
	if gitRepoDomain == "" {
		gitRepoDomain = gitDomainPlaceholder
	} else {
		if pubKeys, ok := sshkeys.DomainToPublicKeys[gitRepoDomain]; ok { // Check in our hardcoded set of keys and their ~/.ssh/known_hosts file.
			knownHosts = strings.Join(pubKeys, "\n")
		} else if pubKeyLine, err := knownhosts.GetKnownHostsLine(gitRepoDomain); err == nil { // Check online by connecting to the host.
			knownHosts = pubKeyLine
		} else {
			problemDesc := fmt.Sprintf("Unable to find the public key for the domain %s from known_hosts, please enter it. If you are not sure what this is press Enter and you will be able to edit it later: ", gitRepoDomain)
			example := sshkeys.DomainToPublicKeys["github.com"][0]
			problem, err := qatypes.NewInputProblem(problemDesc, []string{"Ex : " + example}, knownHostsPlaceholder)
			if err != nil {
				log.Fatalf("Unable to create problem : %s", err)
			}
			problem, err = qaengine.FetchAnswer(problem)
			if err != nil {
				log.Fatalf("Unable to fetch answer : %s", err)
			}
			newline, err := problem.GetStringAnswer()
			if err != nil {
				log.Fatalf("Unable to get answer : %s", err)
			}
			knownHosts = newline
		}

		if key, ok := sshkeys.GetSSHKey(gitRepoDomain); ok {
			gitPrivateKey = key
		}
	}

	return irtypes.Storage{
		StorageType: irtypes.SecretKind,
		Name:        name,
		SecretType:  corev1.SecretTypeSSHAuth,
		Annotations: map[string]string{"tekton.dev/git-0": gitRepoDomain},
		StringData: map[string]string{
			corev1.SSHAuthPrivateKey: gitPrivateKey,
			"known_hosts":            knownHosts,
		},
	}
}
