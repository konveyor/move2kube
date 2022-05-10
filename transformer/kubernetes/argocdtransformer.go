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
	"fmt"
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/transformer/kubernetes/apiresource"
	"github.com/konveyor/move2kube/transformer/kubernetes/irpreprocessor"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// ArgoCD implements Transformer interface
type ArgoCD struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	ArgoCDConfig *ArgoCDYamlConfig
}

// ArgoCDYamlConfig stores the ArgoCD related information
type ArgoCDYamlConfig struct {
	OutputPath string `yaml:"outputPath"`
}

const (
	defaultArgoCDYamlsOutputPath = common.DeployDir + string(os.PathSeparator) + common.CICDDir + string(os.PathSeparator) + "argocd"
	baseAppName                  = "argo-app"
)

func (t *ArgoCD) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.Env = env
	t.ArgoCDConfig = &ArgoCDYamlConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.ArgoCDConfig); err != nil {
		return fmt.Errorf("unable to load the config for the ArgoCD tranformer. Actual: %+v . Error: %q", t.Config.Spec.Config, err)
	}
	if t.ArgoCDConfig.OutputPath == "" {
		t.ArgoCDConfig.OutputPath = defaultArgoCDYamlsOutputPath
	}
	return nil
}

func (t *ArgoCD) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

func (*ArgoCD) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	return nil, nil
}

func (t *ArgoCD) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	logrus.Tracef("ArgoCD transformer Transform start")
	defer logrus.Tracef("ArgoCD transformer Transform end")

	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}

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
		resources := []apiresource.IAPIResource{
			new(apiresource.ArgoCDApplication),
		}
		deployCICDDir := t.ArgoCDConfig.OutputPath
		tempDest := filepath.Join(t.Env.TempPath, deployCICDDir)
		logrus.Debugf("Generating ArgoCD yamls for CI/CD")
		enhancedIR := t.setupEnhancedIR(ir, t.Env.GetProjectName())
		files, err := apiresource.TransformIRAndPersist(enhancedIR, tempDest, resources, clusterConfig)
		if err != nil {
			logrus.Errorf("failed to transform and persist IR. Error: %q", err)
			continue
		}
		for _, file := range files {
			destPath, err := filepath.Rel(t.Env.TempPath, file)
			if err != nil {
				logrus.Errorf("failed to make the yaml path %s relative to the temporary directory %s . Error: %q", file, t.Env.TempPath, err)
				continue
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.DefaultPathMappingType,
				SrcPath:  file,
				DestPath: destPath,
			})
		}
		createdArtifact := transformertypes.Artifact{
			Name: t.Config.Name,
			Type: artifacts.KubernetesYamlsArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.KubernetesYamlsPathType: {deployCICDDir},
			},
		}
		createdArtifacts = append(createdArtifacts, createdArtifact)
		logrus.Debugf("ArgoCD generated %d new objects", len(files))
	}

	return pathMappings, createdArtifacts, nil
}

// setupEnhancedIR returns EnhancedIR containing ArgoCD components
func (t *ArgoCD) setupEnhancedIR(oldir irtypes.IR, projectName string) irtypes.EnhancedIR {
	ir := irtypes.NewEnhancedIRFromIR(oldir)
	// Prefix the project name and make the name a valid k8s name.
	p := func(baseName string) string {
		r := common.GetRandomString()
		return common.MakeStringDNSSubdomainNameCompliant(fmt.Sprintf("%s-%s-%s", projectName, baseName, r))
	}
	appName := p(baseAppName)
	ir.ArgoCDResources = irtypes.ArgoCDResources{
		Applications: []irtypes.Application{{Name: appName}},
	}
	return ir
}
