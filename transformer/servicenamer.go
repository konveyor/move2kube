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
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

type service struct {
	path       string
	pathsuffix string
}

func nameServices(projName string, services map[string][]plantypes.PlanArtifact) map[string][]plantypes.PlanArtifact {
	sts := services[""]
	delete(services, "")
	// Collate services by service dir path or shared common base dir
	knownServicePaths := map[string]string{} //[path]name
	for sn, s := range services {
		for _, st := range s {
			if pps, ok := st.Paths[artifacts.ServiceDirPathType]; ok {
				for _, pp := range pps {
					knownServicePaths[pp] = sn
				}
			}
		}
	}
	// Collate services by service dir path or shared common base dir
	servicePaths := map[string][]plantypes.PlanArtifact{}
	for _, st := range sts {
		pps, ok := st.Paths[artifacts.ServiceDirPathType]
		bpp := common.CleanAndFindCommonDirectory(pps)
		if !ok {
			paths := []string{}
			for _, p := range st.Paths {
				paths = append(paths, p...)
			}
			if len(paths) > 0 {
				bpp = common.CleanAndFindCommonDirectory(paths)
			} else {
				logrus.Errorf("No paths in the transformer. Ignoring transformer : %+v", st)
				continue
			}
		}
		found := false
		for kpp, sn := range knownServicePaths {
			if common.IsParent(bpp, kpp) {
				services[sn] = append(services[sn], st)
				found = true
			}
		}
		if !found {
			servicePaths[bpp] = append(servicePaths[bpp], st)
		}
	}
	// Find if base dir is a git repo, and has only one service or many services
	gitRepoNames := map[string][]string{} // [repoName][]basePath
	basePathRepos := map[string]string{}
	for sp := range servicePaths {
		repoName, _, _, repoURL, _, err := common.GatherGitInfo(sp)
		if err != nil {
			logrus.Debugf("Unable to find any git repo for directory %s : %s", sp, err)
			continue
		}
		if repoName == "" {
			logrus.Debugf("No repo name found for repo at %s", repoURL)
			continue
		}
		if bps, ok := gitRepoNames[repoName]; ok {
			gitRepoNames[repoName] = append(bps, sp)
		} else {
			gitRepoNames[repoName] = []string{sp}
		}
		basePathRepos[sp] = repoName
	}
	for repoName, basePaths := range gitRepoNames {
		if len(basePaths) == 1 {
			// Only one service in repo
			services[repoName] = servicePaths[basePaths[0]]
			delete(servicePaths, basePaths[0])
		}
	}
	// Only one set of unnamed services, use service name
	if len(services) == 0 && len(servicePaths) == 1 {
		for _, ts := range servicePaths {
			services[projName] = ts
		}
		return services
	}

	repoServices := map[string][]service{}
	for sp := range servicePaths {
		repo, ok := basePathRepos[sp]
		if !ok {
			repo = projName
		}
		p := service{sp, sp}
		if ps, ok := repoServices[repo]; !ok {
			repoServices[repo] = []service{p}
		} else {
			repoServices[repo] = append(ps, p)
		}
	}
	sServices := map[string][]service{}
	for repo, services := range repoServices {
		if len(services) == 1 {
			sServices[repo] = []service{services[0]}
			continue
		}
		for k, v := range bucketServices(services) {
			separator := "-"
			if repo == "" || k == "" {
				separator = ""
			}
			nk := repo + separator + k
			if v1, ok := sServices[nk]; ok {
				sServices[nk] = append(v, v1...)
			} else {
				sServices[nk] = v
			}
		}
	}
	//TODO: Consider whether we should take into consideration pre-existing serviceNames
	svcs := map[string][]plantypes.PlanArtifact{}
	for sn, ps := range sServices {
		sn = common.NormalizeForMetadataName(sn)
		for _, p := range ps {
			svcs[sn] = servicePaths[p.path]
		}
	}
	return plantypes.MergeServices(services, svcs)
}

func bucketServices(services []service) map[string][]service {
	nServices := map[string][]service{}
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
			nServices[prefix] = []service{df}
		} else {
			nServices[prefix] = append(pdfs, df)
		}
	}
	sServicess := map[string][]service{}
	for p, paths := range nServices {
		if len(paths) == 1 {
			sServicess[p] = []service{paths[0]}
		} else if p == "" {
			for _, v := range paths {
				if v1, ok := sServicess[p]; ok {
					sServicess[p] = append(v1, v)
				} else {
					sServicess[p] = []service{v}
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

func findCommonPrefix(files []service) string {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.pathsuffix
	}
	return common.CleanAndFindCommonDirectory(paths)
}

func trimPrefix(files []service, prefix string) []service {
	for i, f := range files {
		files[i].pathsuffix = strings.TrimPrefix(f.pathsuffix, prefix+string(filepath.Separator))
	}
	return files
}
