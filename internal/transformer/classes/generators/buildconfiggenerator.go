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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/sshkeys"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
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

// BuildConfig implements Transformer interface
type BuildConfig struct {
	Config transformertypes.Transformer
	Env    environment.Environment
}

type BuildConfigConfig struct {
}

func (t *BuildConfig) Init(tc transformertypes.Transformer, env environment.Environment) error {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *BuildConfig) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.Config, t.Env
}

func (t *BuildConfig) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

func (t *BuildConfig) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

func (t *BuildConfig) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	logrus.Debugf("Translating IR using Buildconfig transformer")
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
		if !(len(pC.TargetCluster.Spec.GetSupportedVersions("BuildConfig")) > 0) {
			logrus.Debugf("BuildConfig was not found on the target cluster.")
			return nil, nil, nil
		}
		apis := []apiresource.IAPIResource{&apiresource.BuildConfig{}, &apiresource.Storage{}}
		tempDest := filepath.Join(t.Env.TempPath, common.DeployDir, common.CICDDir, "buildconfig")
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

// SetupEnhancedIR return enhanced IR used by BuildConfig
func (t *BuildConfig) SetupEnhancedIR(oldir irtypes.IR, planName string) irtypes.EnhancedIR {
	ir := irtypes.NewEnhancedIRFromIR(oldir)

	// Prefix the project name and make the name a valid k8s name.
	projectName := planName
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
