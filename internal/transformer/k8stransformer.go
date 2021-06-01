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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/deepcopy"
	parameterize "github.com/konveyor/move2kube/internal/parameterizer"
	"github.com/konveyor/move2kube/internal/transformer/kustomize"
	"github.com/konveyor/move2kube/internal/transformer/templates"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	outputtypes "github.com/konveyor/move2kube/types/output"
	templatev1 "github.com/openshift/api/template/v1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	templatesDir = "templates"
)

// K8sTransformer implements Transformer interface
type K8sTransformer struct {
	RootDir                         string
	TransformedObjects              []runtime.Object
	ParameterizedTransformedObjects []runtime.Object
	Containers                      []irtypes.Container
	Values                          outputtypes.HelmValues
	TargetClusterSpec               collecttypes.ClusterMetadataSpec
	Name                            string
	IgnoreUnsupportedKinds          bool
	ExposedServicePaths             map[string]string
}

// NewK8sTransformer creates a new instance of K8sTransformer
func NewK8sTransformer() *K8sTransformer {
	kt := new(K8sTransformer)
	kt.TransformedObjects = []runtime.Object{}
	kt.ParameterizedTransformedObjects = []runtime.Object{}
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

	kt.TransformedObjects = convertIRToObjects(irtypes.NewEnhancedIRFromIR(ir), kt.getAPIResources())

	parameterizedIR, err := parameterize.Parameterize(deepcopy.DeepCopy(ir).(irtypes.IR))
	if err != nil {
		log.Debugf("Failed to paramterize the IR. Error: %q", err)
		parameterizedIR = ir
	}
	kt.Values = parameterizedIR.Values

	kt.ParameterizedTransformedObjects = convertIRToObjects(irtypes.NewEnhancedIRFromIR(parameterizedIR), kt.getAPIResources())
	if len(kt.TransformedObjects) != len(kt.ParameterizedTransformedObjects) {
		log.Errorf(
			"Failed to parameterize properly. Expected both lists to have the same number of objects.\nFound %d normal objects:\n%+v\nFound %d paramertized objects:\n%+v",
			len(kt.TransformedObjects), kt.TransformedObjects, len(kt.ParameterizedTransformedObjects), kt.ParameterizedTransformedObjects,
		)
	}
	kt.reorderParameterizedObjects()

	kt.RootDir = ir.RootDir

	for _, service := range ir.Services {
		if service.HasValidAnnotation(common.ExposeSelector) {
			kt.ExposedServicePaths[service.Name] = service.ServiceRelPath
		}
	}

	log.Debugf("Total transformed objects : %d", len(kt.TransformedObjects))

	return nil
}

func (kt *K8sTransformer) reorderParameterizedObjects() {
	reorderedObjects := []runtime.Object{}
	usedSet := map[int]bool{}
	for _, obj := range kt.TransformedObjects {
		paramObjIdxs := kt.getAllMatchingParameterizedObjects(obj)
		if len(paramObjIdxs) < 1 {
			log.Errorf("Failed to find a parameterized object that matches the object:\n%+v", obj)
			reorderedObjects = append(reorderedObjects, obj)
			continue
		}
		if len(paramObjIdxs) > 1 {
			matchedParamObjs := []runtime.Object{}
			for _, paramObjIdx := range paramObjIdxs {
				matchedParamObjs = append(matchedParamObjs, kt.ParameterizedTransformedObjects[paramObjIdx])
			}
			log.Errorf("Found multiple parameterized objects:\n%+v\nthat match the object:\n%+v", matchedParamObjs, obj)
		}
		found := false
		for _, paramObjIdx := range paramObjIdxs {
			if !usedSet[paramObjIdx] {
				reorderedObjects = append(reorderedObjects, kt.ParameterizedTransformedObjects[paramObjIdx])
				usedSet[paramObjIdx] = true
				found = true
				break
			}
		}
		if !found {
			log.Errorf("Failed to find a parameterized object, that wasn't used already, to match against object:\n%+v", obj)
			reorderedObjects = append(reorderedObjects, obj)
		}
	}
	kt.ParameterizedTransformedObjects = reorderedObjects
}

func (kt *K8sTransformer) getAllMatchingParameterizedObjects(obj runtime.Object) []int {
	paramObjIdxs := []int{}
	for paramObjIdx, paramObj := range kt.ParameterizedTransformedObjects {
		if common.IsSameRuntimeObject(obj, paramObj) {
			paramObjIdxs = append(paramObjIdxs, paramObjIdx)
		}
	}
	return paramObjIdxs
}

func (kt *K8sTransformer) getAPIResources() []apiresource.IAPIResource {
	return []apiresource.IAPIResource{&apiresource.Deployment{}, &apiresource.Storage{}, &apiresource.Service{}, &apiresource.ImageStream{}, &apiresource.NetworkPolicy{}}
}

