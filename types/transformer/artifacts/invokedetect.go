/*
 *  Copyright IBM Corporation 2023
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package artifacts

import (
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// InvokeDetectPathType defines the path type (for use in artifacts) that indicates the directory
	// on which to run the detect function of another transformer.
	InvokeDetectPathType transformertypes.PathType = "InvokeDetect"
)

const (
	// InvokeDetectConfigType is the type for the InvokeDetect type artifact's configuration
	InvokeDetectConfigType transformertypes.ConfigType = "InvokeDetect"
)

// InvokeDetectConfig stores the configuration for an InvokeDetect artifact
type InvokeDetectConfig struct {
	TransformerSelector metav1.LabelSelector `yaml:"transformerSelector" json:"transformerSelector"`
}
