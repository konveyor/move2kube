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

package transform

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/knownhosts"
	"github.com/konveyor/move2kube/internal/common/sshkeys"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/internal/types/tekton"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	giturls "github.com/whilp/git-urls"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// TektonTransformer is the set of tekton based CI/CD resources we generate.
type TektonTransformer struct {
	shouldRun                bool
	transformedTektonObjects []runtime.Object
	TargetClusterSpec        collecttypes.ClusterMetadataSpec
	IgnoreUnsupportedKinds   bool
	extraFiles               map[string]string // file path: file contents
}

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

// Transform translates intermediate representation to destination objects
func (tekSet *TektonTransformer) Transform(ir irtypes.IR) error {
	tekSet.shouldRun = false
	for _, container := range ir.Containers {
		if container.New {
			tekSet.shouldRun = true
			break
		}
	}
	if !tekSet.shouldRun {
		return nil
	}
	tekSet.TargetClusterSpec = ir.TargetClusterSpec
	tekSet.IgnoreUnsupportedKinds = ir.Kubernetes.IgnoreUnsupportedKinds
	log.Infof("Generating Tekton pipeline for CI/CD")
	enhancedIR := tekSet.SetupEnhancedIR(ir)
	tektonResources := tekSet.GetAPIResources()
	tekSet.transformedTektonObjects = convertIRToObjects(enhancedIR, tektonResources)
	if !ir.TargetClusterSpec.IsTektonInstalled() {
		log.Infof("Tekton was not found on the target cluster. Please install Tekton Pipelines for CI/CD (https://github.com/tektoncd/pipeline).")
	}
	return nil
}

// WriteObjects writes Transformed objects to filesystem. Also does some final transformations on the generated yamls.
func (tekSet *TektonTransformer) WriteObjects(outputPath string, transformPaths []string) error {
	if !tekSet.shouldRun {
		return nil
	}
	cicdPath := filepath.Join(outputPath, common.DeployDir, "cicd")
	// deploy/cicd/tekton/
	tektonPath := filepath.Join(cicdPath, "tekton")
	if _, err := writeTransformedObjects(tektonPath, tekSet.transformedTektonObjects, tekSet.TargetClusterSpec, false, transformPaths); err != nil {
		log.Errorf("Error occurred while writing transformed objects. Error: %q", err)
		return err
	}
	if len(tekSet.extraFiles) == 0 {
		return nil
	}
	for relFilePath, fileContents := range tekSet.extraFiles {
		filePath := filepath.Join(outputPath, relFilePath)
		filePerms := common.DefaultFilePermission
		if filepath.Ext(relFilePath) == ".sh" {
			filePerms = common.DefaultExecutablePermission
		}
		parentPath := filepath.Dir(filePath)
		if err := os.MkdirAll(parentPath, common.DefaultDirectoryPermission); err != nil {
			log.Errorf("Unable to create directory at path %s Error: %q", parentPath, err)
			continue
		}
		if err := ioutil.WriteFile(filePath, []byte(fileContents), filePerms); err != nil {
			log.Errorf("Failed to write the contents %s to a file at path %s Error: %q", fileContents, filePath, err)
		}
	}
	return nil
}

// GetAPIResources returns api resources related to Tekton
func (tekSet *TektonTransformer) GetAPIResources() []apiresource.IAPIResource {
	return []apiresource.IAPIResource{&apiresource.Service{},
		&apiresource.ServiceAccount{},
		&apiresource.RoleBinding{},
		&apiresource.Role{},
		&apiresource.Storage{},
		&apiresource.EventListener{},
		&apiresource.TriggerBinding{},
		&apiresource.TriggerTemplate{},
		&apiresource.Pipeline{},
	}
}

// SetupEnhancedIR returns EnhancedIR containing Tekton components
func (tekSet *TektonTransformer) SetupEnhancedIR(oldir irtypes.IR) irtypes.EnhancedIR {
	ir := irtypes.NewEnhancedIRFromIR(oldir)

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
	gitEventIngressName := p(baseGitEventIngressName)
	// https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#how-does-the-eventlistener-work
	gitEventListenerServiceName := "el-" + gitEventListenerName

	res := tekton.Resources{}
	res.EventListeners = []tekton.EventListener{{
		Name:                gitEventListenerName,
		ServiceAccountName:  tektonTriggersAdminServiceAccountName,
		TriggerBindingName:  triggerBindingName,
		TriggerTemplateName: triggerTemplateName,
	}}
	res.TriggerBindings = []tekton.TriggerBinding{{Name: triggerBindingName}}
	res.TriggerTemplates = []tekton.TriggerTemplate{{
		Name:               triggerTemplateName,
		PipelineName:       pipelineName,
		PipelineRunName:    pipelineName + "-$(uid)", // appends a random string to the name to make it unique
		ServiceAccountName: clonePushServiceAccountName,
		WorkspaceName:      workspaceName,
		StorageClassName:   common.DefaultStorageClassName,
	}}
	res.Pipelines = []tekton.Pipeline{{
		Name:          pipelineName,
		WorkspaceName: workspaceName,
	}}
	ir.TektonResources = res
	ir.Services = map[string]irtypes.Service{
		gitEventListenerServiceName: {
			Name:               gitEventIngressName,
			BackendServiceName: gitEventListenerServiceName,
			OnlyIngress:        true,
			ServiceRelPath:     "/" + gitEventListenerServiceName, // this has to be an absolute path otherwise k8s will complain
			PodSpec: core.PodSpec{
				Containers: []core.Container{{
					Ports: []core.ContainerPort{{ContainerPort: int32(8080)}},
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
			{APIGroups: []string{triggersv1alpha1.SchemeGroupVersion.Group}, Resources: []string{"eventlisteners", "triggerbindings", "triggertemplates"}, Verbs: []string{"get"}},
			{APIGroups: []string{v1beta1.SchemeGroupVersion.Group}, Resources: []string{"pipelineruns"}, Verbs: []string{"create"}},
			{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get", "list", "watch"}},
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
		SecretType:  core.SecretTypeDockerConfigJSON,
		// TODO: not sure if this annotation is specific to docker hub
		Annotations: map[string]string{"tekton.dev/docker-0": registryURL},
		Content:     map[string][]byte{core.DockerConfigJSONKey: []byte(dockerConfigJSON)},
	}

	secrets := []irtypes.Storage{imageRegistrySecret}
	gitDomains := []string{}
	for _, container := range ir.Containers {
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
		log.Info("No remote git repos detected. You might want to configure the git repository links manually.")
	} else {
		for _, gitDomain := range gitDomains {
			// This name is also used by tekton to create a volume to hold secrets. If there is a dot k8s will complain.
			normalizedGitDomain := strings.Replace(gitDomain, ".", "-", -1)
			gitSecretName := fmt.Sprintf("%s-%s", gitSecretNamePrefix, normalizedGitDomain)
			gitSecretName = common.MakeStringDNSSubdomainNameCompliant(gitSecretName)
			secrets = append(secrets, tekSet.createGitSecret(gitSecretName, gitDomain))
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

func (*TektonTransformer) createGitSecret(name, gitRepoDomain string) irtypes.Storage {
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