// WriteObjects writes the transformed objects to files.
// The output folder structure is given below:
// myproject/
//   deploy/
//     custom/yamls/
//     yamls/
//     cicd/
//       tekton/
//       argocd/
//   scripts/
//   source/
//   my/dest/helm/
//       myproject/
//   my/other/dest/helm/
//       myproject/
//   my/dest/operator/
//   another/kustomize/
//       base/
//       overlay/
//         dev/
//         staging/
//         prod/
//   yetanother/openshift-templates/

func (kt *K8sTransformer) WriteObjects(outputPath string, transformPaths []string) error {
	deployPath := filepath.Join(outputPath, common.DeployDir)
	if err := os.MkdirAll(deployPath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Unable to create deploy directory at path %s Error: %q", outputPath, err)
	}

	// source/
	areNewImagesCreated := writeContainers(kt.Containers, outputPath, kt.RootDir, kt.Values.RegistryURL, kt.Values.RegistryNamespace)

	// deploy/helm/ and scripts/deployhelm.sh
	helmPath := filepath.Join(deployPath, common.HelmDir, kt.Name)
	if err := kt.generateHelmArtifacts(helmPath, outputPath, kt.Values, transformPaths); err != nil {
		log.Debugf("Failed to generate helm metadata properly, continuing anyway. Error: %q", err)
	}

	// deploy/yamls/
	log.Debugf("Total %d services to be serialized.", len(kt.TransformedObjects))
	fixedConvertedTransformedObjs, err := fixConvertAndTransformObjs(kt.TransformedObjects, kt.TargetClusterSpec, kt.IgnoreUnsupportedKinds, transformPaths)
	if err != nil {
		log.Errorf("Failed to fix, convert and transform the objects. Error: %q", err)
	}
	k8sArtifactsPath := filepath.Join(deployPath, "yamls")
	if _, err := writeObjects(k8sArtifactsPath, fixedConvertedTransformedObjs); err != nil {
		log.Errorf("Failed to write the transformed objects to the directory at path %s . Error: %q", k8sArtifactsPath, err)
	}
	// scripts/deploy.sh
	kt.writeDeployScript(kt.Name, outputPath)

	// deploy/operator/
	if err := kt.createOperator(kt.Name, filepath.Join(deployPath, "operator"), helmPath); err != nil {
		log.Errorf("Failed to generate the operator. Error: %q", err)
	}

	// deploy/kustomize/
	if err := kt.generateKustomize(filepath.Join(deployPath, "kustomize"), transformPaths); err != nil {
		log.Errorf("Failed to generate the kustomize artifacts. Error: %q", err)
	}

	// README.md
	kt.writeReadMe(kt.Name, areNewImagesCreated, outputPath)

	// deploy/openshift-templates/
	openshiftTemplatesPath := filepath.Join(deployPath, common.OCTemplatesDir)
	if _, err := kt.generateOpenshiftTemplates(openshiftTemplatesPath, outputPath, fixedConvertedTransformedObjs); err != nil {
		log.Errorf("Failed to write the openshift templates to the directory at path %s . Error: %q", openshiftTemplatesPath, err)
	}

	return nil
}

