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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/a8m/tree"
	"github.com/a8m/tree/ostree"
	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/k8sschema"
	"github.com/konveyor/move2kube/internal/k8sschema/fixer"
	"github.com/konveyor/move2kube/internal/starlark/gettransformdata"
	"github.com/konveyor/move2kube/internal/starlark/runtransforms"
	startypes "github.com/konveyor/move2kube/internal/starlark/types"
	"github.com/konveyor/move2kube/internal/transformer/templates"
	"github.com/konveyor/move2kube/internal/transformer/transformations"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	"github.com/otiai10/copy"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate go run  ../../scripts/generator/generator.go templates

// Transformer translates intermediate representation to destination artifacts
type Transformer interface {
	// Transform translates intermediate representation to destination objects
	Transform(ir irtypes.IR) error
	// WriteObjects writes Transformed objects to filesystem. Also does some final transformations on the generated yamls.
	WriteObjects(outputDirectory string, transformPaths []string) error
}

// Transform transforms the IR into runtime.Objects and write all the deployments artifacts to files.
func Transform(ir irtypes.IR, outputPath string, transformPaths []string) error {
	transformers := GetTransformers()
	for _, transformer := range transformers {
		if err := transformer.Transform(ir); err != nil {
			log.Errorf("Error during translate. Error: %q", err)
			return err
		} else if err := transformer.WriteObjects(outputPath, transformPaths); err != nil {
			log.Errorf("Unable to write objects Error: %q", err)
			return err
		}
	}
	return nil
}

// GetTransformers returns all the transformers that can operate on the IR
func GetTransformers() []Transformer {
	return []Transformer{new(TektonTransformer), NewBuildconfigTransformer(), new(KnativeTransformer), NewK8sTransformer()}
}

// ConvertIRToObjects converts IR to a runtime objects
func convertIRToObjects(ir irtypes.EnhancedIR, apis []apiresource.IAPIResource) []runtime.Object {
	targetObjs := []runtime.Object{}
	ignoredObjs := ir.CachedObjects
	for _, apiResource := range apis {
		newObjs, ignoredResources := (&apiresource.APIResource{IAPIResource: apiResource}).ConvertIRToObjects(ir)
		ignoredObjs = k8sschema.Intersection(ignoredObjs, ignoredResources)
		targetObjs = append(targetObjs, newObjs...)
	}
	targetObjs = append(targetObjs, ignoredObjs...)
	return targetObjs
}

