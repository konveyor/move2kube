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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/common/sshkeys"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/transformer/kubernetes/apiresource"
	"github.com/konveyor/move2kube/transformer/kubernetes/irpreprocessor"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// BuildConfig implements Transformer interface
type BuildConfig struct {
	Config            transformertypes.Transformer
	Env               *environment.Environment
	BuildConfigConfig *BuildConfigYamlConfig
}

// BuildConfigYamlConfig stores the BuildConfig related information
type BuildConfigYamlConfig struct {
	OutputPath              string `yaml:"outputPath"`
	SetDefaultValuesInYamls bool   `yaml:"setDefaultValuesInYamls"`
}

const (
	defaultBuildConfigYamlsOutputPath = common.DeployDir + string(os.PathSeparator) + common.CICDDir + string(os.PathSeparator) + "buildconfig"
	// BuildConfigArtifacts stores the BuildConfig Artifact Name
	BuildConfigArtifacts transformertypes.ArtifactType = "BuildConfigYamls"
)

const (
	baseBuildConfigName   = "clone-build-push"
	baseWebHookSecretName = "web-hook"
)

// Init initializes the transformer
func (t *BuildConfig) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.Env = env
	t.BuildConfigConfig = &BuildConfigYamlConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.BuildConfigConfig); err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.BuildConfigConfig, err)
		return err
	}
	if t.BuildConfigConfig.OutputPath == "" {
		t.BuildConfigConfig.OutputPath = defaultBuildConfigYamlsOutputPath
	}
	if !t.BuildConfigConfig.SetDefaultValuesInYamls {
		t.BuildConfigConfig.SetDefaultValuesInYamls = setDefaultValuesInYamls
	}
	return nil
}

// GetConfig returns the transformer config
func (t *BuildConfig) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *BuildConfig) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *BuildConfig) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	logrus.Debugf("Translating IR using Buildconfig transformer")
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
		if !(len(clusterConfig.Spec.GetSupportedVersions("BuildConfig")) > 0) {
			logrus.Debugf("BuildConfig was not found on the target cluster.")
			return nil, nil, nil
		}
		apis := []apiresource.IAPIResource{new(apiresource.BuildConfig), new(apiresource.Storage)}
		deployCICDDir := t.BuildConfigConfig.OutputPath
		tempDest := filepath.Join(t.Env.TempPath, deployCICDDir)
		logrus.Infof("Generating Buildconfig pipeline for CI/CD")
		enhancedIR := t.setupEnhancedIR(ir, t.Env.GetProjectName())
		files, err := apiresource.TransformIRAndPersist(enhancedIR, tempDest, apis, clusterConfig, t.BuildConfigConfig.SetDefaultValuesInYamls)
		if err != nil {
			logrus.Errorf("Unable to transform and persist IR : %s", err)
			return nil, nil, err
		}
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
		a := transformertypes.Artifact{
			Name: t.Config.Name,
			Type: artifacts.KubernetesYamlsArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.KubernetesYamlsPathType: {deployCICDDir},
			},
		}
		createdArtifacts = append(createdArtifacts, a)
		logrus.Debugf("Total transformed objects : %d", len(files))
	}
	return pathMappings, createdArtifacts, nil
}

// setupEnhancedIR return enhanced IR used by BuildConfig
func (t *BuildConfig) setupEnhancedIR(oldir irtypes.IR, planName string) irtypes.EnhancedIR {
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
		if irContainer.Build.ContextPath == "" {
			continue
		}
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
