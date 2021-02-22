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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/sshkeys"
	"github.com/konveyor/move2kube/internal/transformer/templates"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	log "github.com/sirupsen/logrus"
	giturls "github.com/whilp/git-urls"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// BuildconfigTransformer is the set of CI/CD resources we generate.
type BuildconfigTransformer struct {
	shouldRun                     bool
	transformedBuildConfigObjects []runtime.Object
	TargetClusterSpec             collecttypes.ClusterMetadataSpec
	extraFiles                    map[string]string // file path: file contents
}

const (
	baseBuildConfigName   = "clone-build-push"
	baseWebHookSecretName = "web-hook"
)

// NewBuildconfigTransformer creates a new CICDAPIResourceSet
func NewBuildconfigTransformer() *BuildconfigTransformer {
	return &BuildconfigTransformer{extraFiles: map[string]string{}}
}

// Transform translates intermediate representation to destination objects
func (bcTransformer *BuildconfigTransformer) Transform(ir irtypes.IR) error {
	bcTransformer.shouldRun = false
	if !ir.TargetClusterSpec.IsBuildConfigSupported() {
		log.Debugf("BuildConfig was not found on the target cluster.")
		return nil
	}
	for _, container := range ir.Containers {
		if container.New {
			bcTransformer.shouldRun = true
			break
		}
	}
	if !bcTransformer.shouldRun {
		return nil
	}
	bcTransformer.TargetClusterSpec = ir.TargetClusterSpec
	// BuildConfig (Openshift)
	log.Infof("The target cluster has support for BuildConfig, also generating build configs for CI/CD")
	bcTransformer.transformedBuildConfigObjects = convertIRToObjects(bcTransformer.SetupEnhancedIR(ir), bcTransformer.GetAPIResources())
	return nil
}

// WriteObjects writes Transformed objects to filesystem. Also does some final transformations on the generated yamls.
func (bcTransformer *BuildconfigTransformer) WriteObjects(outputPath string, transformPaths []string) error {
	if !bcTransformer.shouldRun {
		return nil
	}
	cicdPath := filepath.Join(outputPath, common.DeployDir, "cicd")
	// deploy/cicd/buildconfig/
	bcPath := filepath.Join(cicdPath, "buildconfig")
	if _, err := writeTransformedObjects(bcPath, bcTransformer.transformedBuildConfigObjects, bcTransformer.TargetClusterSpec, transformPaths); err != nil {
		log.Errorf("Error occurred while writing transformed objects. Error: %q", err)
		return err
	}
	if len(bcTransformer.extraFiles) == 0 {
		return nil
	}
	for relFilePath, fileContents := range bcTransformer.extraFiles {
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

// GetAPIResources returns api resources related to buildconfig
func (*BuildconfigTransformer) GetAPIResources() []apiresource.IAPIResource {
	return []apiresource.IAPIResource{&apiresource.BuildConfig{}, &apiresource.Storage{}}
}

// SetupEnhancedIR return enhanced IR used by BuildConfig
func (bcTransformer *BuildconfigTransformer) SetupEnhancedIR(oldir irtypes.IR) irtypes.EnhancedIR {
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
				secret := bcTransformer.createGitSecret(gitSecretName, "")
				secrets[gitDomain] = secret
				ir.Storages = append(ir.Storages, secret)
			}

			webhookSecretName := fmt.Sprintf("%s-%s", webHookSecretNamePrefix, imageName)
			webhookSecretName = common.MakeStringDNSSubdomainNameCompliant(webhookSecretName)
			webhookSecret := bcTransformer.createWebHookSecret(webhookSecretName)
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

			webHookURL := bcTransformer.getWebHookURL(buildConfigName, string(webhookSecret.Content["WebHookSecretKey"]), "generic")
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
				secret := bcTransformer.createGitSecret(gitSecretName, gitDomain)
				secrets[gitDomain] = secret
				ir.Storages = append(ir.Storages, secret)
			}

			webhookSecretName := fmt.Sprintf("%s-%s", webHookSecretNamePrefix, imageName)
			webhookSecretName = common.MakeStringDNSSubdomainNameCompliant(webhookSecretName)
			webhookSecret := bcTransformer.createWebHookSecret(webhookSecretName)
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

			webHookURL := bcTransformer.getWebHookURL(buildConfigName, string(webhookSecret.Content["WebHookSecretKey"]), bcTransformer.getWebHookType(gitDomain))
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
		bcTransformer.extraFiles[filepath.Join(common.ScriptsDir, "deploy-cicd.sh")] = deployCICDScript
	}
	return ir
}

func (*BuildconfigTransformer) createGitSecret(name, gitRepoDomain string) irtypes.Storage {
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

func (bcTransformer *BuildconfigTransformer) createWebHookSecret(name string) irtypes.Storage {
	return irtypes.Storage{
		StorageType: irtypes.SecretKind,
		Name:        name,
		Content:     map[string][]byte{"WebHookSecretKey": []byte(bcTransformer.generateWebHookSecretKey())},
	}
}

func (*BuildconfigTransformer) generateWebHookSecretKey() string {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Warnf("Failed to read random bytes to generate web hook secret key. Error: %q", err)
	}
	return hex.EncodeToString(randomBytes)
}

func (*BuildconfigTransformer) getWebHookType(gitDomain string) string {
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

func (*BuildconfigTransformer) getWebHookURL(buildConfigName, webHookSecretKey, webHookType string) string {
	return "$HOST_AND_PORT/apis/build.openshift.io/v1/namespaces/$NAMESPACE/buildconfigs/" + buildConfigName + "/webhooks/" + webHookSecretKey + "/" + webHookType
}
