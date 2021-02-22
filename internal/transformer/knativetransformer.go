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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/transformer/templates"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	outputtypes "github.com/konveyor/move2kube/types/output"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

// KnativeTransformer implements Transformer interface
type KnativeTransformer struct {
	RootDir            string
	TransformedObjects []runtime.Object
	Containers         []irtypes.Container
	Values             outputtypes.HelmValues
	TargetClusterSpec  collecttypes.ClusterMetadataSpec
	Name               string
}

// Transform translates intermediate representation to destination objects
func (kt *KnativeTransformer) Transform(ir irtypes.IR) error {
	log.Debugf("Starting Knative transform")
	log.Debugf("Total services to be transformed : %d", len(ir.Services))

	kt.Name = ir.Name
	kt.Containers = ir.Containers
	kt.TargetClusterSpec = ir.TargetClusterSpec
	kt.TransformedObjects = convertIRToObjects(irtypes.NewEnhancedIRFromIR(ir), kt.getAPIResources())
	kt.RootDir = ir.RootDir
	log.Debugf("Total transformed objects : %d", len(kt.TransformedObjects))

	return nil
}

func (kt *KnativeTransformer) getAPIResources() []apiresource.IAPIResource {
	return []apiresource.IAPIResource{new(apiresource.KnativeService)}
}

// WriteObjects writes the transformed knative resources to files
func (kt *KnativeTransformer) WriteObjects(outputPath string, transformPaths []string) error {
	artifactspath := filepath.Join(outputPath, common.DeployDir, "knative")
	log.Debugf("Total services to be serialized : %d", len(kt.TransformedObjects))
	if _, err := writeTransformedObjects(artifactspath, kt.TransformedObjects, kt.TargetClusterSpec, transformPaths); err != nil {
		log.Errorf("Error occurred while writing knative transformed objects. Error: %q", err)
	}
	kt.writeDeployScript(kt.Name, outputPath)
	return nil
}

func (kt *KnativeTransformer) writeDeployScript(proj string, outpath string) {
	scriptspath := filepath.Join(outpath, common.ScriptsDir)
	if err := os.MkdirAll(scriptspath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Unable to create directory %s : %s", scriptspath, err)
	}
	deployKnativeScriptPath := filepath.Join(scriptspath, "deployknative.sh")
	if err := ioutil.WriteFile(deployKnativeScriptPath, []byte(templates.DeployKnative_sh), common.DefaultExecutablePermission); err != nil {
		log.Errorf("Failed to write the deploy script at path %s . Error: %q", deployKnativeScriptPath, err)
	}
}
