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

package common

import (
	"fmt"

	"github.com/konveyor/move2kube/transformer/types"
	"github.com/sirupsen/logrus"
)

// GetInfoFromK8sResource returns some useful information given a k8s resource
func GetInfoFromK8sResource(k8sResource types.K8sResourceT) (kind string, apiVersion string, name string, err error) {
	logrus.Trace("start getInfoFromK8sResource")
	defer logrus.Trace("end getInfoFromK8sResource")
	kindI, ok := k8sResource["kind"]
	if !ok {
		return "", "", "", fmt.Errorf("There is no kind specified in the k8s resource %+v", k8sResource)
	}
	kind, ok = kindI.(string)
	if !ok {
		return "", "", "", fmt.Errorf("Expected kind to be of type string. Actual value %+v is of type %T", kindI, kindI)
	}
	apiVersionI, ok := k8sResource["apiVersion"]
	if !ok {
		return kind, "", "", fmt.Errorf("There is no apiVersion specified in the k8s resource %+v", k8sResource)
	}
	apiVersion, ok = apiVersionI.(string)
	if !ok {
		return kind, "", "", fmt.Errorf("Expected apiVersion to be of type string. Actual value %+v is of type %T", apiVersionI, apiVersionI)
	}
	metadataI, ok := k8sResource["metadata"]
	if !ok {
		return kind, apiVersion, "", fmt.Errorf("There is no metadata specified in the k8s resource %+v", k8sResource)
	}
	name, err = getNameFromMetadata(metadataI)
	if err != nil {
		return kind, apiVersion, "", err
	}
	return kind, apiVersion, name, nil
}

func getNameFromMetadata(metadataI interface{}) (string, error) {
	metadata, ok := metadataI.(map[interface{}]interface{})
	if !ok {
		metadata, ok := metadataI.(types.MapT)
		if !ok {
			return "", fmt.Errorf("Expected metadata to be of map type. Actual value %+v is of type %T", metadataI, metadataI)
		}
		nameI, ok := metadata["name"]
		if !ok {
			return "", fmt.Errorf("There is no name specified in the k8s resource metadata %+v", metadata)
		}
		name, ok := nameI.(string)
		if !ok {
			return "", fmt.Errorf("Expected name to be of type string. Actual value %+v is of type %T", nameI, nameI)
		}
		return name, nil
	}
	nameI, ok := metadata["name"]
	if !ok {
		return "", fmt.Errorf("There is no name specified in the k8s resource metadata %+v", metadata)
	}
	name, ok := nameI.(string)
	if !ok {
		return "", fmt.Errorf("Expected name to be of type string. Actual value %+v is of type %T", nameI, nameI)
	}
	return name, nil
}
