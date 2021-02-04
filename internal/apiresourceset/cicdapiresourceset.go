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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/sshkeys"
	"github.com/konveyor/move2kube/internal/transformer/templates"
	irtypes "github.com/konveyor/move2kube/internal/types"
	log "github.com/sirupsen/logrus"
	giturls "github.com/whilp/git-urls"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// CICDAPIResourceSet is the set of CI/CD resources we generate.
type CICDAPIResourceSet struct {
	ExtraFiles map[string]string // file path: file contents
}

// GetScheme returns K8s scheme
func (*CICDAPIResourceSet) GetScheme() *runtime.Scheme {
	return new(K8sAPIResourceSet).GetScheme()
}

// NewCICDAPIResourceSet creates a new CICDAPIResourceSet
func NewCICDAPIResourceSet() *CICDAPIResourceSet {
	return &CICDAPIResourceSet{ExtraFiles: map[string]string{}}
}

const (
	baseBuildConfigName   = "clone-build-push"
	baseWebHookSecretName = "web-hook"
)

// CreateAPIResources converts IR to runtime objects
func (cicdSet *CICDAPIResourceSet) CreateAPIResources(oldir irtypes.IR) []runtime.Object {
	ir := cicdSet.setupEnhancedIR(oldir)
	targetobjs := []runtime.Object{}
	for _, a := range cicdSet.getAPIResources(ir) {
		objs := a.CreateNewResources(ir, []string{string(irtypes.SecretKind)})
		targetobjs = append(targetobjs, objs...)
	}
	return targetobjs
}

func (cicdSet *CICDAPIResourceSet) setupEnhancedIR(oldir irtypes.IR) irtypes.EnhancedIR {
	ir := irtypes.NewEnhancedIRFromIR(oldir)

	// Prefix the project name and make the name a valid k8s name.
	projectName := ir.Name
	p := func(name string) string {
		name = fmt.Sprintf("%s-%s", projectName, name)
		return common.MakeStringDNSSubdomainNameCompliant(name)
	}
	buildConfigNamePrefix := p(baseBuildConfigName)
	gitSecretNamePrefix := p(baseGitSecretName)
	webHookSecretNamePrefix := p(baseWebHookSecretName)

	// Generate secrets for git domains.
	secrets := map[string]irtypes.Storage{}
	ir.Storages = []irtypes.Storage{}
	gitRepoToWebHookURLs := map[string][]string{}
	for _, irContainer := range ir.Containers {
		if len(irContainer.ImageNames) == 0 {
			log.Warnf("the irtypes.Container has no images: %+v", irContainer)
			continue
		}
		imageName := irContainer.ImageNames[0]
		imageStreamName, imageStreamTag := new(apiresource.ImageStream).GetImageStreamNameAndTag(imageName)
		if irContainer.RepoInfo.GitRepoURL == "" {
			// No git repo. Create build config and secrets anyway with placeholders.
			gitDomain := "generic"
			gitSecretName := fmt.Sprintf("%s-%s", gitSecretNamePrefix, gitDomain)
			gitSecretName = common.MakeStringDNSSubdomainNameCompliant(gitSecretName)
			if _, ok := secrets[gitDomain]; !ok {
				secret := cicdSet.createGitSecret(gitSecretName, "")
				secrets[gitDomain] = secret
				ir.Storages = append(ir.Storages, secret)
			}

			webhookSecretName := fmt.Sprintf("%s-%s", webHookSecretNamePrefix, imageName)
			webhookSecretName = common.MakeStringDNSSubdomainNameCompliant(webhookSecretName)
			webhookSecret := cicdSet.createWebHookSecret(webhookSecretName)
			ir.Storages = append(ir.Storages, webhookSecret)

			buildConfigName := fmt.Sprintf("%s-%s", buildConfigNamePrefix, imageName)
			buildConfigName = common.MakeStringDNSSubdomainNameCompliant(buildConfigName)
			ir.BuildConfigs = append(ir.BuildConfigs, irtypes.BuildConfig{
				RepoInfo:          irContainer.RepoInfo,
				Name:              buildConfigName,
				ImageStreamName:   imageStreamName,
				ImageStreamTag:    imageStreamTag,
				SourceSecretName:  gitSecretName,
				WebhookSecretName: webhookSecretName,
			})

			webHookURL := cicdSet.getWebHookURL(buildConfigName, string(webhookSecret.Content["WebHookSecretKey"]), "generic")
			gitRepoToWebHookURLs[gitDomain] = append(gitRepoToWebHookURLs[gitDomain], webHookURL)
		} else {
			gitRepoURL, err := giturls.Parse(irContainer.RepoInfo.GitRepoURL)
			if err != nil {
				log.Warnf("Failed to parse git repo url %s Error: %q", irContainer.RepoInfo.GitRepoURL, err)
				continue
			}
			if gitRepoURL.Hostname() == "" {
				continue
			}

			gitDomain := gitRepoURL.Hostname()
			gitSecretName := fmt.Sprintf("%s-%s", gitSecretNamePrefix, strings.Replace(gitDomain, ".", "-", -1))
			gitSecretName = common.MakeStringDNSSubdomainNameCompliant(gitSecretName)
			if _, ok := secrets[gitDomain]; !ok {
				secret := cicdSet.createGitSecret(gitSecretName, gitDomain)
				secrets[gitDomain] = secret
				ir.Storages = append(ir.Storages, secret)
			}

			webhookSecretName := fmt.Sprintf("%s-%s", webHookSecretNamePrefix, imageName)
			webhookSecretName = common.MakeStringDNSSubdomainNameCompliant(webhookSecretName)
			webhookSecret := cicdSet.createWebHookSecret(webhookSecretName)
			ir.Storages = append(ir.Storages, webhookSecret)

			buildConfigName := fmt.Sprintf("%s-%s", buildConfigNamePrefix, imageName)
			buildConfigName = common.MakeStringDNSSubdomainNameCompliant(buildConfigName)
			ir.BuildConfigs = append(ir.BuildConfigs, irtypes.BuildConfig{
				RepoInfo:          irContainer.RepoInfo,
				Name:              buildConfigName,
				ImageStreamName:   imageStreamName,
				ImageStreamTag:    imageStreamTag,
				SourceSecretName:  gitSecretName,
				WebhookSecretName: webhookSecretName,
			})

			webHookURL := cicdSet.getWebHookURL(buildConfigName, string(webhookSecret.Content["WebHookSecretKey"]), cicdSet.getWebHookType(gitDomain))
			gitRepoToWebHookURLs[irContainer.RepoInfo.GitRepoURL] = append(gitRepoToWebHookURLs[irContainer.RepoInfo.GitRepoURL], webHookURL)
		}
	}
	templateParams := struct {
		IsBuildConfig        bool
		GitRepoToWebHookURLs map[string][]string
	}{
		IsBuildConfig:        true,
		GitRepoToWebHookURLs: gitRepoToWebHookURLs,
	}
	deployCICDScript, err := common.GetStringFromTemplate(templates.DeployCICD_sh, templateParams)
	if err != nil {
		log.Errorf("Failed to fill the template %s with the parameters %+v Error: %q", templates.DeployCICD_sh, templateParams, err)
	} else {
		cicdSet.ExtraFiles["deploy-cicd.sh"] = deployCICDScript
	}
	return ir
}

