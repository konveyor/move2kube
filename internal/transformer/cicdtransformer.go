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
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/konveyor/move2kube/internal/apiresourceset"
	common "github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
)

// CICDTransformer creates the necessary tekton artifacts needed for CI/CD.
type CICDTransformer struct {
	cachedObjs []runtime.Object
}

// Transform creates tekton resources needed for CI/CD
func (cicd *CICDTransformer) Transform(ir irtypes.IR) error {
	cicd.cachedObjs = new(apiresourceset.TektonAPIResourceSet).CreateAPIResources(ir)
	return nil
}

// WriteObjects writes the CI/CD artifacts to files
func (cicd *CICDTransformer) WriteObjects(outDirectory string) error {
	cicdPath := filepath.Join(outDirectory, "cicd")
	if err := os.MkdirAll(cicdPath, common.DefaultDirectoryPermission); err != nil {
		log.Fatalf("Failed to create the CI/CD directory at path %q. Error: %q", cicdPath, err)
		return err
	}
	if _, err := writeTransformedObjects(cicdPath, cicd.cachedObjs); err != nil {
		log.Errorf("Error occurred while writing transformed objects. Error: %q", err)
		return err
	}
	return nil
}
