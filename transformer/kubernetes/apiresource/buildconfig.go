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

package apiresource

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	okdbuildv1 "github.com/openshift/api/build/v1"
	"github.com/sirupsen/logrus"
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

// getSupportedKinds returns the kinds that this type supports.
func (*BuildConfig) getSupportedKinds() []string {
	return []string{buildConfigKind}
}

// createNewResources creates the runtime objects from the intermediate representation.
func (bc *BuildConfig) createNewResources(ir irtypes.EnhancedIR, _ []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	for _, irBuildConfig := range ir.BuildConfigs {
		buildConfig := bc.createNewResource(irBuildConfig, ir, targetCluster)
		objs = append(objs, &buildConfig)
	}
	return objs
}

func (bc *BuildConfig) createNewResource(irBuildConfig irtypes.BuildConfig, ir irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) okdbuildv1.BuildConfig {
	/*
		apiVersion: build.openshift.io/v1
		kind: BuildConfig
		metadata:
			name: samplenodejs-bc
		spec:
			source:
				type: Git
				git:
					uri: git@gitlab.com:myusername/myrepo.git
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
	contextPath := irBuildConfig.ContainerBuild.ContextPath
	var repoDir, repoURL, repoBranchName string
	var err error
	if contextPath != "" {
		_, repoDir, _, repoURL, repoBranchName, err = common.GatherGitInfo(irBuildConfig.ContainerBuild.ContextPath)
		if err != nil {
			logrus.Debugf("Unable to identify git repo for %s : %s", irBuildConfig.ContainerBuild.ContextPath, err)
		}
	}
	if repoDir != "" {
		relDockerfilePath, err := filepath.Rel(repoDir, contextPath)
		if err != nil {
			logrus.Debugf("Failed to make the path %s relative to the path %s Error %q", contextPath, repoDir, err)
		} else {
			contextPath = filepath.Dir(relDockerfilePath)
		}
	}
	gitRepoURL := gitRepoURLPlaceholder
	if gitRepoURL != "" {
		gitRepoURL = repoURL
	}
	branchName := defaultGitRepoBranch
	if repoBranchName != "" {
		branchName = repoBranchName
	}
	src := okdbuildv1.BuildSource{}
	src.Type = okdbuildv1.BuildSourceGit
	src.Git = &okdbuildv1.GitBuildSource{URI: gitRepoURL, Ref: branchName}
	src.ContextDir = contextPath
	src.SourceSecret = &corev1.LocalObjectReference{Name: irBuildConfig.SourceSecretName}
	return src
}

func (*BuildConfig) getBuildStrategy(irBuildConfig irtypes.BuildConfig, ir irtypes.EnhancedIR) okdbuildv1.BuildStrategy {
	var repoDir string
	var err error
	if irBuildConfig.ContainerBuild.ContextPath != "" {
		_, repoDir, _, _, _, err = common.GatherGitInfo(irBuildConfig.ContainerBuild.ContextPath)
		if err != nil {
			logrus.Debugf("Unable to identify git repo for %s : %s", irBuildConfig.ContainerBuild.ContextPath, err)
		}
	}
	dockerfilePath := dockerfilePathPlaceholder
	if repoDir != "" {
		relDockerfilePath, err := filepath.Rel(repoDir, irBuildConfig.ContainerBuild.ContextPath)
		if err != nil {
			logrus.Debugf("Failed to make the path %s relative to the path %s Error %q", irBuildConfig.ContainerBuild.ContextPath, repoDir, err)
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
	if irBuildConfig.ContainerBuild.ContextPath != "" {
		_, _, _, repoURL, _, _ := common.GatherGitInfo(irBuildConfig.ContainerBuild.ContextPath)
		if repoURL != "" {
			gitRepoURLObj, err := giturls.Parse(repoURL)
			if err != nil {
				if repoURL != "" {
					logrus.Warnf("Failed to parse git repo url %s Error: %q", repoURL, err)
				}
			} else if gitRepoURLObj.Hostname() == "" {
				logrus.Warnf("Successfully parsed git repo url %s but the host name is empty: %+v", repoURL, gitRepoURLObj)
			} else {
				webHookType = bc.getWebHookType(gitRepoURLObj.Hostname())
			}
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
		logrus.Debugf("The git repo %s is not on github, gitlab or bitbucket. Defaulting to generic web hook trigger", gitDomain)
		return okdbuildv1.GenericWebHookBuildTriggerType
	}
}

// convertToClusterSupportedKinds converts the object to supported types if possible.
func (bc *BuildConfig) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if common.IsStringPresent(bc.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}
