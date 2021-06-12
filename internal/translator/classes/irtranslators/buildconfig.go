/*
Copyright IBM Corporation 2021

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

package irtranslators

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/sshkeys"
	"github.com/konveyor/move2kube/internal/configuration"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

const (
	BuildConfigArtifacts = "BuildConfigYamls"
)

const (
	baseBuildConfigName   = "clone-build-push"
	baseWebHookSecretName = "web-hook"
)

// BuildConfig implements Translator interface
type BuildConfig struct {
	Config translatortypes.Translator
}

type BuildConfigConfig struct {
}

func (t *BuildConfig) Init(tc translatortypes.Translator) error {
	t.Config = tc
	return nil
}

func (t *BuildConfig) GetConfig() translatortypes.Translator {
	return t.Config
}

func (t *BuildConfig) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *BuildConfig) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *BuildConfig) KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *BuildConfig) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *BuildConfig) PlanDetect(plantypes.Plan) ([]plantypes.Translator, error) {
	ts := []plantypes.Translator{{
		Mode:          plantypes.ModeContainer,
		ArtifactTypes: []string{BuildConfigArtifacts},
	}}
	return ts, nil
}

func (t *BuildConfig) TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.Patch, error) {
	return nil, nil
}

func (t *BuildConfig) TranslateIR(ir irtypes.IR, plan plantypes.Plan, tempOutputDir string) (pathMappings []translatortypes.PathMapping, err error) {
	targetCluster, err := new(configuration.ClusterMDLoader).GetTargetClusterMetadataForPlan(plan)
	if err != nil {
		err := fmt.Errorf("unable to find target cluster : %+v", plan.Spec.TargetCluster)
		logrus.Errorf("%s", err)
		return nil, err
	}
	if !(len(targetCluster.Spec.GetSupportedVersions("BuildConfig")) > 0) {
		logrus.Debugf("BuildConfig was not found on the target cluster.")
		return nil, nil
	}
	apis := []apiresource.IAPIResource{&apiresource.BuildConfig{}, &apiresource.Storage{}}
	tempDest := filepath.Join(tempOutputDir, common.DeployDir, common.CICDDir, "buildconfig")
	logrus.Infof("Generating Tekton pipeline for CI/CD")
	enhancedIR := t.SetupEnhancedIR(ir)
	if files, err := apiresource.TransformAndPersist(enhancedIR, tempDest, apis, targetCluster); err == nil {
		for _, f := range files {
			if destPath, err := filepath.Rel(tempOutputDir, f); err != nil {
				logrus.Errorf("Invalid yaml path : %s", destPath)
			} else {
				pathMappings = append(pathMappings, translatortypes.PathMapping{
					Type:     translatortypes.DefaultPathMappingType,
					SrcPath:  f,
					DestPath: destPath,
				})
			}
		}
		logrus.Debugf("Total transformed objects : %d", len(files))
	} else {
		logrus.Errorf("Unable to translate and persist IR : %s", err)
		return nil, err
	}
	return pathMappings, nil
}

// SetupEnhancedIR return enhanced IR used by BuildConfig
func (t *BuildConfig) SetupEnhancedIR(oldir irtypes.IR) irtypes.EnhancedIR {
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
	for imageName, irContainer := range ir.ContainerImages {
		imageStreamName, imageStreamTag := new(apiresource.ImageStream).GetImageStreamNameAndTag(imageName)
		_, _, gitHostName, gitURL, _, _ := common.GatherGitInfo(irContainer.Build.ContextPath)
		if gitURL == "" {
			// No git repo. Create build config and secrets anyway with placeholders.
			gitDomain := "generic"
			gitSecretName := fmt.Sprintf("%s-%s", gitSecretNamePrefix, gitDomain)
			gitSecretName = common.MakeStringDNSSubdomainNameCompliant(gitSecretName)
			if _, ok := secrets[gitDomain]; !ok {
				secret := t.createGitSecret(gitSecretName, "")
				secrets[gitDomain] = secret
				ir.Storages = append(ir.Storages, secret)
			}
			webhookSecretName := fmt.Sprintf("%s-%s", webHookSecretNamePrefix, imageName)
			webhookSecretName = common.MakeStringDNSSubdomainNameCompliant(webhookSecretName)
			webhookSecret := t.createWebHookSecret(webhookSecretName)
			ir.Storages = append(ir.Storages, webhookSecret)

			buildConfigName := fmt.Sprintf("%s-%s", buildConfigNamePrefix, imageName)
			buildConfigName = common.MakeStringDNSSubdomainNameCompliant(buildConfigName)
			ir.BuildConfigs = append(ir.BuildConfigs, irtypes.BuildConfig{
				Name:              buildConfigName,
				ImageStreamName:   imageStreamName,
				ImageStreamTag:    imageStreamTag,
				SourceSecretName:  gitSecretName,
				WebhookSecretName: webhookSecretName,
			})

			webHookURL := t.getWebHookURL(buildConfigName, string(webhookSecret.Content["WebHookSecretKey"]), "generic")
			gitRepoToWebHookURLs[gitDomain] = append(gitRepoToWebHookURLs[gitDomain], webHookURL)
		} else {
			if gitHostName == "" {
				continue
			}

			gitSecretName := fmt.Sprintf("%s-%s", gitSecretNamePrefix, strings.Replace(gitHostName, ".", "-", -1))
			gitSecretName = common.MakeStringDNSSubdomainNameCompliant(gitSecretName)
			if _, ok := secrets[gitHostName]; !ok {
				secret := t.createGitSecret(gitSecretName, gitHostName)
				secrets[gitHostName] = secret
				ir.Storages = append(ir.Storages, secret)
			}

			webhookSecretName := fmt.Sprintf("%s-%s", webHookSecretNamePrefix, imageName)
			webhookSecretName = common.MakeStringDNSSubdomainNameCompliant(webhookSecretName)
			webhookSecret := t.createWebHookSecret(webhookSecretName)
			ir.Storages = append(ir.Storages, webhookSecret)

			buildConfigName := fmt.Sprintf("%s-%s", buildConfigNamePrefix, imageName)
			buildConfigName = common.MakeStringDNSSubdomainNameCompliant(buildConfigName)
			ir.BuildConfigs = append(ir.BuildConfigs, irtypes.BuildConfig{
				Name:              buildConfigName,
				ImageStreamName:   imageStreamName,
				ImageStreamTag:    imageStreamTag,
				SourceSecretName:  gitSecretName,
				WebhookSecretName: webhookSecretName,
			})

			webHookURL := t.getWebHookURL(buildConfigName, string(webhookSecret.Content["WebHookSecretKey"]), t.getWebHookType(gitHostName))
			gitRepoToWebHookURLs[gitURL] = append(gitRepoToWebHookURLs[gitURL], webHookURL)
		}
	}
	return ir
}

func (t *BuildConfig) createGitSecret(name, gitRepoDomain string) irtypes.Storage {
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

func (t *BuildConfig) createWebHookSecret(name string) irtypes.Storage {
	return irtypes.Storage{
		StorageType: irtypes.SecretKind,
		Name:        name,
		Content:     map[string][]byte{"WebHookSecretKey": []byte(t.generateWebHookSecretKey())},
	}
}

func (t *BuildConfig) generateWebHookSecretKey() string {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		logrus.Warnf("Failed to read random bytes to generate web hook secret key. Error: %q", err)
	}
	return hex.EncodeToString(randomBytes)
}

func (t *BuildConfig) getWebHookType(gitDomain string) string {
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

func (t *BuildConfig) getWebHookURL(buildConfigName, webHookSecretKey, webHookType string) string {
	return "$HOST_AND_PORT/apis/build.openshift.io/v1/namespaces/$NAMESPACE/buildconfigs/" + buildConfigName + "/webhooks/" + webHookSecretKey + "/" + webHookType
}