func (kt *K8sTransformer) generateHelmArtifacts(helmPath string, outputPath string, values outputtypes.HelmValues, transformPaths []string) error {
	if err := os.MkdirAll(helmPath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Unable to create Helm directory at path %s Error: %q", helmPath, err)
		return err
	}
	// README.md
	readme := "This chart was created by Move2Kube\n"
	if err := ioutil.WriteFile(filepath.Join(helmPath, "README.md"), []byte(readme), common.DefaultFilePermission); err != nil {
		log.Errorf("Error while writing Readme : %s", err)
		return err
	}

	// Chart.yaml
	if err := common.WriteTemplateToFile(templates.Chart_tpl, struct{ Name string }{filepath.Base(helmPath)}, filepath.Join(helmPath, "Chart.yaml"), common.DefaultFilePermission); err != nil {
		log.Errorf("Error while writing Chart.yaml : %s", err)
		return err
	}

	// values.yaml
	valuesPath := filepath.Join(helmPath, "values.yaml")
	if err := common.WriteYaml(valuesPath, values); err != nil {
		log.Warn("Error in writing Helm values", err)
	} else {
		log.Debugf("Wrote Helm values to file: %s", valuesPath)
	}

	// templates/
	if err := os.MkdirAll(filepath.Join(helmPath, templatesDir), common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Unable to create templates directory : %s", err)
		return err
	}

	// templates/NOTES.txt
	notesStr, err := common.GetStringFromTemplate(templates.NOTES_txt, struct {
		IsHelm              bool
		ExposedServicePaths map[string]string
	}{
		IsHelm:              true,
		ExposedServicePaths: kt.ExposedServicePaths,
	})
	if err != nil {
		log.Errorf("Failed to fill the NOTES.txt template %s with the service paths %v Error: %q", templates.NOTES_txt, kt.ExposedServicePaths, err)
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(helmPath, templatesDir, "NOTES.txt"), []byte(templates.HelmNotes_txt+notesStr), common.DefaultFilePermission); err != nil {
		log.Errorf("Error while writing Helm NOTES.txt : %s", err)
		return err
	}

	// scripts/deployhelm.sh
	scriptsPath := filepath.Join(outputPath, common.ScriptsDir)
	if err := os.MkdirAll(scriptsPath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Unable to create scripts directory at path %s Error: %q", scriptsPath, err)
		return err
	}
	deployHelmScriptPath := filepath.Join(scriptsPath, "deployhelm.sh")
	if err := common.WriteTemplateToFile(templates.DeployHelm_sh, struct{ Project string }{Project: kt.Name}, deployHelmScriptPath, common.DefaultExecutablePermission); err != nil {
		log.Errorf("Unable to create deploy helm script at path %s Error: %q", deployHelmScriptPath, err)
		return err
	}

	// templates/
	helmArtifactsPath := filepath.Join(helmPath, templatesDir)
	if _, err := writeTransformedObjects(helmArtifactsPath, kt.ParameterizedTransformedObjects, kt.TargetClusterSpec, kt.IgnoreUnsupportedKinds, transformPaths); err != nil {
		log.Errorf("Error occurred while writing transformed objects. Error: %q", err)
		return err
	}
	return nil
}

func (kt *K8sTransformer) createOperator(projectName string, operatorPath string, helmPath string) error {
	if _, err := exec.LookPath("operator-sdk"); err != nil {
		log.Warnf("Unable to find operator-sdk. Skipping operator generation. Error: %q", err)
		return err
	}
	if err := os.MkdirAll(operatorPath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Failed to create the operator directory at path %s . Error: %q", operatorPath, err)
		return err
	}
	cmd := exec.Command("operator-sdk", "init", "--plugins=helm", "--helm-chart="+helmPath, "--domain=io", "--group="+projectName, "--version=v1alpha1")
	cmd.Dir = operatorPath
	output, err := cmd.Output()
	if err != nil {
		log.Warnf("Failed to create the operator. Output:\n%s\nError: %q", string(output), err)
		return err
	}
	log.Debugf("Output from operator creation:\n%s", string(output))
	return nil
}

func (kt *K8sTransformer) writeDeployScript(proj string, outputPath string) {
	scriptspath := filepath.Join(outputPath, common.ScriptsDir)
	if err := os.MkdirAll(scriptspath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Unable to create directory %s : %s", scriptspath, err)
	}
	deployScriptPath := filepath.Join(scriptspath, "deploy.sh")
	if err := ioutil.WriteFile(deployScriptPath, []byte(templates.Deploy_sh), common.DefaultExecutablePermission); err != nil {
		log.Errorf("Failed to write the deploy script at path %s . Error: %q", deployScriptPath, err)
	}
	deployKustomizeScriptPath := filepath.Join(scriptspath, "deploykustomize.sh")
	if err := ioutil.WriteFile(deployKustomizeScriptPath, []byte(templates.DeployKustomize_sh), common.DefaultExecutablePermission); err != nil {
		log.Errorf("Failed to write the deploy kustomize script at path %s . Error: %q", deployKustomizeScriptPath, err)
	}
	deployKnativeScriptPath := filepath.Join(scriptspath, "deployknative.sh")
	if err := ioutil.WriteFile(deployKnativeScriptPath, []byte(templates.DeployKnative_sh), common.DefaultExecutablePermission); err != nil {
		log.Errorf("Failed to write the deploy knative script at path %s . Error: %q", deployKnativeScriptPath, err)
	}
	notes := struct {
		IsHelm              bool
		IngressHost         string
		ExposedServicePaths map[string]string
	}{
		IsHelm:              false,
		IngressHost:         kt.TargetClusterSpec.Host,
		ExposedServicePaths: kt.ExposedServicePaths,
	}
	k8sArtifactsPath := filepath.Join(outputPath, common.DeployDir, "yamls")
	if err := os.MkdirAll(k8sArtifactsPath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Failed to make the k8s yamls directory at path %s . Error: %q", k8sArtifactsPath, err)
	}
	k8sNotesPath := filepath.Join(k8sArtifactsPath, "NOTES.txt")
	if err := common.WriteTemplateToFile(templates.NOTES_txt, notes, k8sNotesPath, common.DefaultFilePermission); err != nil {
		log.Errorf("Failed to write the NOTES.txt file at path %s . Error: %q", k8sNotesPath, err)
	}
	kustomizeArtifactsPath := filepath.Join(outputPath, common.DeployDir, "kustomize")
	if err := os.MkdirAll(kustomizeArtifactsPath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Failed to make the k8s yamls directory at path %s . Error: %q", kustomizeArtifactsPath, err)
	}
	kustomizeNotesPath := filepath.Join(kustomizeArtifactsPath, "NOTES.txt")
	if err := common.WriteTemplateToFile(templates.NOTES_txt, notes, kustomizeNotesPath, common.DefaultFilePermission); err != nil {
		log.Errorf("Failed to write the NOTES.txt file at path %s . Error: %q", kustomizeNotesPath, err)
	}
}

