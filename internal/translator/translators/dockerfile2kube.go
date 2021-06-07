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

package translator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	dockerparser "github.com/moby/buildkit/frontend/dockerfile/parser"
	log "github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// DockerfileTranslator implements Translator interface for using preexisting dockerfiles
type DockerfileTranslator struct {
}

// GetServiceOptions - output a plan based on the input directory contents
func (dockerfileTranslator *DockerfileTranslator) GetServiceOptions(inputPath string, plan plantypes.Plan) (map[string]plantypes.Service, error) {
	services := map[string]plantypes.Service{}
	sdfs, err := getDockerfileServices(inputPath, plan.Name)
	if err != nil {
		log.Errorf("Unable to get Dockerfiles : %s", err)
		return services, err
	}
	for sn, dfs := range sdfs {
		ns := plantypes.Service{
			ContainerizationOptions: make([]plantypes.ContainerizationOption, len(dfs)),
			SourceArtifacts:         make([]plantypes.SourceArtifact, len(dfs)),
		}
		for dfi, df := range dfs {
			var repoInfo plantypes.RepoInfo
			if gitURL, gitBranch, err := common.GatherGitInfo(dfs[0].path); err != nil {
				log.Warnf("Error while parsing the git repo at path %q Error: %q", dfs[0].path, err)
			} else {
				repoInfo = plantypes.RepoInfo{
					GitRepoURL:    gitURL,
					GitRepoBranch: gitBranch,
				}
			}
			ns.ContainerizationOptions[dfi] = plantypes.ContainerizationOption{
				BuildType:   plantypes.DockerFileContainerBuildTypeValue,
				ContextPath: df.context,
				ID:          df.path,
				RepoInfo:    repoInfo,
			}
			ns.SourceArtifacts[dfi] = plantypes.SourceArtifact{
				Type:      plantypes.DockerfileArtifactType,
				ID:        df.path,
				Artifacts: []string{df.path},
			}
		}
		services[sn] = ns
	}
	return services, nil
}

// Translate translates artifacts to IR
func (dockerfileTranslator *DockerfileTranslator) Translate(serviceName string, service plantypes.Service, plan plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(plan)
	if len(service.ContainerizationOptions) == 0 {
		log.Debugf("The service %s has no containerization target options. Skipping.", serviceName)
		continue
	}
	log.Debugf("Translating %s", serviceName)
	irContainer, err := new(containerizer.ReuseDockerfileContainerizer).GetContainer(plan, service)
	if err != nil {
		log.Warnf("Unable to get reuse the Dockerfile for service %s even though build parameters are present. Error: %q", serviceName, err)
		continue
	}
	irContainer.RepoInfo = service.RepoInfo
	irContainer.RepoInfo.TargetPath = service.ContainerizationOptions[0]
	ir.AddContainer(irContainer)

	irService := irtypes.NewServiceFromPlanService(service)
	container := core.Container{Name: serviceName, Image: service.Image}
	for _, port := range irContainer.ExposedPorts {
		// Add the port to the k8s pod.
		container.Ports = append(container.Ports, core.ContainerPort{ContainerPort: int32(port)})
		// Forward the port on the k8s service to the k8s pod.
		podPort := irtypes.Port{Number: int32(port)}
		servicePort := podPort
		irService.AddPortForwarding(servicePort, podPort)
	}
	irService.Containers = []core.Container{container}
	ir.Services[serviceName] = irService
	return ir, nil
}

func isDockerFile(path string) (isDockerfile bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		log.Debugf("Unable to open file %s : %s", path, err)
		return false, err
	}
	defer f.Close()
	res, err := dockerparser.Parse(f)
	if err != nil {
		log.Debugf("Unable to parse file %s as Docker files : %s", path, err)
		return false, err
	}
	for _, dfchild := range res.AST.Children {
		if dfchild.Value == "from" {
			r := regexp.MustCompile(`(?i)FROM\s+(--platform=[^\s]+)?[^\s]+(\s+AS\s+[^\s]+)?\s*(#.+)?$`)
			if r.MatchString(dfchild.Original) {
				log.Debugf("Identified a docker file : " + path)
				return true, nil
			}
			return false, nil
		}
		if dfchild.Value == "arg" {
			continue
		}
		return false, fmt.Errorf("%s is not a valid Dockerfile", path)
	}
	return false, fmt.Errorf("%s is not a valid Dockerfile", path)
}