// writeContainers returns true if any scripts were written
func writeContainers(containers []irtypes.Container, outputPath, rootDir, registryURL, registryNamespace string) bool {
	sourcePath := filepath.Join(outputPath, common.SourceDir)
	log.Debugf("containersPath: %s", sourcePath)
	if err := os.MkdirAll(sourcePath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Unable to create directory %s : %s", sourcePath, err)
	}
	scriptsPath := path.Join(outputPath, common.ScriptsDir)
	if err := os.MkdirAll(scriptsPath, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Unable to create directory %s : %s", scriptsPath, err)
	}
	log.Debugf("Total number of containers : %d", len(containers))
	buildScripts := []string{}
	dockerImages := []string{}
	manualImages := []string{}
	for _, container := range containers {
		log.Debugf("Container : %t", container.New)
		if !container.New {
			continue
		}
		if len(container.NewFiles) == 0 {
			manualImages = append(manualImages, container.ImageNames...)
		}
		log.Debugf("New Container : %s", container.ImageNames[0])
		dockerImages = append(dockerImages, container.ImageNames...)
		for relPath, filecontents := range container.NewFiles {
			writePath := filepath.Join(sourcePath, relPath)
			directory := filepath.Dir(writePath)
			if err := os.MkdirAll(directory, common.DefaultDirectoryPermission); err != nil {
				log.Errorf("Unable to create directory %s : %s", directory, err)
				continue
			}
			fileperm := common.DefaultFilePermission
			if filepath.Ext(writePath) == ".sh" {
				fileperm = common.DefaultExecutablePermission
				buildScripts = append(buildScripts, filepath.Join(common.SourceDir, relPath))
			}
			log.Debugf("Writing at %s", writePath)
			if err := ioutil.WriteFile(writePath, []byte(filecontents), fileperm); err != nil {
				log.Warnf("Error writing to file at path %s Error: %q", writePath, err)
			}
		}
	}
	// Write build scripts
	if len(manualImages) > 0 {
		writepath := filepath.Join(outputPath, "Manualimages.md")
		err := common.WriteTemplateToFile(templates.Manualimages_md, struct {
			Scripts []string
		}{
			Scripts: manualImages,
		}, writepath, common.DefaultFilePermission)
		if err != nil {
			log.Errorf("Unable to create manual image : %s", err)
		}
	}

	{
		// write out the list of new files as a directory tree
		treeBytes := bytes.Buffer{}
		treeBufBytes := io.Writer(&treeBytes)
		sourceTree := tree.New(sourcePath)
		opts := &tree.Options{
			Fs:      new(ostree.FS),
			OutFile: treeBufBytes,
		}
		numDir, numFiles := sourceTree.Visit(opts)
		log.Debugf("Visiting files in source/ . Found %d directories and %d files", numDir, numFiles)
		sourceTree.Print(opts)
		log.Debugf("%s", treeBytes.String())
		newFiles := common.SourceDir + "/\n" + strings.Join(strings.Split(treeBytes.String(), "\n")[1:], "\n") // remove the first line containing source directory path
		newFilesTextPath := filepath.Join(outputPath, "newfiles.txt")
		if err := ioutil.WriteFile(newFilesTextPath, []byte(newFiles), common.DefaultFilePermission); err != nil {
			log.Errorf("Faled to create a file at path %s . Error: %q", newFilesTextPath, err)
		}
	}

	if len(buildScripts) > 0 {
		buildScriptMap := map[string]string{}
		for _, value := range buildScripts {
			buildScriptDir, buildScriptFile := filepath.Split(value)
			buildScriptMap[buildScriptFile] = buildScriptDir
		}
		log.Debugf("buildscripts %s", buildScripts)
		log.Debugf("buildScriptMap %s", buildScriptMap)
		writepath := filepath.Join(scriptsPath, "buildimages.sh")
		if err := common.WriteTemplateToFile(templates.Buildimages_sh, buildScriptMap, writepath, common.DefaultExecutablePermission); err != nil {
			log.Errorf("Unable to create script to build images : %s", err)
		}

		// copy all the sources into source/
		sourcePath := filepath.Join(outputPath, common.SourceDir)
		if err := os.MkdirAll(sourcePath, common.DefaultDirectoryPermission); err != nil {
			log.Errorf("Failed to create the source directory at path %s . Error: %q", sourcePath, err)
		} else if err := copy.Copy(rootDir, sourcePath); err != nil {
			log.Errorf("Failed to copy the sources over to the folder at path %s Error: %q", sourcePath, err)
		}
	}
	if len(dockerImages) > 0 {
		writepath := filepath.Join(scriptsPath, "pushimages.sh")
		err := common.WriteTemplateToFile(templates.Pushimages_sh, struct {
			Images            []string
			RegistryURL       string
			RegistryNamespace string
		}{
			Images:            dockerImages,
			RegistryURL:       registryURL,
			RegistryNamespace: registryNamespace,
		}, writepath, common.DefaultExecutablePermission)
		if err != nil {
			log.Errorf("Unable to create script to push images : %s", err)
		}
		return true
	}
	return false
}

func fixAndConvert(obj runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, ignoreUnsupportedKinds bool) (runtime.Object, error) {
	fixedobj := fixer.Fix(obj)
	return k8sschema.ConvertToSupportedVersion(fixedobj, clusterSpec, ignoreUnsupportedKinds)
}

