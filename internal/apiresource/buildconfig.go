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

package apiresource

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	okdbuildv1 "github.com/openshift/api/build/v1"
	log "github.com/sirupsen/logrus"
	giturls "github.com/whilp/git-urls"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BuildConfig handles all objects like an Openshift BuildConfig.
type BuildConfig struct {
}

const (
	buildConfigKind = "BuildConfig"
)

// GetSupportedKinds returns the kinds that this type supports.
func (*BuildConfig) GetSupportedKinds() []string {
	return []string{buildConfigKind}
}

// CreateNewResources creates the runtime objects from the intermediate representation.
func (bc *BuildConfig) CreateNewResources(ir irtypes.EnhancedIR, _ []string) []runtime.Object {
	objs := []runtime.Object{}
	for _, irBuildConfig := range ir.BuildConfigs {
		buildConfig := bc.createNewResource(irBuildConfig, ir)
		objs = append(objs, &buildConfig)
	}
	return objs
}

func (bc *BuildConfig) createNewResource(irBuildConfig irtypes.BuildConfig, ir irtypes.EnhancedIR) okdbuildv1.BuildConfig {
	/*
		apiVersion: build.openshift.io/v1
		kind: BuildConfig
		metadata:
			name: samplenodejs-bc
		spec:
			source:
				type: Git
				git:
					uri: git@gitlab.com:hari.balagopal/samplenodejs.git
					ref: splittest1
				sourceSecret:
					name: gitlab-secret
			strategy:
				type: Docker
			output:
				to:
					kind: ImageStreamTag
					name: samplenodejs-image-stream:foo
			triggers:
				-	type: GitLab
					gitlab:
						secretReference:
							name: samplenodejs-webhook-secret
	*/
	buildConfig := okdbuildv1.BuildConfig{}
	buildConfig.TypeMeta = metav1.TypeMeta{
		Kind:       buildConfigKind,
		APIVersion: okdbuildv1.SchemeGroupVersion.String(),
	}
	buildConfig.ObjectMeta.Name = irBuildConfig.Name
	buildConfig.Spec.Source = bc.getBuildSource(irBuildConfig, ir)
	buildConfig.Spec.Strategy = bc.getBuildStrategy(irBuildConfig, ir)
	buildConfig.Spec.Output.To = &corev1.ObjectReference{
		Kind: "ImageStreamTag",
		Name: fmt.Sprintf("%s:%s", irBuildConfig.ImageStreamName, irBuildConfig.ImageStreamTag),
	}
	buildConfig.Spec.Triggers = []okdbuildv1.BuildTriggerPolicy{bc.getBuildTriggerPolicy(irBuildConfig, ir)}
	return buildConfig
}

func (*BuildConfig) getBuildSource(irBuildConfig irtypes.BuildConfig, ir irtypes.EnhancedIR) okdbuildv1.BuildSource {
	contextPath := contextPathPlaceholder
	if irBuildConfig.RepoInfo.GitRepoDir != "" {
		relDockerfilePath, err := filepath.Rel(irBuildConfig.RepoInfo.GitRepoDir, irBuildConfig.RepoInfo.TargetPath)
		if err != nil {
			log.Debugf("Failed to make the path %s relative to the path %s Error %q", irBuildConfig.RepoInfo.TargetPath, irBuildConfig.RepoInfo.GitRepoDir, err)
		} else {
			contextPath = filepath.Dir(relDockerfilePath)
		}
	}
	gitRepoURL := gitRepoURLPlaceholder
	if irBuildConfig.RepoInfo.GitRepoURL != "" {
		gitRepoURL = irBuildConfig.RepoInfo.GitRepoURL
	}
	branchName := defaultGitRepoBranch
	if irBuildConfig.RepoInfo.GitRepoBranch != "" {
		branchName = irBuildConfig.RepoInfo.GitRepoBranch
	}
	src := okdbuildv1.BuildSource{}
	src.Type = okdbuildv1.BuildSourceGit
	src.Git = &okdbuildv1.GitBuildSource{URI: gitRepoURL, Ref: branchName}
	src.ContextDir = contextPath
	src.SourceSecret = &corev1.LocalObjectReference{Name: irBuildConfig.SourceSecretName}
	return src
}