func getDockerfileServices(inputpath string, projName string) (sDockerfiles map[string][]dockerfile, err error) {
	if info, err := os.Stat(inputpath); os.IsNotExist(err) {
		log.Warnf("Error in walking through files due to : %s", err)
		return sDockerfiles, err
	} else if !info.IsDir() {
		log.Warnf("The path %q is not a directory.", inputpath)
	}
	files := []string{}
	err = filepath.Walk(inputpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("Skipping path %s due to error: %s", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		if isdf, _ := isDockerFile(path); isdf {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Warnf("Error in walking through files due to : %s", err)
	}
	log.Debugf("No of dockerfiles identified : %d", len(files))
	repoDockerfiles := map[string][]dockerfile{}
	for _, f := range files {
		repo, context := common.GetGitRepoName(filepath.Dir(f))
		if repo == "" {
			repo = projName
			context = inputpath
		}
		df := dockerfile{f, f, context}
		if dfs, ok := repoDockerfiles[repo]; !ok {
			repoDockerfiles[repo] = []dockerfile{df}
		} else {
			repoDockerfiles[repo] = append(dfs, df)
		}
	}
	sDockerfiles = map[string][]dockerfile{}
	for repo, dfs := range repoDockerfiles {
		if len(dfs) == 1 {
			sDockerfiles[repo] = []dockerfile{dfs[0]}
			continue
		}
		for k, v := range bucketDFs(dfs) {
			separator := "-"
			if repo == "" || k == "" {
				separator = ""
			}
			nk := repo + separator + k
			if v1, ok := sDockerfiles[nk]; ok {
				sDockerfiles[nk] = append(v, v1...)
			} else {
				sDockerfiles[nk] = v
			}
		}
	}
	return sDockerfiles, nil
}

type dockerfile struct {
	path       string
	pathsuffix string
	context    string
}

func bucketDFs(dfs []dockerfile) map[string][]dockerfile {
	nDockerfiles := map[string][]dockerfile{}
	commonPath := findCommonPrefix(dfs)
	if commonPath != "." {
		dfs = trimPrefix(dfs, commonPath)
	}
	for _, df := range dfs {
		parts := strings.Split(df.pathsuffix, string(filepath.Separator))
		prefix := ""
		if len(parts) == 1 {
			prefix = ""
		} else if len(parts) > 1 {
			prefix = parts[0]
			df.context = strings.TrimSuffix(df.path, df.pathsuffix) + parts[0]
		}
		if pdfs, ok := nDockerfiles[prefix]; !ok {
			nDockerfiles[prefix] = []dockerfile{df}
		} else {
			nDockerfiles[prefix] = append(pdfs, df)
		}
	}
	sDockerfiles := map[string][]dockerfile{}
	for p, dfiles := range nDockerfiles {
		if len(dfiles) == 1 {
			sDockerfiles[p] = []dockerfile{dfiles[0]}
		} else if p == "" {
			for _, v := range dfiles {
				if v1, ok := sDockerfiles[p]; ok {
					sDockerfiles[p] = append(v1, v)
				} else {
					sDockerfiles[p] = []dockerfile{v}
				}
			}
		} else {
			for k, v := range bucketDFs(dfiles) {
				separator := "-"
				if p == "" || k == "" {
					separator = ""
				}
				nk := p + separator + k
				if v1, ok := sDockerfiles[nk]; ok {
					sDockerfiles[nk] = append(v, v1...)
				} else {
					sDockerfiles[nk] = v
				}
			}
		}
	}
	return sDockerfiles
}

func checkForDockerfile(path string) bool {
	finfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("There is no file at path %s Error: %q", path, err)
			return false
		}
		log.Errorf("There was an error accessing the file at path %s Error: %q", path, err)
		return false
	}
	if finfo.IsDir() {
		log.Errorf("The path %s points to a directory. Expected a Dockerfile.", path)
		return false
	}
	return true
}