// fixConvertAndTransformObjs runs fixers, converts to a supported version and runs transformations on the objects
func fixConvertAndTransformObjs(objs []runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, ignoreUnsupportedKinds bool, transformPaths []string) ([]runtime.Object, error) {
	// Fix and convert
	fixedAndConvertedObjs := []runtime.Object{}
	for _, obj := range objs {
		fixedAndConvertedObj, err := fixAndConvert(obj, clusterSpec, ignoreUnsupportedKinds)
		if err != nil {
			log.Errorf("Failed to fix and convert the runtime.Object. Object:\n%+v\nError: %q", obj, err)
			fixedAndConvertedObj = obj
		}
		fixedAndConvertedObjs = append(fixedAndConvertedObjs, fixedAndConvertedObj)
	}
	// Transform
	// Transform - get the k8s resources
	k8sResources := []startypes.K8sResourceT{}
	for _, fixedAndConvertedObj := range fixedAndConvertedObjs {
		k8sResource, err := gettransformdata.GetK8sResourceFromObject(fixedAndConvertedObj)
		if err != nil {
			log.Errorf("Failed to convert the object into a K8sResourceT. Object:\n%+v\nError: %q", fixedAndConvertedObj, err)
			return nil, err
		}
		k8sResources = append(k8sResources, k8sResource)
	}
	// Transform - get the transforms
	transforms, err := transformations.GetTransformsFromPathsUsingDefaults(transformPaths)
	if err != nil {
		log.Errorf("Failed to get the transformations. Error: %q", err)
		return nil, err
	}
	// Transform - run the transformations on the k8s resources
	transformedK8sResources, err := runtransforms.ApplyTransforms(transforms, k8sResources)
	if err != nil {
		log.Errorf("Failed to apply the transformations. Error: %q", err)
		return nil, err
	}
	fixedConvertedAndTransformedObjs := []runtime.Object{}
	for i, transformedK8sResource := range transformedK8sResources {
		fixedConvertedAndTransformedObj, err := gettransformdata.GetObjectFromK8sResource(transformedK8sResource, fixedAndConvertedObjs[i])
		if err != nil {
			log.Errorf("Failed to convert the K8sResourceT back into a runtime.Object. K8s resource:\n%+v\nObject:\n%+v\nError: %q", transformedK8sResource, fixedAndConvertedObjs[i], err)
			fixedConvertedAndTransformedObj = fixedAndConvertedObjs[i]
		}
		fixedConvertedAndTransformedObjs = append(fixedConvertedAndTransformedObjs, fixedConvertedAndTransformedObj)
	}
	return fixedConvertedAndTransformedObjs, nil
}

// writeObjects writes the runtime objects to yaml files
func writeObjects(outputPath string, objs []runtime.Object) ([]string, error) {
	if err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission); err != nil {
		return nil, err
	}
	filesWritten := []string{}
	for _, obj := range objs {
		objYamlBytes, err := common.MarshalObjToYaml(obj)
		if err != nil {
			log.Errorf("failed to marshal the runtime.Object to yaml. Object:\n%+v\nError: %q", obj, err)
			continue
		}
		yamlPath := filepath.Join(outputPath, getFilename(obj))
		if err := ioutil.WriteFile(yamlPath, objYamlBytes, common.DefaultFilePermission); err != nil {
			log.Errorf("failed to write the yaml to file at path %s . Error: %q", yamlPath, err)
			continue
		}
		filesWritten = append(filesWritten, yamlPath)
	}
	return filesWritten, nil
}

func getFilename(obj runtime.Object) string {
	val := reflect.ValueOf(obj).Elem()
	typeMeta := val.FieldByName("TypeMeta").Interface().(metav1.TypeMeta)
	objectMeta := val.FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
	return fmt.Sprintf("%s-%s.yaml", objectMeta.Name, strings.ToLower(typeMeta.Kind))
}

func writeTransformedObjects(outputPath string, objs []runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, ignoreUnsupportedKinds bool, transformPaths []string) ([]string, error) {
	fixedConvertedAndTransformedObjs, err := fixConvertAndTransformObjs(objs, clusterSpec, ignoreUnsupportedKinds, transformPaths)
	if err != nil {
		log.Errorf("Failed to fix, convert and transform objects. Error: %q", err)
		return nil, err
	}
	return writeObjects(outputPath, fixedConvertedAndTransformedObjs)
}
