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
	"path/filepath"
	"strings"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/knownhosts"
	"github.com/konveyor/move2kube/internal/common/sshkeys"
	"github.com/konveyor/move2kube/internal/qaengine"
	internaltypes "github.com/konveyor/move2kube/internal/types"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	giturls "github.com/whilp/git-urls"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

const (
	roleKind                               = "Role"
	roleBindingKind                        = "RoleBinding"
	pipelineKind                           = "Pipeline"
	pipelineRunKind                        = "PipelineRun"
	ingressKind                            = "Ingress"
	eventListenerKind                      = "EventListener"
	triggerTemplateKind                    = "TriggerTemplate"
	gitRepoURLPlaceholder                  = "<TODO: insert git repo url>"
	gitDomainPlaceholder                   = "<TODO: insert git repo domain>"
	contextPathPlaceholder                 = "<TODO: insert path to the directory containing Dockerfile>"
	dockerfilePathPlaceholder              = "<TODO: insert path to the Dockerfile>"
	knownHostsPlaceholder                  = "<TODO: insert the known host keys for your git repo>"
	gitPrivateKeyPlaceholder               = "<TODO: insert the private ssh key for your git repo>"
	defaultGitRepoBranch                   = "master"
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
	baseTektonTriggersAdminRoleName        = "tekton-triggers-admin-role"
	baseTektonTriggersAdminRoleBindingName = "tekton-triggers-admin-role-binding"
)

// TektonAPIResourceSet is the set of CI/CD resources we generate.
type TektonAPIResourceSet struct {
}

// CreateAPIResources creates CI/CD resources from the IR.
func (*TektonAPIResourceSet) CreateAPIResources(ir irtypes.IR) []runtime.Object {
	projectName := ir.Name
	// Prefix the project name and make the name a valid k8s name.
	p := func(name string) string {
		name = fmt.Sprintf("%s-%s", projectName, name)
		return common.MakeStringDNSSubdomainNameCompliant(name)
	}

	pipelineName := p(basePipelineName)
	gitSecretNamePrefix := p(baseGitSecretName)
	clonePushServiceAccountName := p(baseClonePushServiceAccountName)
	registrySecretName := p(baseRegistrySecretName)
	gitEventIngressName := p(baseGitEventIngressName)
	gitEventListenerName := p(baseGitEventListenerName)
	triggerBindingName := p(baseTriggerBindingName)
	tektonTriggersServiceAccountName := p(baseTektonTriggersServiceAccountName)
	triggerTemplateName := p(baseTriggerTemplateName)
	workspaceName := p(baseWorkspaceName)
	tektonTriggersAdminRoleName := p(baseTektonTriggersAdminRoleName)
	tektonTriggersAdminRoleBindingName := p(baseTektonTriggersAdminRoleBindingName)

	gitSecrets := createGitSecrets(gitSecretNamePrefix, ir)
	role, serviceAccount, roleBinding := createTektonTriggersRBAC(tektonTriggersAdminRoleName, tektonTriggersServiceAccountName, tektonTriggersAdminRoleBindingName)
	objs := []runtime.Object{
		role, serviceAccount, roleBinding,
		createCloneBuildPushPipeline(pipelineName, workspaceName, ir),
		createClonePushServiceAccount(clonePushServiceAccountName, gitSecrets, registrySecretName),
		createGitEventIngress(gitEventIngressName, gitEventListenerName),
		createGitEventTriggerBinding(triggerBindingName),
		createRegistrySecret(registrySecretName),
		createGitEventListener(gitEventListenerName, tektonTriggersServiceAccountName, triggerBindingName, triggerTemplateName),
		createTriggerTemplate(triggerTemplateName, pipelineName, clonePushServiceAccountName, workspaceName, ir),
	}
	for _, gitSecret := range gitSecrets {
		objs = append(objs, gitSecret)
	}
	return objs
}

