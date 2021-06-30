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

package analysers

import (
	"encoding/xml"
	"github.com/creekorful/mvnparser"
	"path/filepath"
	"io/ioutil"
	"github.com/konveyor/move2kube/environment"
	//"github.com/konveyor/move2kube/internal/common"
	//"github.com/konveyor/move2kube/internal/transformer/classes/analysers/compose"
	//collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	SpringbootServiceConfigType transformertypes.ConfigType = "SpringbootService"
)

const (
	// composeFilePathType defines the source artifact type of Docker compose
	mavenPomXML transformertypes.PathType = "MavenPomXML"
	// imageInfoPathType defines the source artifact type of image info
	//imageInfoPathType transformertypes.PathType = "ImageInfo"
)

// SpringbootAnalyser implements Transformer interface
type SpringbootAnalyser struct {
	Config transformertypes.Transformer
	Env    environment.Environment
}

type SpringbootConfig struct {
	ServiceName string `yaml:"serviceName,omitempty"`
}

func (t *SpringbootAnalyser) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

func (t *SpringbootAnalyser) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.Config, t.Env
}

func (t *SpringbootAnalyser) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	
	return nil, nil, nil
}

func (t *SpringbootAnalyser) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	
	destEntries, err := ioutil.ReadDir(dir)
		if err != nil {
			logrus.Errorf("Unable to process directory %s : %s", dir, err)
		} else {
			for _, de := range destEntries {
				if de.Name() == "pom.xml"{

					// filled with previously declared xml 
					pomStr,err := ioutil.ReadFile(filepath.Join(dir,de.Name() ))
					if err != nil{
						return nil,nil,err
					}
					
					// Load project from string
					var project mvnparser.MavenProject
					if err := xml.Unmarshal([]byte(pomStr), &project); err != nil {
						logrus.Errorf("unable to unmarshal pom file. Reason: %s", err)
						return nil, nil, err
					}
					
					if len(project.Modules) != 0{
						return nil, nil, nil
					}


					
					ct := plantypes.Transformer{
						Mode:                   plantypes.ModeContainer,
						ArtifactTypes:          []transformertypes.ArtifactType{irtypes.IRArtifactType, artifacts.ContainerBuildArtifactType},
						ExclusiveArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
						Configs: map[transformertypes.ConfigType]interface{}{
							SpringbootServiceConfigType: SpringbootConfig{
								ServiceName: filepath.Base(dir),
							}},
						Paths: map[transformertypes.PathType][]string{
							mavenPomXML: {
							filepath.Join(dir, "pom.xml"),
							},
							artifacts.ProjectPathPathType: {dir},

						},
					}

					

					return map[string]plantypes.Service {filepath.Base(dir):[]plantypes.Transformer{ct}}, nil, nil
				}

				

				
			}
		}

	
	return nil, nil, nil
}

func (t *SpringbootAnalyser) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	
	return nil, nil, nil
}



