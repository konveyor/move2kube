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
	"os/exec"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/transformer/templates"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	outputtypes "github.com/konveyor/move2kube/types/output"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	helmTemplatesRelPath = "templates"
)

// K8sTransformer implements Transformer interface
type K8sTransformer struct {
	RootDir                string
	TransformedObjects     []runtime.Object
	Containers             []irtypes.Container
	Values                 outputtypes.HelmValues
	TargetClusterSpec      collecttypes.ClusterMetadataSpec
	Helm                   bool
	Name                   string
	IgnoreUnsupportedKinds bool
	ExposedServicePaths    map[string]string
	AddCopySources         bool
}

// NewK8sTransformer creates a new instance of K8sTransformer
func NewK8sTransformer() *K8sTransformer {
	kt := new(K8sTransformer)
	kt.TransformedObjects = []runtime.Object{}
	kt.Containers = []irtypes.Container{}
	kt.ExposedServicePaths = map[string]string{}
	return kt
}

// Transform translates intermediate representation to destination objects
func (kt *K8sTransformer) Transform(ir irtypes.IR) error {
	log.Debugf("Starting Kubernetes transform")
	log.Debugf("Total services to be transformed : %d", len(ir.Services))

	kt.Name = ir.Name
	kt.Values = ir.Values
	kt.Containers = ir.Containers
	kt.TargetClusterSpec = ir.TargetClusterSpec
	kt.IgnoreUnsupportedKinds = ir.Kubernetes.IgnoreUnsupportedKinds
	kt.Helm = (ir.Kubernetes.ArtifactType == plantypes.Helm)

	kt.TransformedObjects = convertIRToObjects(irtypes.NewEnhancedIRFromIR(ir), kt.getAPIResources())
	kt.RootDir = ir.RootDir
	kt.AddCopySources = ir.AddCopySources

	for _, service := range ir.Services {
		if service.HasValidAnnotation(common.ExposeSelector) {
			kt.ExposedServicePaths[service.Name] = service.ServiceRelPath
		}
	}

	log.Debugf("Total transformed objects : %d", len(kt.TransformedObjects))

	return nil
}

func (kt *K8sTransformer) getAPIResources() []apiresource.IAPIResource {
	return []apiresource.IAPIResource{&apiresource.Deployment{}, &apiresource.Storage{}, &apiresource.Service{}, &apiresource.ImageStream{}, &apiresource.NetworkPolicy{}}
}

// WriteObjects writes Transformed objects to filesystem
func (kt *K8sTransformer) WriteObjects(outpath string) error {
	areNewImagesCreated := writeContainers(kt.Containers, outpath, kt.RootDir, kt.Values.RegistryURL, kt.Values.RegistryNamespace, kt.AddCopySources)

	artifactspath := filepath.Join(outpath, kt.Name)
	if kt.Helm {
		if err := kt.generateHelmMetadata(artifactspath, kt.Values); err != nil {
			log.Debugf("Failed to generate helm metadata properly, continuing anyway. Error: %q", err)
		}
		artifactspath = filepath.Join(artifactspath, helmTemplatesRelPath)
	}
	log.Debugf("Total services to be serialized : %d", len(kt.TransformedObjects))
	_, err := writeTransformedObjects(artifactspath, kt.TransformedObjects, kt.TargetClusterSpec, kt.IgnoreUnsupportedKinds)
	if err != nil {
		log.Errorf("Error occurred while writing transformed objects %s", err)
	}
	if kt.Helm {
		_ = kt.createOperator(kt.Name, outpath)
	} else {
		kt.writeDeployScript(kt.Name, outpath)
	}
	kt.writeReadMe(kt.Name, areNewImagesCreated, kt.Helm, kt.AddCopySources, outpath)
	return nil
}