func createGitSecrets(gitSecretNamePrefix string, ir irtypes.IR) [](*corev1.Secret) {
	secrets := [](*corev1.Secret){}
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
		// TODO: This message should be printed only if there are any new containers being created.
		log.Debug("No remote git repos found. CI/CD pipeline requires a remote git repo to pull the source code from.")
		gitSecretName := common.MakeStringDNSSubdomainNameCompliant(gitSecretNamePrefix)
		secrets = append(secrets, createGitSecret(gitSecretName, ""))
		return secrets
	}

	for _, gitDomain := range gitDomains {
		gitSecretName := fmt.Sprintf("%s-%s", gitSecretNamePrefix, gitDomain)
		gitSecretName = common.MakeStringDNSSubdomainNameCompliant(gitSecretName)
		secrets = append(secrets, createGitSecret(gitSecretName, gitDomain))
	}

	return secrets
}

func createGitSecret(name, gitRepoDomain string) *corev1.Secret {
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

	secret := new(corev1.Secret)
	secret.TypeMeta = metav1.TypeMeta{
		Kind:       string(internaltypes.SecretKind),
		APIVersion: corev1.SchemeGroupVersion.String(),
	}
	secret.ObjectMeta = metav1.ObjectMeta{
		Name: name,
		Annotations: map[string]string{
			"tekton.dev/git-0": gitRepoDomain,
		},
	}
	secret.Type = corev1.SecretTypeSSHAuth
	secret.StringData = map[string]string{
		corev1.SSHAuthPrivateKey: gitPrivateKey,
		"known_hosts":            knownHosts,
	}
	return secret
}

