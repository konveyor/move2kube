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

package getparameterizers

import (
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/newparameterizer/types"
	log "github.com/sirupsen/logrus"
)

// GetParameterizers returns the paramterizers given a file path
func GetParameterizers(parameterizerPath string) ([]types.ParameterizerT, error) {
	log.Trace("start GetParameterizers")
	defer log.Trace("end GetParameterizers")
	parameterizerPaths, err := common.GetFilesByExt(parameterizerPath, []string{".yaml"})
	if err != nil {
		log.Errorf("failed to get all the parameterizer yaml files from the path %s . Error: %q", parameterizerPath, err)
		return nil, err
	}
	return GetParameterizersFromPaths(parameterizerPaths)
}

// GetParameterizersFromPaths returns the parameterizers given a list of script file paths
func GetParameterizersFromPaths(parameterizerPaths []string) ([]types.ParameterizerT, error) {
	log.Trace("start GetParameterizersFromPaths")
	defer log.Trace("end GetParameterizersFromPaths")
	parameterizers := []types.ParameterizerT{}
	for _, parameterizerPath := range parameterizerPaths {
		currParameterizers, err := GetParameterizersFromPath(parameterizerPath)
		if err != nil {
			log.Errorf("failed to get the parameterizers from the file at path %s Error: %q", parameterizerPath, err)
			continue
		}
		parameterizers = append(parameterizers, currParameterizers...)
	}
	return parameterizers, nil
}

// GetParameterizersFromPath gets a list of parameterizers given a file path
func GetParameterizersFromPath(parameterizerPath string) ([]types.ParameterizerT, error) {
	log.Trace("start GetParameterizersFromPath")
	defer log.Trace("end GetParameterizersFromPath")
	return new(SimpleParameterizerT).GetParameterizersFromPath(parameterizerPath)
}
