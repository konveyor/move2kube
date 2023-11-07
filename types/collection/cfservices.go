/*
 *  Copyright IBM Corporation 2021
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

package collection

import (
	"github.com/konveyor/move2kube-wasm/types"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v2"
)

// CfServicesMetadataKind defines kind of cf services file
const CfServicesMetadataKind types.Kind = "CfServices"

// CfServices defines definition of cf runtime instance apps file
type CfServices struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             CfServicesSpec `yaml:"spec,omitempty"`
}

// CfServicesSpec stores the data
type CfServicesSpec struct {
	CfServices []cfclient.Service `yaml:"services"`
}

// NewCfServices creates a new instance of CfServices
func NewCfServices() CfServices {
	return CfServices{
		TypeMeta: types.TypeMeta{
			Kind:       string(CfServicesMetadataKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
}
