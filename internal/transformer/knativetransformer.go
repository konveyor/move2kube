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
	RootDir                string
	TransformedObjects     []runtime.Object
	Containers             []irtypes.Container
	Values                 outputtypes.HelmValues
	TargetClusterSpec      collecttypes.ClusterMetadataSpec
	Name                   string
	IgnoreUnsupportedKinds bool
	AddCopySources         bool
}

// Transform translates intermediate representation to destination objects
func (kt *KnativeTransformer) Transform(ir irtypes.IR) error {
	log.Debugf("Starting Knative transform")
	log.Debugf("Total services to be transformed : %d", len(ir.Services))

	kt.Name = ir.Name
	kt.Values = ir.Values
	kt.Containers = ir.Containers
	kt.TargetClusterSpec = ir.TargetClusterSpec
	kt.IgnoreUnsupportedKinds = ir.Kubernetes.IgnoreUnsupportedKinds
	kt.TransformedObjects = convertIRToObjects(irtypes.NewEnhancedIRFromIR(ir), kt.getAPIResources())
	kt.RootDir = ir.RootDir
	kt.AddCopySources = ir.AddCopySources
	log.Debugf("Total transformed objects : %d", len(kt.TransformedObjects))

	return nil
}

func (kt *KnativeTransformer) getAPIResources() []apiresource.IAPIResource {
	return []apiresource.IAPIResource{&apiresource.KnativeService{}}
}

// WriteObjects writes Transformed objects to filesystem
func (kt *KnativeTransformer) WriteObjects(outpath string, transformPaths []string) error {
	areNewImagesCreated := writeContainers(kt.Containers, outpath, kt.RootDir, kt.Values.RegistryURL, kt.Values.RegistryNamespace, kt.AddCopySources)

	artifactspath := filepath.Join(outpath, kt.Name)
	log.Debugf("Total services to be serialized : %d", len(kt.TransformedObjects))

	_, err := writeTransformedObjects(artifactspath, kt.TransformedObjects, kt.TargetClusterSpec, kt.IgnoreUnsupportedKinds, transformPaths)
	if err != nil {
		log.Errorf("Error occurred while writing transformed objects %s", err)
	}
	kt.writeDeployScript(kt.Name, outpath)
	kt.writeReadeMe(kt.Name, areNewImagesCreated, kt.AddCopySources, outpath)
	return nil
}

func (kt *KnativeTransformer) writeDeployScript(proj string, outpath string) {
	scriptspath := filepath.Join(outpath, common.ScriptsDir)
	err := os.MkdirAll(scriptspath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create directory %s : %s", scriptspath, err)
	}
	err = common.WriteTemplateToFile(templates.Deploy_sh, struct {
		Project string
	}{
		Project: proj,
	}, filepath.Join(scriptspath, "deploy.sh"), common.DefaultExecutablePermission)
	if err != nil {
		log.Errorf("Unable to write deploy script : %s", err)
	}
}

func (kt *KnativeTransformer) writeReadeMe(project string, areNewImages bool, addCopySources bool, outpath string) {
	err := common.WriteTemplateToFile(templates.KnativeReadme_md, struct {
		Project        string
		NewImages      bool
		AddCopySources bool
	}{
		Project:        project,
		NewImages:      areNewImages,
		AddCopySources: addCopySources,
	}, filepath.Join(outpath, "Readme.md"), common.DefaultFilePermission)
	if err != nil {
		log.Errorf("Unable to write Readme : %s", err)
	}
}