// generateKustomize generates all the kustomize artifacts given both the original and parameterized objects.
func (kt *K8sTransformer) generateKustomize(kustomizePath string, transformPaths []string) error {
	if err := os.MkdirAll(kustomizePath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Failed to create the kustomize directory at path %s . Error: %q", kustomizePath, err)
		return err
	}
	// deploy/kustomize/base/
	kustomizeBaseDir := filepath.Join(kustomizePath, "base")
	if _, err := writeTransformedObjects(kustomizeBaseDir, kt.TransformedObjects, kt.TargetClusterSpec, kt.IgnoreUnsupportedKinds, transformPaths); err != nil {
		log.Errorf("Error occurred while writing transformed objects. Error: %q", err)
	}

	filenames := []string{}
	fixedConvertedObjs := []runtime.Object{}
	fixedConvertedParamObjs := []runtime.Object{}

	for i, obj := range kt.TransformedObjects {
		paramObj := kt.ParameterizedTransformedObjects[i]
		fixedObj, err := fixAndConvert(obj, kt.TargetClusterSpec, kt.IgnoreUnsupportedKinds)
		if err != nil {
			continue
		}
		fixedParamObj, err := fixAndConvert(paramObj, kt.TargetClusterSpec, kt.IgnoreUnsupportedKinds)
		if err != nil {
			continue
		}
		filenames = append(filenames, getFilename(fixedObj))
		fixedConvertedObjs = append(fixedConvertedObjs, fixedObj)
		fixedConvertedParamObjs = append(fixedConvertedParamObjs, fixedParamObj)
	}

	return kustomize.GenerateKustomize(kustomizePath, filenames, fixedConvertedObjs, fixedConvertedParamObjs)
}

func (kt *K8sTransformer) writeReadMe(project string, areNewImages bool, outpath string) {
	err := common.WriteTemplateToFile(templates.K8sReadme_md, struct {
		Project   string
		NewImages bool
	}{
		Project:   project,
		NewImages: areNewImages,
	}, filepath.Join(outpath, "README.md"), common.DefaultFilePermission)
	if err != nil {
		log.Errorf("Unable to write readme : %s", err)
	}
}

func (kt *K8sTransformer) generateOpenshiftTemplates(ocTemplatesPath, outputPath string, objs []runtime.Object) ([]string, error) {
	// deploy/openshift-templates/
	raws := []runtime.RawExtension{}
	for _, obj := range objs {
		raws = append(raws, runtime.RawExtension{Object: obj})
	}
	templ := &templatev1.Template{
		TypeMeta:   metav1.TypeMeta{APIVersion: "template.openshift.io/v1", Kind: "Template"},
		ObjectMeta: metav1.ObjectMeta{Name: common.MakeStringDNSNameCompliant(kt.Name)},
		Objects:    raws,
	}
	filesWritten, err := writeObjects(ocTemplatesPath, []runtime.Object{templ})
	if err != nil {
		log.Errorf("failed to write the openshift template objects. Error: %q", err)
		return filesWritten, err
	}
	if len(filesWritten) == 0 {
		return filesWritten, fmt.Errorf("no files written")
	}

	// scripts/deployoctemplates.sh
	scriptsPath := filepath.Join(outputPath, common.ScriptsDir)
	if err := os.MkdirAll(scriptsPath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("unable to create scripts directory at path %s . Error: %q", scriptsPath, err)
		return filesWritten, err
	}
	deployScriptPath := filepath.Join(scriptsPath, "deployoctemplates.sh")
	filename := filepath.Base(filesWritten[0])
	if err := common.WriteTemplateToFile(templates.DeployOCTemplates_sh, struct{ Filename string }{Filename: filename}, deployScriptPath, common.DefaultExecutablePermission); err != nil {
		log.Errorf("unable to create deploy openshift templates script at path %s Error: %q", deployScriptPath, err)
		return filesWritten, err
	}
	filesWritten = append(filesWritten, deployScriptPath)
	return filesWritten, nil
}
