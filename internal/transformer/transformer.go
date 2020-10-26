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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/transformer/templates"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate go run github.com/konveyor/move2kube/internal/common/generator templates

// Transformer translates intermediate representation to destination artifacts
type Transformer interface {
	// Transform translates intermediate representation to destination objects
	Transform(ir irtypes.IR) error
	// WriteObjects writes Transformed objects to filesystem
	WriteObjects(outDirectory string) error
}

// GetTransformer returns a transformer that is suitable for an IR
func GetTransformer(ir irtypes.IR) Transformer {
	if ir.Kubernetes.ArtifactType == plan.Knative {
		return &KnativeTransformer{}
	}
	return &K8sTransformer{}
}

// writeContainers returns true if any scripts were written
func writeContainers(containers []irtypes.Container, outpath, rootDir, registryURL, registryNamespace string) bool {
	containersdirectory := "containers"
	containerspath := path.Join(outpath, containersdirectory)
	log.Debugf("containerspath %s", containerspath)
	err := os.MkdirAll(containerspath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create directory %s : %s", containerspath, err)
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
		writepath := filepath.Join(outpath, "buildimages.sh")
		err := common.WriteTemplateToFile(templates.Buildimages_sh, buildScriptMap, writepath, common.DefaultExecutablePermission)
		if err != nil {
			log.Errorf("Unable to create script to build images : %s", err)
		}

		writepath = filepath.Join(outpath, "copysources.sh")
		err = common.WriteTemplateToFile(templates.CopySources_sh, struct {
			Src string
			Dst string
		}{
			Src: "$1", //Parameterized source folder
			Dst: containersdirectory,
		}, writepath, common.DefaultExecutablePermission)
		if err != nil {
			log.Errorf("Unable to create script to build images : %s", err)
		}
	}
	if len(dockerimages) > 0 {
		writepath := filepath.Join(outpath, "pushimages.sh")
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

func writeTransformedObjects(path string, objs []runtime.Object) ([]string, error) {
	fileswritten := []string{}
	err := os.MkdirAll(path, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create directory %s : %s", path, err)
		return nil, err
	}
	for _, obj := range objs {
		//Encode object
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
		//Write to file
		data := b.Bytes()
		val := reflect.ValueOf(obj).Elem()
		typeMeta := val.FieldByName("TypeMeta").Interface().(metav1.TypeMeta)
		objectMeta := val.FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
		file := fmt.Sprintf("%s-%s.yaml", objectMeta.Name, strings.ToLower(typeMeta.Kind))
		file = filepath.Join(path, file)
		if err := ioutil.WriteFile(file, data, common.DefaultFilePermission); err != nil {
			log.Errorf("Failed to write %q Error: %q", typeMeta.Kind, err)
			continue
		}
		fileswritten = append(fileswritten, file)
		log.Debugf("%q created", file)
	}
	return fileswritten, nil
}
