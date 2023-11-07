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

package transformer

import (
	"github.com/konveyor/move2kube-wasm/common"
	plantypes "github.com/konveyor/move2kube-wasm/types/plan"
	"github.com/konveyor/move2kube-wasm/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
)

type servicePathInfo struct {
	path       string
	pathsuffix string
}

func nameServices(projectName string, inputServicesMap map[string][]plantypes.PlanArtifact) map[string][]plantypes.PlanArtifact {
	unnamedServices := inputServicesMap[""]
	delete(inputServicesMap, "")
	logrus.Debug("Collate named services by service dir path or shared common base dir")
	serviceDirToServiceName := map[string]string{}
	for serviceName, namedServices := range inputServicesMap {
		for _, namedService := range namedServices {
			if serviceDirs, ok := namedService.Paths[artifacts.ServiceDirPathType]; ok {
				for _, serviceDir := range serviceDirs {
					serviceDirToServiceName[serviceDir] = serviceName
				}
			}
		}
	}
	logrus.Debug("Collate unnamed services by service dir path or shared common base dir")
	serviceDirToUnnamedServices := map[string][]plantypes.PlanArtifact{}
	for _, unnamedService := range unnamedServices {
		serviceDirs, ok := unnamedService.Paths[artifacts.ServiceDirPathType]
		commonServiceDir := common.CleanAndFindCommonDirectory(serviceDirs)
		if !ok {
			allPathsUsedByService := []string{}
			for _, paths := range unnamedService.Paths {
				allPathsUsedByService = append(allPathsUsedByService, paths...)
			}
			if len(allPathsUsedByService) == 0 {
				logrus.Errorf("failed to find any paths in the service. Ignoring the service: %+v", unnamedService)
				continue
			}
			commonServiceDir = common.CleanAndFindCommonDirectory(allPathsUsedByService)
		}
		found := false
		for serviceDir, serviceName := range serviceDirToServiceName {
			if common.IsParent(commonServiceDir, serviceDir) {
				inputServicesMap[serviceName] = append(inputServicesMap[serviceName], unnamedService)
				found = true
			}
		}
		if !found {
			serviceDirToUnnamedServices[commonServiceDir] = append(serviceDirToUnnamedServices[commonServiceDir], unnamedService)
		}
	}
	logrus.Debug("Find if base dir is a git repo, and has only one service or many services")
	repoNameToDirs := map[string][]string{} // [repoName][]repoDir
	repoDirToName := map[string]string{}
	//TODO: WASI
	//for serviceDir := range serviceDirToUnnamedServices {
	//	repoName, _, _, repoURL, _, err := common.GatherGitInfo(serviceDir)
	//	if err != nil {
	//		logrus.Debugf("failed to find any git repo for directory '%s' . Error: %q", serviceDir, err)
	//		continue
	//	}
	//	if repoName == "" {
	//		logrus.Debugf("no repo name found for the git repo at '%s' . Skipping", repoURL)
	//		continue
	//	}
	//	if repoDirs, ok := repoNameToDirs[repoName]; ok {
	//		repoNameToDirs[repoName] = append(repoDirs, serviceDir)
	//	} else {
	//		repoNameToDirs[repoName] = []string{serviceDir}
	//	}
	//	repoDirToName[serviceDir] = repoName
	//}
	for repoName, repoDirs := range repoNameToDirs {
		if len(repoDirs) == 1 {
			repoDir := repoDirs[0]
			logrus.Debugf("Only one service directory '%s' has the repo name '%s'", repoDir, repoName)
			repoName = common.NormalizeForMetadataName(repoName)
			inputServicesMap[repoName] = serviceDirToUnnamedServices[repoDir]
			delete(serviceDirToUnnamedServices, repoDir)
		}
	}
	if len(inputServicesMap) == 0 && len(serviceDirToUnnamedServices) == 1 {
		logrus.Debug("All remaining unnamed services use the same service directory")
		for _, unnamedServices := range serviceDirToUnnamedServices {
			normalizedProjectName := common.NormalizeForMetadataName(projectName)
			inputServicesMap[normalizedProjectName] = unnamedServices
		}
		return inputServicesMap
	}

	repoNameToServicePathInfos := map[string][]servicePathInfo{}
	for serviceDir := range serviceDirToUnnamedServices {
		repoName, ok := repoDirToName[serviceDir]
		if !ok {
			repoName = projectName
		}
		pathInfo := servicePathInfo{
			path:       serviceDir,
			pathsuffix: serviceDir,
		}
		repoNameToServicePathInfos[repoName] = append(repoNameToServicePathInfos[repoName], pathInfo)
	}
	newNameToServicePathInfos := map[string][]servicePathInfo{}
	for repoName, pathInfos := range repoNameToServicePathInfos {
		if len(pathInfos) == 1 {
			newNameToServicePathInfos[repoName] = []servicePathInfo{pathInfos[0]}
			continue
		}
		for bucketedName, bucketedPathInfos := range bucketServices(pathInfos) {
			separator := "-"
			if repoName == "" || bucketedName == "" {
				separator = ""
			}
			newName := repoName + separator + bucketedName
			if v1, ok := newNameToServicePathInfos[newName]; ok {
				newNameToServicePathInfos[newName] = append(bucketedPathInfos, v1...)
			} else {
				newNameToServicePathInfos[newName] = bucketedPathInfos
			}
		}
	}
	//TODO: Decide whether we should take into consideration pre-existing service names
	outputServicesMap := map[string][]plantypes.PlanArtifact{}
	for newName, pathInfos := range newNameToServicePathInfos {
		normalizedNewName := common.NormalizeForMetadataName(newName)
		for _, pathInfo := range pathInfos {
			outputServicesMap[normalizedNewName] = serviceDirToUnnamedServices[pathInfo.path]
		}
	}
	return plantypes.MergeServices(inputServicesMap, outputServicesMap)
}

