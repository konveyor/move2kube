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

package api

import (
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/k8sschema"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrintValidate - Print validate output
func PrintValidate(inputPath string) error {
	codecs := serializer.NewCodecFactory(k8sschema.GetSchema())

	filePaths, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Errorf("Unable to fetch yaml files at path %q Error: %q", inputPath, err)
		return err
	}
	for _, filePath := range filePaths {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			logrus.Debugf("Failed to read the yaml file at path %q Error: %q", filePath, err)
			continue
		}
		docs, err := common.SplitYAML(data)
		if err != nil {
			logrus.Debugf("Failed to split the file at path %q into YAML documents. Error: %q", filePath, err)
			continue
		}
		for _, doc := range docs {
			obj, _, err := codecs.UniversalDeserializer().Decode(doc, nil, nil)
			if err != nil {
				continue
			}
			objectMeta := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
			for k, v := range objectMeta.Annotations {
				if strings.HasPrefix(k, common.TODOAnnotation) {
					logrus.Infof("%s : %s", k, v)
				}
			}
		}
	}
	return nil
}
