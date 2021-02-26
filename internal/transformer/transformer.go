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
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/k8sschema"
	"github.com/konveyor/move2kube/internal/k8sschema/fixer"
	"github.com/konveyor/move2kube/internal/starlark"
	"github.com/konveyor/move2kube/internal/starlark/gettransformdata"
	"github.com/konveyor/move2kube/internal/starlark/runtransforms"
	"github.com/konveyor/move2kube/internal/starlark/types"
	"github.com/konveyor/move2kube/internal/transformer/templates"
	"github.com/konveyor/move2kube/internal/transformer/transformations"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	"github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate go run github.com/konveyor/move2kube/internal/common/generator templates

// Transformer translates intermediate representation to destination artifacts
type Transformer interface {
	// Transform translates intermediate representation to destination objects
	Transform(ir irtypes.IR) error
	// WriteObjects writes Transformed objects to filesystem
	WriteObjects(outDirectory string, TransformPaths []string) error
}

// GetTransformer returns a transformer that is suitable for an IR
func GetTransformer(ir irtypes.IR) Transformer {
	if ir.Kubernetes.ArtifactType == plan.Knative {
		return &KnativeTransformer{}
	}
	return NewK8sTransformer()
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
func writeContainers(containers []irtypes.Container, outpath, rootDir, registryURL, registryNamespace string, addCopySources bool) bool {
	containersdirectory := "containers"
	containerspath := path.Join(outpath, containersdirectory)
	log.Debugf("containerspath %s", containerspath)
	err := os.MkdirAll(containerspath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create directory %s : %s", containerspath, err)
	}
	scriptspath := path.Join(outpath, common.ScriptsDir)
	err = os.MkdirAll(scriptspath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create directory %s : %s", scriptspath, err)
	}
	log.Debugf("Total number of containers : %d", len(containers))
	buildscripts := []string{}
	dockerimages := []string{}
	manualimages := []string{}
	for _, container := range containers {
		log.Debugf("Container : %t", container.New)
		if !container.New {
			continue
		}
		if len(container.NewFiles) == 0 {
			manualimages = append(manualimages, container.ImageNames...)
		}
		log.Debugf("New Container : %s", container.ImageNames[0])
		dockerimages = append(dockerimages, container.ImageNames...)
		for relPath, filecontents := range container.NewFiles {
			writepath := filepath.Join(containerspath, relPath)
			directory := filepath.Dir(writepath)
			if err := os.MkdirAll(directory, common.DefaultDirectoryPermission); err != nil {
				log.Errorf("Unable to create directory %s : %s", directory, err)
				continue
			}
			fileperm := common.DefaultFilePermission
			if filepath.Ext(writepath) == ".sh" {
				fileperm = common.DefaultExecutablePermission
				buildscripts = append(buildscripts, filepath.Join(containersdirectory, relPath))
			}
			log.Debugf("Writing at %s", writepath)
			err = ioutil.WriteFile(writepath, []byte(filecontents), fileperm)
			if err != nil {
				log.Warnf("Error writing file at %s : %s", writepath, err)
			}
		}
	}
	//Write build scripts
	if len(manualimages) > 0 {
		writepath := filepath.Join(outpath, "Manualimages.md")
		err := common.WriteTemplateToFile(templates.Manualimages_md, struct {
			Scripts []string
		}{
			Scripts: manualimages,
		}, writepath, common.DefaultFilePermission)
		if err != nil {
			log.Errorf("Unable to create manual image : %s", err)
		}
	}
	if len(buildscripts) > 0 {
		buildScriptMap := map[string]string{}
		for _, value := range buildscripts {
			buildScriptDir, buildScriptFile := filepath.Split(value)
			buildScriptMap[buildScriptFile] = buildScriptDir
		}
		log.Debugf("buildscripts %s", buildscripts)
		log.Debugf("buildScriptMap %s", buildScriptMap)
		writepath := filepath.Join(scriptspath, "buildimages.sh")
		err := common.WriteTemplateToFile(templates.Buildimages_sh, buildScriptMap, writepath, common.DefaultExecutablePermission)
		if err != nil {
			log.Errorf("Unable to create script to build images : %s", err)
		}
		if addCopySources {
			relRootDir, err := filepath.Rel(outpath, rootDir)
			if err != nil {
				log.Errorf("Failed to make the root directory path %q relative to the output directory %q Error %q", rootDir, outpath, err)
				relRootDir = rootDir
			}
			writepath = filepath.Join(scriptspath, "copysources.sh")
			err = common.WriteTemplateToFile(templates.CopySources_sh, struct {
				RelRootDir string
				Dst        string
			}{
				RelRootDir: relRootDir,
				Dst:        containersdirectory,
			}, writepath, common.DefaultExecutablePermission)
			if err != nil {
				log.Errorf("Unable to create script to build images : %s", err)
			}
		}
	}
	if len(dockerimages) > 0 {
		writepath := filepath.Join(scriptspath, "pushimages.sh")
		err := common.WriteTemplateToFile(templates.Pushimages_sh, struct {
			Images            []string
			RegistryURL       string
			RegistryNamespace string
		}{
			Images:            dockerimages,
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

func writeTransformedObjects(outputPath string, objs []runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, ignoreUnsupportedKinds bool, transformPaths []string) ([]string, error) {
	k8sResources := []types.K8sResourceT{}
	for _, obj := range objs {
		fixedobj := fixer.Fix(obj)
		obj, err := k8sschema.ConvertToSupportedVersion(fixedobj, clusterSpec, ignoreUnsupportedKinds)
		if err != nil {
			log.Warnf("Ignoring object : %s", err)
			continue
		}
		// Encode object as yaml
		j, err := json.Marshal(obj)
		if err != nil {
			log.Errorf("Error while Marshalling object : %s", err)
			continue
		}
		var jsonObj interface{}
		err = yaml.Unmarshal(j, &jsonObj)
		if err != nil {
			log.Errorf("Unable to unmarshal json : %s", err)
			continue
		}
		var b bytes.Buffer
		encoder := yaml.NewEncoder(&b)
		encoder.SetIndent(2)
		if err := encoder.Encode(jsonObj); err != nil {
			log.Errorf("Error while Encoding object : %s", err)
			continue
		}
		// Get k8s resources from yaml
		k8sYaml := string(b.Bytes())
		currK8sResources, err := gettransformdata.GetK8sResourcesFromYaml(k8sYaml)
		if err != nil {
			log.Errorf("Failed to decode the k8s resource from the marshalled yaml. Error: %q", err)
			continue
		}
		k8sResources = append(k8sResources, currK8sResources...)
	}

	// Run transformations on k8s resources
	transforms, err := gettransformdata.GetTransformsFromPaths(transformPaths, transformations.AnswerFn, transformations.AskStaticQuestion, transformations.AskDynamicQuestion)
	if err != nil {
		log.Fatalf("Failed to get the transformations. Error: %q", err)
	}
	transformedK8sResources, err := runtransforms.ApplyTransforms(transforms, k8sResources)
	if err != nil {
		log.Fatalf("Failed to apply the transformations. Error: %q", err)
	}
	return starlark.WriteResources(transformedK8sResources, outputPath)
}