func createTektonTriggersRBAC(roleName string, serviceAccountName string, roleBindingName string) (runtime.Object, runtime.Object, runtime.Object) {
	role := new(rbacv1.Role)
	role.TypeMeta = metav1.TypeMeta{
		Kind:       roleKind,
		APIVersion: rbacv1.SchemeGroupVersion.String(),
	}
	role.ObjectMeta = metav1.ObjectMeta{Name: roleName}
	role.Rules = []rbacv1.PolicyRule{
		rbacv1.PolicyRule{APIGroups: []string{triggersv1alpha1.SchemeGroupVersion.Group}, Resources: []string{"eventlisteners", "triggerbindings", "triggertemplates"}, Verbs: []string{"get"}},
		rbacv1.PolicyRule{APIGroups: []string{v1beta1.SchemeGroupVersion.Group}, Resources: []string{"pipelineruns"}, Verbs: []string{"create"}},
		rbacv1.PolicyRule{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get", "list", "watch"}},
	}

	serviceAccount := new(corev1.ServiceAccount)
	serviceAccount.TypeMeta = metav1.TypeMeta{
		Kind:       rbacv1.ServiceAccountKind,
		APIVersion: corev1.SchemeGroupVersion.String(),
	}
	serviceAccount.ObjectMeta = metav1.ObjectMeta{Name: serviceAccountName}

	roleBinding := new(rbacv1.RoleBinding)
	roleBinding.TypeMeta = metav1.TypeMeta{
		Kind:       roleBindingKind,
		APIVersion: rbacv1.SchemeGroupVersion.String(),
	}
	roleBinding.ObjectMeta = metav1.ObjectMeta{Name: roleBindingName}
	roleBinding.Subjects = []rbacv1.Subject{
		rbacv1.Subject{Kind: rbacv1.ServiceAccountKind, Name: serviceAccountName},
	}
	roleBinding.RoleRef = rbacv1.RoleRef{APIGroup: rbacv1.SchemeGroupVersion.Group, Kind: roleKind, Name: roleName}

	return role, serviceAccount, roleBinding
}

func createCloneBuildPushPipeline(name, workspaceName string, ir irtypes.IR) runtime.Object {
	pipeline := new(v1beta1.Pipeline)
	pipeline.TypeMeta = metav1.TypeMeta{
		Kind:       pipelineKind,
		APIVersion: v1beta1.SchemeGroupVersion.String(),
	}
	pipeline.ObjectMeta = metav1.ObjectMeta{Name: name}
	pipeline.Spec.Params = []v1beta1.ParamSpec{
		v1beta1.ParamSpec{Name: "image-registry-url", Description: "registry-domain/namespace where the output image should be pushed.", Type: v1beta1.ParamTypeString},
	}
	pipeline.Spec.Workspaces = []v1beta1.PipelineWorkspaceDeclaration{
		v1beta1.PipelineWorkspaceDeclaration{Name: workspaceName, Description: "This workspace will receive the cloned git repo and be passed to the kaniko task for building the image."},
	}
	tasks := []v1beta1.PipelineTask{}
	firstTask := true
	prevTaskName := ""
	for i, container := range ir.Containers {
		if !container.New {
			continue
		}
		if container.ContainerBuildType == plantypes.ManualContainerBuildTypeValue || container.ContainerBuildType == plantypes.ReuseContainerBuildTypeValue {
			log.Debugf("Manual or reuse containerization. We will skip this for CICD.")
			continue
		}
		if container.ContainerBuildType == plantypes.DockerFileContainerBuildTypeValue || container.ContainerBuildType == plantypes.ReuseDockerFileContainerBuildTypeValue {
			cloneTaskName := "clone-" + fmt.Sprint(i)
			gitRepoURL := container.RepoInfo.GitRepoURL
			if gitRepoURL == "" {
				gitRepoURL = gitRepoURLPlaceholder
			}
			branchName := container.RepoInfo.GitRepoBranch
			if branchName == "" {
				branchName = defaultGitRepoBranch
			}

			cloneTask := v1beta1.PipelineTask{
				Name:    cloneTaskName,
				TaskRef: &v1beta1.TaskRef{Name: "git-clone"},
				Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
					v1beta1.WorkspacePipelineTaskBinding{Name: "output", Workspace: workspaceName},
				},
				Params: []v1beta1.Param{
					v1beta1.Param{Name: "url", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: gitRepoURL}},
					v1beta1.Param{Name: "revision", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: branchName}},
					v1beta1.Param{Name: "deleteExisting", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "true"}},
				},
			}
			if !firstTask {
				cloneTask.RunAfter = []string{prevTaskName}
			}

			imageName := container.ImageNames[0]
			// Assume there is no git repo. If there is no git repo we can't do CI/CD.
			dockerfilePath := dockerfilePathPlaceholder
			contextPath := contextPathPlaceholder
			// If there is a git repo, set the correct context and dockerfile paths.
			if container.RepoInfo.GitRepoDir != "" {
				relDockerfilePath, err := filepath.Rel(container.RepoInfo.GitRepoDir, container.RepoInfo.TargetPath)
				if err != nil {
					// TODO : Make it a error message after finishing the absolute path refactoring
					log.Debugf("ERROR: Failed to make the path %q relative to the path %q Error %q", container.RepoInfo.GitRepoDir, container.RepoInfo.TargetPath, err)
				} else {
					dockerfilePath = relDockerfilePath
					// We can't figure out the context from the source. So assume the context is the directory containing the dockerfile.
					contextPath = filepath.Dir(relDockerfilePath)
				}
			}

			buildPushTaskName := "build-push-" + fmt.Sprint(i)
			buildPushTask := v1beta1.PipelineTask{
				RunAfter: []string{cloneTaskName},
				Name:     buildPushTaskName,
				TaskRef:  &v1beta1.TaskRef{Name: "kaniko"},
				Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
					v1beta1.WorkspacePipelineTaskBinding{Name: "source", Workspace: workspaceName},
				},
				Params: []v1beta1.Param{
					v1beta1.Param{Name: "IMAGE", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "$(params.image-registry-url)/" + imageName}},
					v1beta1.Param{Name: "DOCKERFILE", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: dockerfilePath}},
					v1beta1.Param{Name: "CONTEXT", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: contextPath}},
				},
			}
			tasks = append(tasks, cloneTask, buildPushTask)
			firstTask = false
			prevTaskName = buildPushTaskName
		} else if container.ContainerBuildType == plantypes.S2IContainerBuildTypeValue {
			log.Debugf("%v not yet supported for Tekton", container.ContainerBuildType)
		} else if container.ContainerBuildType == plantypes.CNBContainerBuildTypeValue {
			log.Debugf("%v not yet supported for Tekton", container.ContainerBuildType)
		} else {
			log.Debugf("Unknown containerization method: %v", container.ContainerBuildType)
		}
	}
	pipeline.Spec.Tasks = tasks
	return pipeline
}

