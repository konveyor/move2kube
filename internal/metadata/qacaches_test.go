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

	log "github.com/sirupsen/logrus"

	"github.com/konveyor/move2kube/internal/metadata"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/stretchr/testify/suite"
)

type QACachesLoaderTestSuite struct {
	suite.Suite

	loader metadata.QACacheLoader
	plan   plantypes.Plan
}

// SetupSuite runs before the tests in the suite are run
func (*QACachesLoaderTestSuite) SetupSuite() {
	log.SetLevel(log.DebugLevel)
}

// SetupTest runs before each test
func (s *QACachesLoaderTestSuite) SetupTest() {
	s.loader = metadata.QACacheLoader{}
	s.plan = plantypes.NewPlan()
}

func (s *QACachesLoaderTestSuite) TestEmptyDir() {
	// git fails to handle empty directories so create temporary directory
	dir := s.T().TempDir()
	want := plantypes.NewPlan()
	s.NoError(s.loader.UpdatePlan(dir, &s.plan))
	s.Equal(want, s.plan)
}

func (s *QACachesLoaderTestSuite) TestFile() {
	want := plantypes.NewPlan()
	want.Spec.Inputs.QACaches = []string{"testdata/qa/valid/valid.yaml"}
	s.NoError(s.loader.UpdatePlan("testdata/qa/valid/valid.yaml", &s.plan))
	s.Equal(want, s.plan)
}

func (s *QACachesLoaderTestSuite) copyfile(src, dst string) {
	source, err := os.Open(src)
	s.NoError(err)
	defer source.Close()
	destination, err := os.Create(dst)
	s.NoError(err)
	defer destination.Close()
	_, err = io.Copy(destination, source)
	s.NoError(err)
}

func (s *QACachesLoaderTestSuite) TestBadPerm() {
	// git poorly handles directory permissions so create temporary directory
	dir := s.T().TempDir()
	s.copyfile("testdata/qa/valid/valid.yaml", filepath.Join(dir, "valid.yml"))
	err := os.Chmod(dir, 0) // d---------
	s.NoError(err)

	want := plantypes.NewPlan()
	s.Error(s.loader.UpdatePlan(dir, &s.plan))
	s.Equal(want, s.plan)
	err = os.Chmod(dir, os.ModePerm) // restore permissions to safely remove dir on cleanup
	s.NoError(err)
}

func (s *QACachesLoaderTestSuite) TestInvalid() {
	want := plantypes.NewPlan()
	s.NoError(s.loader.UpdatePlan("testdata/qa/invalid", &s.plan))
	s.Equal(want, s.plan)
}

func (s *QACachesLoaderTestSuite) TestNonYaml() {
	want := plantypes.NewPlan()
	s.NoError(s.loader.UpdatePlan("testdata/qa/nonyaml", &s.plan))
	s.Equal(want, s.plan)
}

func (s *QACachesLoaderTestSuite) TestValid() {
	want := plantypes.NewPlan()
	want.Spec.Inputs.QACaches = []string{"testdata/qa/valid/valid.yaml"}
	s.NoError(s.loader.UpdatePlan("testdata/qa/valid", &s.plan))
	s.Equal(want, s.plan)
}

func (s *QACachesLoaderTestSuite) TestValidInvalid() {
	want := plantypes.NewPlan()
	want.Spec.Inputs.QACaches = []string{"testdata/qa/valid_invalid/valid.yaml"}
	s.NoError(s.loader.UpdatePlan("testdata/qa/valid_invalid", &s.plan))
	s.Equal(want, s.plan)
}

func (s *QACachesLoaderTestSuite) TestLoadToIRForQA() {
	s.plan.Spec.Inputs.QACaches = []string{"testdata/qa/valid/valid.yaml"}
	ir := irtypes.NewIR(s.plan)
	s.NoError(s.loader.LoadToIR(s.plan, &ir))
}

// TestQACachesLoader runs test suite
func TestQACachesLoader(t *testing.T) {
	suite.Run(t, new(QACachesLoaderTestSuite))
}
