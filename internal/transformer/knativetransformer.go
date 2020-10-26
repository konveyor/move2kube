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
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/konveyor/move2kube/internal/apiresourceset"
	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/transformer/templates"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	outputtypes "github.com/konveyor/move2kube/types/output"
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
	kt.TransformedObjects = (&apiresourceset.KnativeAPIResourceSet{}).CreateAPIResources(ir)
	kt.RootDir = ir.RootDir

	log.Debugf("Total transformed objects : %d", len(kt.TransformedObjects))

	return nil
}

// WriteObjects writes Transformed objects to filesystem
func (kt *KnativeTransformer) WriteObjects(outpath string) error {
	areNewImagesCreated := writeContainers(kt.Containers, outpath, kt.RootDir, kt.Values.RegistryURL, kt.Values.RegistryNamespace)

	artifactspath := filepath.Join(outpath, kt.Name)
	log.Debugf("Total services to be serialized : %d", len(kt.TransformedObjects))

	_, err := writeTransformedObjects(artifactspath, kt.TransformedObjects)
	if err != nil {
		log.Errorf("Error occurred while writing transformed objects %s", err)
	}
	kt.writeDeployScript(kt.Name, outpath)
	kt.writeReadeMe(kt.Name, areNewImagesCreated, outpath)
	return nil
}

func (kt *KnativeTransformer) writeDeployScript(proj string, outpath string) {
	err := common.WriteTemplateToFile(templates.Deploy_sh, struct {
		Project string
	}{
		Project: proj,
	}, filepath.Join(outpath, "deploy.sh"), common.DefaultExecutablePermission)
	if err != nil {
		log.Errorf("Unable to write deploy script : %s", err)
	}
}

func (kt *KnativeTransformer) writeReadeMe(project string, areNewImages bool, outpath string) {
	err := common.WriteTemplateToFile(templates.KnativeReadme_md, struct {
		Project   string
		NewImages bool
	}{
		Project:   project,
		NewImages: areNewImages,
	}, filepath.Join(outpath, "Readme.md"), common.DefaultFilePermission)
	if err != nil {
		log.Errorf("Unable to write Readme : %s", err)
	}
}