func createClonePushServiceAccount(name string, gitSecrets [](*corev1.Secret), registrySecretName string) runtime.Object {
	serviceAccount := new(corev1.ServiceAccount)
	serviceAccount.TypeMeta = metav1.TypeMeta{
		Kind:       rbacv1.ServiceAccountKind,
		APIVersion: corev1.SchemeGroupVersion.String(),
	}
	serviceAccount.ObjectMeta = metav1.ObjectMeta{Name: name}
	serviceAccount.Secrets = []corev1.ObjectReference{
		corev1.ObjectReference{Name: registrySecretName},
	}
	for _, gitSecret := range gitSecrets {
		serviceAccount.Secrets = append(serviceAccount.Secrets, corev1.ObjectReference{Name: gitSecret.ObjectMeta.Name})
	}

	return serviceAccount
}

func createGitEventIngress(name, gitEventListenerName string) runtime.Object {
	secretName := "<TODO: insert name of TLS secret>"
	hostName := "<TODO: insert subdomain where you want to receive git events>"
	serviceName := "el-" + gitEventListenerName // https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#how-does-the-eventlistener-work
	servicePort := int32(8080)

	ingress := new(networkingv1beta1.Ingress)
	ingress.TypeMeta = metav1.TypeMeta{
		Kind:       ingressKind,
		APIVersion: networkingv1beta1.SchemeGroupVersion.String(),
	}
	ingress.ObjectMeta = metav1.ObjectMeta{Name: name}
	ingress.Spec = networkingv1beta1.IngressSpec{
		TLS: []networkingv1beta1.IngressTLS{
			networkingv1beta1.IngressTLS{Hosts: []string{hostName}, SecretName: secretName},
		},
		Rules: []networkingv1beta1.IngressRule{
			networkingv1beta1.IngressRule{Host: hostName, IngressRuleValue: networkingv1beta1.IngressRuleValue{HTTP: &networkingv1beta1.HTTPIngressRuleValue{
				Paths: []networkingv1beta1.HTTPIngressPath{
					networkingv1beta1.HTTPIngressPath{Backend: networkingv1beta1.IngressBackend{
						ServiceName: serviceName,
						ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: servicePort},
					}},
				},
			}}},
		},
	}

	return ingress
}

func createGitEventTriggerBinding(name string) runtime.Object {
	triggerBinding := new(triggersv1alpha1.TriggerBinding)
	triggerBinding.TypeMeta = metav1.TypeMeta{
		Kind:       string(triggersv1alpha1.NamespacedTriggerBindingKind),
		APIVersion: triggersv1alpha1.SchemeGroupVersion.String(),
	}
	triggerBinding.ObjectMeta = metav1.ObjectMeta{Name: name}
	return triggerBinding
}

