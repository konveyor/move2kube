/*
Copyright IBM Corporation 2021

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

package transformer

import (
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/sirupsen/logrus"

	plantypes "github.com/konveyor/move2kube/types/plan"
)

type project struct {
	path       string
	pathsuffix string
}

func nameServices(projName string, nServices map[string]plantypes.Service, sts []plantypes.Transformer) (services map[string]plantypes.Service) {
	services = nServices
	// Collate services by project path or shared common base dir
	servicePaths := make(map[string][]plantypes.Transformer)
	for _, st := range sts {
		pps, ok := st.Paths[plantypes.ProjectPathPathType]
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
		if ts, ok := servicePaths[bpp]; ok {
			servicePaths[bpp] = append(ts, st)
		} else {
			servicePaths[bpp] = []plantypes.Transformer{st}
		}
	}
	// Find if base dir is a git repo, and has only one service or many services
	gitRepoNames := make(map[string][]string) // [repoName][]basePath
	basePathRepos := make(map[string]string)
	for sp := range servicePaths {
		repoName, _, _, repoUrl, _, err := common.GatherGitInfo(sp)
		if err != nil {
			logrus.Debugf("Unable to find any git repo for directory %s : %s", sp, err)
			continue
		}
		if repoName == "" {
			logrus.Debugf("No repo name found for repo at %s", repoUrl)
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
	// Only one set of unnamed services, use project name
	if len(nServices) == 0 && len(servicePaths) == 1 {
		for _, ts := range servicePaths {
			services[projName] = ts
		}
		return services
	}

	repoProjects := map[string][]project{}
	for sp := range servicePaths {
		repo, ok := basePathRepos[sp]
		if !ok {
			repo = projName
		}
		p := project{sp, sp}
		if ps, ok := repoProjects[repo]; !ok {
			repoProjects[repo] = []project{p}
		} else {
			repoProjects[repo] = append(ps, p)
		}
	}
	sProjects := map[string][]project{}
	for repo, projects := range repoProjects {
		if len(projects) == 1 {
			sProjects[repo] = []project{projects[0]}
			continue
		}
		for k, v := range bucketProjects(projects) {
			separator := "-"
			if repo == "" || k == "" {
				separator = ""
			}
			nk := repo + separator + k
			if v1, ok := sProjects[nk]; ok {
				sProjects[nk] = append(v, v1...)
			} else {
				sProjects[nk] = v
			}
		}
	}
	//TODO: Consider whether we should take into consideration pre-existing serviceNames
	svcs := map[string]plantypes.Service{}
	for sn, ps := range sProjects {
		for _, p := range ps {
			svcs[sn] = servicePaths[p.path]
		}
	}
	return plantypes.MergeServices(services, svcs)
}

func bucketProjects(projects []project) map[string][]project {
	nProjects := map[string][]project{}
	commonPath := findCommonPrefix(projects)
	if commonPath != "." {
		projects = trimPrefix(projects, commonPath)
	}
	for _, df := range projects {
		parts := strings.Split(df.pathsuffix, string(filepath.Separator))
		prefix := ""
		if len(parts) == 0 {
			prefix = ""
		} else if len(parts) > 0 {
			prefix = parts[0]
		}
		if pdfs, ok := nProjects[prefix]; !ok {
			nProjects[prefix] = []project{df}
		} else {
			nProjects[prefix] = append(pdfs, df)
		}
	}
	sProjects := map[string][]project{}
	for p, paths := range nProjects {
		if len(paths) == 1 {
			sProjects[p] = []project{paths[0]}
		} else if p == "" {
			for _, v := range paths {
				if v1, ok := sProjects[p]; ok {
					sProjects[p] = append(v1, v)
				} else {
					sProjects[p] = []project{v}
				}
			}
		} else {
			for k, v := range bucketProjects(paths) {
				separator := "-"
				if p == "" || k == "" {
					separator = ""
				}
				nk := p + separator + k
				if v1, ok := sProjects[nk]; ok {
					sProjects[nk] = append(v, v1...)
				} else {
					sProjects[nk] = v
				}
			}
		}
	}
	return sProjects
}

func findCommonPrefix(files []project) string {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.pathsuffix
	}
	return common.CleanAndFindCommonDirectory(paths)
}

func trimPrefix(files []project, prefix string) []project {
	for i, f := range files {
		files[i].pathsuffix = strings.TrimPrefix(f.pathsuffix, prefix+string(filepath.Separator))
	}
	return files
}