func (kt *K8sTransformer) generateHelmMetadata(dirName string, values outputtypes.HelmValues) error {
	err := os.MkdirAll(dirName, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create Helm Metadata directory %s : %s", dirName, err)
		return err
	}
	//README.md
	readme := "This chart was created by Move2Kube\n"
	err = ioutil.WriteFile(filepath.Join(dirName, "README.md"), []byte(readme), common.DefaultFilePermission)
	if err != nil {
		log.Errorf("Error while writing Readme : %s", err)
	}

	//Chart.yaml
	type ChartDetails struct {
		Name string
	}
	err = common.WriteTemplateToFile(templates.Chart_tpl, ChartDetails{filepath.Base(dirName)}, filepath.Join(dirName, "Chart.yaml"), common.DefaultFilePermission)
	if err != nil {
		log.Errorf("Error while writing Chart.yaml : %s", err)
	}

	// Create templates directory
	err = os.MkdirAll(filepath.Join(dirName, helmTemplatesRelPath), common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create templates directory : %s", err)
	}

	notesStr, err := common.GetStringFromTemplate(templates.NOTES_txt, struct {
		IsHelm              bool
		ExposedServicePaths map[string]string
	}{
		IsHelm:              true,
		ExposedServicePaths: kt.ExposedServicePaths,
	})
	if err != nil {
		log.Errorf("Failed to fill the NOTES.txt template %s with the service paths %v Error: %q", templates.NOTES_txt, kt.ExposedServicePaths, err)
	}

	//NOTES.txt
	err = ioutil.WriteFile(filepath.Join(dirName, helmTemplatesRelPath, "NOTES.txt"), []byte(templates.HelmNotes_txt+notesStr), common.DefaultFilePermission)
	if err != nil {
		log.Errorf("Error while writing Helm NOTES.txt : %s", err)
	}

	//values.yaml
	outputPath := filepath.Join(dirName, "values.yaml")
	err = common.WriteYaml(outputPath, values)
	if err != nil {
		log.Warn("Error in writing Helm values", err)
	} else {
		log.Debugf("Wrote Helm values to file: %s", outputPath)
	}

	scriptspath := filepath.Join(filepath.Dir(dirName), common.ScriptsDir)
	err = os.MkdirAll(scriptspath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create directory %s : %s", scriptspath, err)
	}
	err = common.WriteTemplateToFile(templates.Helminstall_sh, struct {
		Project string
	}{
		Project: filepath.Base(dirName),
	}, filepath.Join(scriptspath, "helminstall.sh"), common.DefaultExecutablePermission)
	return err
}

func (kt *K8sTransformer) createOperator(projectname string, basepath string) error {
	_, err := exec.LookPath("operator-sdk")
	if err != nil {
		log.Warnf("Unable to find operator-sdk. Skipping operator generation : %s", err)
		return err
	}
	operatorname := projectname + "-operator"
	operatorpath := filepath.Join(basepath, operatorname)
	_, err = os.Stat(operatorpath)
	if !os.IsNotExist(err) {
		os.RemoveAll(operatorpath)
	}
	err = os.MkdirAll(operatorpath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create Operator directory %s : %s", operatorpath, err)
		return err
	}
	helmchartpath, err := filepath.Abs(filepath.Join(basepath, projectname))
	if err != nil {
		log.Warnf("Could not resolve helm chart path : %s", err)
		return err
	}
	cmd := exec.Command("operator-sdk", "init", "--plugins=helm", "--helm-chart="+helmchartpath, "--domain=io", "--group="+projectname, "--version=v1alpha1")
	cmd.Dir = operatorpath
	output, err := cmd.Output()
	if err != nil {
		log.Warnf("Error during operator creation : %s, %s", err, output)
		return err
	}
	return nil
}

func (kt *K8sTransformer) writeDeployScript(proj string, outpath string) {
	scriptspath := filepath.Join(outpath, common.ScriptsDir)
	err := os.MkdirAll(scriptspath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create directory %s : %s", scriptspath, err)
	}
	deployScriptPath := filepath.Join(scriptspath, "deploy.sh")
	err = common.WriteTemplateToFile(templates.Deploy_sh, struct {
		Project string
	}{
		Project: proj,
	}, deployScriptPath, common.DefaultExecutablePermission)
	if err != nil {
		log.Errorf("Failed to write the deploy script at path %q Error: %q", deployScriptPath, err)
	}

	notesPath := filepath.Join(outpath, "NOTES.txt")
	err = common.WriteTemplateToFile(templates.NOTES_txt, struct {
		IsHelm              bool
		IngressHost         string
		ExposedServicePaths map[string]string
	}{
		IsHelm:              false,
		IngressHost:         kt.TargetClusterSpec.Host,
		ExposedServicePaths: kt.ExposedServicePaths,
	}, notesPath, common.DefaultFilePermission)
	if err != nil {
		log.Errorf("Failed to write the NOTES.txt file at path %q Error: %q", notesPath, err)
	}

}

func (kt *K8sTransformer) writeReadMe(project string, areNewImages bool, isHelm bool, addCopySources bool, outpath string) {
	err := common.WriteTemplateToFile(templates.K8sReadme_md, struct {
		Project        string
		NewImages      bool
		Helm           bool
		AddCopySources bool
	}{
		Project:        project,
		NewImages:      areNewImages,
		Helm:           isHelm,
		AddCopySources: addCopySources,
	}, filepath.Join(outpath, "Readme.md"), common.DefaultFilePermission)
	if err != nil {
		log.Errorf("Unable to write readme : %s", err)
	}
}