func (cicdSet *CICDAPIResourceSet) getAPIResources(_ irtypes.EnhancedIR) []apiresource.APIResource {
	return []apiresource.APIResource{
		{
			Scheme:       cicdSet.GetScheme(),
			IAPIResource: &apiresource.BuildConfig{},
		},
		{
			Scheme:       cicdSet.GetScheme(),
			IAPIResource: &apiresource.Storage{},
		},
	}
}

func (*CICDAPIResourceSet) createGitSecret(name, gitRepoDomain string) irtypes.Storage {
	gitPrivateKey := gitPrivateKeyPlaceholder
	if gitRepoDomain != "" {
		if key, ok := sshkeys.GetSSHKey(gitRepoDomain); ok {
			gitPrivateKey = key
		}
	}
	return irtypes.Storage{
		StorageType: irtypes.SecretKind,
		Name:        name,
		SecretType:  core.SecretTypeSSHAuth,
		Content:     map[string][]byte{core.SSHAuthPrivateKey: []byte(gitPrivateKey)},
	}
}

func (cicdSet *CICDAPIResourceSet) createWebHookSecret(name string) irtypes.Storage {
	return irtypes.Storage{
		StorageType: irtypes.SecretKind,
		Name:        name,
		Content:     map[string][]byte{"WebHookSecretKey": []byte(cicdSet.generateWebHookSecretKey())},
	}
}

func (*CICDAPIResourceSet) generateWebHookSecretKey() string {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Warnf("Failed to read random bytes to generate web hook secret key. Error: %q", err)
	}
	return hex.EncodeToString(randomBytes)
}

func (*CICDAPIResourceSet) getWebHookType(gitDomain string) string {
	switch true {
	case strings.Contains(gitDomain, "github"):
		return "github"
	case strings.Contains(gitDomain, "gitlab"):
		return "gitlab"
	case strings.Contains(gitDomain, "bitbucket"):
		return "bitbucket"
	default:
		return "generic"
	}
}

func (*CICDAPIResourceSet) getWebHookURL(buildConfigName, webHookSecretKey, webHookType string) string {
	return "$HOST_AND_PORT/apis/build.openshift.io/v1/namespaces/$NAMESPACE/buildconfigs/" + buildConfigName + "/webhooks/" + webHookSecretKey + "/" + webHookType
}