func bucketServices(services []servicePathInfo) map[string][]servicePathInfo {
	nServices := map[string][]servicePathInfo{}
	commonPath := findCommonPrefix(services)
	if commonPath != "." {
		services = trimPrefix(services, commonPath)
	}
	for _, df := range services {
		parts := strings.Split(df.pathsuffix, string(filepath.Separator))
		prefix := ""
		if len(parts) == 0 {
			prefix = ""
		} else if len(parts) > 0 {
			prefix = parts[0]
		}
		if pdfs, ok := nServices[prefix]; !ok {
			nServices[prefix] = []servicePathInfo{df}
		} else {
			nServices[prefix] = append(pdfs, df)
		}
	}
	sServicess := map[string][]servicePathInfo{}
	for p, paths := range nServices {
		if len(paths) == 1 {
			sServicess[p] = []servicePathInfo{paths[0]}
		} else if p == "" {
			for _, v := range paths {
				if v1, ok := sServicess[p]; ok {
					sServicess[p] = append(v1, v)
				} else {
					sServicess[p] = []servicePathInfo{v}
				}
			}
		} else {
			for k, v := range bucketServices(paths) {
				separator := "-"
				if p == "" || k == "" {
					separator = ""
				}
				nk := p + separator + k
				if v1, ok := sServicess[nk]; ok {
					sServicess[nk] = append(v, v1...)
				} else {
					sServicess[nk] = v
				}
			}
		}
	}
	return sServicess
}

func findCommonPrefix(files []servicePathInfo) string {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.pathsuffix
	}
	return common.CleanAndFindCommonDirectory(paths)
}

func trimPrefix(files []servicePathInfo, prefix string) []servicePathInfo {
	for i, f := range files {
		files[i].pathsuffix = strings.TrimPrefix(f.pathsuffix, prefix+string(filepath.Separator))
	}
	return files
}