func createRegistrySecret(name string) runtime.Object {
	registryURL := "index.docker.io"
	dockerConfigJSON := "<TODO: insert your docker config json>"

	secret := new(corev1.Secret)
	secret.TypeMeta = metav1.TypeMeta{
		Kind:       string(internaltypes.SecretKind),
		APIVersion: corev1.SchemeGroupVersion.String(),
	}
	secret.ObjectMeta = metav1.ObjectMeta{
		Name: name,
		Annotations: map[string]string{
			"tekton.dev/docker-0": registryURL,
		},
	}
	secret.Type = corev1.SecretTypeDockerConfigJson
	secret.StringData = map[string]string{
		corev1.DockerConfigJsonKey: dockerConfigJSON,
	}
	return secret
}

func createGitEventListener(name, serviceAccountName, triggerBindingName, triggerTemplateName string) runtime.Object {
	eventListener := new(triggersv1alpha1.EventListener)
	eventListener.TypeMeta = metav1.TypeMeta{
		Kind:       eventListenerKind,
		APIVersion: triggersv1alpha1.SchemeGroupVersion.String(),
	}
	eventListener.ObjectMeta = metav1.ObjectMeta{Name: name}
	eventListener.Spec = triggersv1alpha1.EventListenerSpec{
		ServiceAccountName: serviceAccountName,
		Triggers: []triggersv1alpha1.EventListenerTrigger{
			triggersv1alpha1.EventListenerTrigger{
				Bindings: []*triggersv1alpha1.EventListenerBinding{
					&triggersv1alpha1.EventListenerBinding{Ref: triggerBindingName},
				},
				Template: &triggersv1alpha1.EventListenerTemplate{Name: triggerTemplateName},
			},
		},
	}
	return eventListener
}

func createTriggerTemplate(name, pipelineName, serviceAccountName, workspaceName string, ir irtypes.IR) runtime.Object {
	storageClassName := "<TODO: insert the storage class you want to use>"
	registryURL := ir.Kubernetes.RegistryURL
	registryNamespace := ir.Kubernetes.RegistryNamespace
	if registryURL == "" {
		registryURL = common.DefaultRegistryURL
	}
	if registryNamespace == "" {
		registryNamespace = "<TODO: insert your registry namespace>"
	}

	// pipelinerun
	pipelineRun := new(v1beta1.PipelineRun)
	pipelineRun.TypeMeta = metav1.TypeMeta{
		Kind:       pipelineRunKind,
		APIVersion: v1beta1.SchemeGroupVersion.String(),
	}
	pipelineRun.ObjectMeta = metav1.ObjectMeta{Name: name}
	pipelineRun.Spec = v1beta1.PipelineRunSpec{
		PipelineRef:        &v1beta1.PipelineRef{Name: pipelineName},
		ServiceAccountName: serviceAccountName,
		Workspaces: []v1beta1.WorkspaceBinding{
			v1beta1.WorkspaceBinding{
				Name: workspaceName,
				VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
					Spec: corev1.PersistentVolumeClaimSpec{
						StorageClassName: &storageClassName,
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources:        corev1.ResourceRequirements{Requests: corev1.ResourceList{"storage": resource.MustParse("1Gi")}},
					},
				},
			},
		},
		Params: []v1beta1.Param{
			v1beta1.Param{Name: "image-registry-url", Value: v1beta1.ArrayOrString{Type: "string", StringVal: registryURL + "/" + registryNamespace}},
		},
	}

	// trigger template
	triggerTemplate := new(triggersv1alpha1.TriggerTemplate)
	triggerTemplate.TypeMeta = metav1.TypeMeta{
		Kind:       triggerTemplateKind,
		APIVersion: triggersv1alpha1.SchemeGroupVersion.String(),
	}
	triggerTemplate.ObjectMeta = metav1.ObjectMeta{Name: name}

	triggerTemplate.Spec = triggersv1alpha1.TriggerTemplateSpec{
		ResourceTemplates: []triggersv1alpha1.TriggerResourceTemplate{
			triggersv1alpha1.TriggerResourceTemplate{
				RawExtension: runtime.RawExtension{Object: pipelineRun},
			},
		},
	}

	return triggerTemplate
}