func (*BuildConfig) getBuildStrategy(irBuildConfig irtypes.BuildConfig, ir irtypes.EnhancedIR) okdbuildv1.BuildStrategy {
	dockerfilePath := dockerfilePathPlaceholder
	if irBuildConfig.RepoInfo.GitRepoDir != "" {
		relDockerfilePath, err := filepath.Rel(irBuildConfig.RepoInfo.GitRepoDir, irBuildConfig.RepoInfo.TargetPath)
		if err != nil {
			log.Debugf("Failed to make the path %s relative to the path %s Error %q", irBuildConfig.RepoInfo.TargetPath, irBuildConfig.RepoInfo.GitRepoDir, err)
		} else {
			dockerfilePath = relDockerfilePath
		}
	}
	strategy := okdbuildv1.BuildStrategy{}
	strategy.Type = okdbuildv1.DockerBuildStrategyType
	strategy.DockerStrategy = &okdbuildv1.DockerBuildStrategy{DockerfilePath: dockerfilePath}
	return strategy
}

func (bc *BuildConfig) getBuildTriggerPolicy(irBuildConfig irtypes.BuildConfig, ir irtypes.EnhancedIR) okdbuildv1.BuildTriggerPolicy {
	webHookType := okdbuildv1.GenericWebHookBuildTriggerType
	if irBuildConfig.RepoInfo.GitRepoURL != "" {
		gitRepoURLObj, err := giturls.Parse(irBuildConfig.RepoInfo.GitRepoURL)
		if err != nil {
			log.Warnf("Failed to parse git repo url %s Error: %q", irBuildConfig.RepoInfo.GitRepoURL, err)
		} else if gitRepoURLObj.Hostname() == "" {
			log.Warnf("Successfully parsed git repo url %s but the host name is empty: %+v", irBuildConfig.RepoInfo.GitRepoURL, gitRepoURLObj)
		} else {
			webHookType = bc.getWebHookType(gitRepoURLObj.Hostname())
		}
	}
	policy := okdbuildv1.BuildTriggerPolicy{Type: webHookType}
	webHookTrigger := okdbuildv1.WebHookTrigger{SecretReference: &okdbuildv1.SecretLocalReference{Name: irBuildConfig.WebhookSecretName}}
	switch policy.Type {
	case okdbuildv1.GitHubWebHookBuildTriggerType:
		policy.GitHubWebHook = &webHookTrigger
	case okdbuildv1.GitLabWebHookBuildTriggerType:
		policy.GitLabWebHook = &webHookTrigger
	case okdbuildv1.BitbucketWebHookBuildTriggerType:
		policy.BitbucketWebHook = &webHookTrigger
	default:
		log.Warnf("the git repo is not on github, gitlab or bitbucket. Using generic webhook")
		policy.GenericWebHook = &webHookTrigger
	}
	return policy
}

func (*BuildConfig) getWebHookType(gitDomain string) okdbuildv1.BuildTriggerType {
	switch true {
	case strings.Contains(gitDomain, "github"):
		return okdbuildv1.GitHubWebHookBuildTriggerType
	case strings.Contains(gitDomain, "gitlab"):
		return okdbuildv1.GitLabWebHookBuildTriggerType
	case strings.Contains(gitDomain, "bitbucket"):
		return okdbuildv1.BitbucketWebHookBuildTriggerType
	default:
		log.Debugf("The git repo %s is not on github, gitlab or bitbucket. Defaulting to generic web hook trigger", gitDomain)
		return okdbuildv1.GenericWebHookBuildTriggerType
	}
}

// ConvertToClusterSupportedKinds converts the object to supported types if possible.
func (bc *BuildConfig) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR) ([]runtime.Object, bool) {
	supKinds := bc.GetSupportedKinds()
	for _, supKind := range supKinds {
		if common.IsStringPresent(supportedKinds, supKind) {
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}
