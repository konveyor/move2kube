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

package metadata_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/move2kube/internal/metadata"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type K8sFilesLoaderTestSuite struct {
	suite.Suite

	loader metadata.K8sFilesLoader
	plan   plantypes.Plan
}

// SetupSuite runs before the tests in the suite are run
func (*K8sFilesLoaderTestSuite) SetupSuite() {
	log.SetLevel(log.DebugLevel)
}

// SetupTest runs before each test
func (s *K8sFilesLoaderTestSuite) SetupTest() {
	s.loader = metadata.K8sFilesLoader{}
	s.plan = plantypes.NewPlan()
}

func (s *K8sFilesLoaderTestSuite) TestEmptyDir() {
	// git fails to handle empty directories so create temporary directory
	dir := s.T().TempDir()
	want := plantypes.NewPlan()
	s.NoError(s.loader.UpdatePlan(dir, &s.plan))
	s.Equal(want, s.plan)
}

func (s *K8sFilesLoaderTestSuite) copyfile(src, dst string) {
	source, err := os.Open(src)
	s.NoError(err)
	defer source.Close()
	destination, err := os.Create(dst)
	s.NoError(err)
	defer destination.Close()
	_, err = io.Copy(destination, source)
	s.NoError(err)
}

func (s *K8sFilesLoaderTestSuite) TestBadPerm() {
	// git poorly handles directory permissions so create temporary directory
	dir := s.T().TempDir()
	s.copyfile("testdata/k8s/valid/valid.yaml", filepath.Join(dir, "valid.yml"))
	err := os.Chmod(dir, 0) // d---------
	s.NoError(err)

	want := plantypes.NewPlan()
	s.Error(s.loader.UpdatePlan(dir, &s.plan))
	s.Equal(want, s.plan)
	err = os.Chmod(dir, os.ModePerm) // restore permissions to safely remove dir on cleanup
	s.NoError(err)
}

func (s *K8sFilesLoaderTestSuite) TestInvalid() {
	want := plantypes.NewPlan()
	s.NoError(s.loader.UpdatePlan("testdata/k8s/invalid", &s.plan))
	s.Equal(want, s.plan)
}

func (s *K8sFilesLoaderTestSuite) TestNonYaml() {
	want := plantypes.NewPlan()
	s.NoError(s.loader.UpdatePlan("testdata/k8s/nonyaml", &s.plan))
	s.Equal(want, s.plan)
}

func (s *K8sFilesLoaderTestSuite) TestValid() {
	want := plantypes.NewPlan()
	want.Spec.Inputs.K8sFiles = []string{"testdata/k8s/valid/valid.yaml"}
	s.NoError(s.loader.UpdatePlan("testdata/k8s/valid", &s.plan))
	s.Equal(want, s.plan)
}

func (s *K8sFilesLoaderTestSuite) TestValidInvalid() {
	want := plantypes.NewPlan()
	want.Spec.Inputs.K8sFiles = []string{"testdata/k8s/valid_invalid/valid.yaml"}
	s.NoError(s.loader.UpdatePlan("testdata/k8s/valid_invalid", &s.plan))
	s.Equal(want, s.plan)
}

// TestK8sFilesLoader runs test suite
func TestK8sFilesLoader(t *testing.T) {
	suite.Run(t, new(K8sFilesLoaderTestSuite))
}
