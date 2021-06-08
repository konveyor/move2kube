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

package transformer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	transformercommon "github.com/konveyor/move2kube/transformer/common"
	"github.com/konveyor/move2kube/transformer/types"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// WriteResources writes out k8s resources to a given directory.
// It will create the output directory if it doesn't exist.
func WriteResources(k8sResources []types.K8sResourceT, outputPath string) ([]string, error) {
	log.Trace("start WriteResources")
	defer log.Trace("end WriteResources")
	if err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission); err != nil {
		return nil, err
	}
	filesWritten := []string{}
	for _, k8sResource := range k8sResources {
		filename, err := getFilename(k8sResource)
		if err != nil {
			continue
		}
		fileOutputPath := filepath.Join(outputPath, filename)
		if err := WriteResource(k8sResource, fileOutputPath); err != nil {
			continue
		}
		filesWritten = append(filesWritten, fileOutputPath)
	}
	return filesWritten, nil
}

func getFilename(k8sResource types.K8sResourceT) (string, error) {
	log.Trace("start getFilename")
	defer log.Trace("end getFilename")
	kind, _, name, err := transformercommon.GetInfoFromK8sResource(k8sResource)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s.yaml", name, strings.ToLower(kind)), nil
}

// WriteResource writes out a k8s resource to a given file path.
func WriteResource(k8sResource types.K8sResourceT, outputPath string) error {
	log.Trace("start WriteResource")
	defer log.Trace("end WriteResource")
	yamlBytes, err := yaml.Marshal(k8sResource)
	if err != nil {
		log.Error("Error while Encoding object")
		return err
	}
	return ioutil.WriteFile(outputPath, yamlBytes, common.DefaultFilePermission)
}
