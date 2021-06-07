/*
Copyright IBM Corporation 2020, 2021

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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	"github.com/konveyor/move2kube/internal/translator/classes"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
)

var (
	translatorTypes map[string]reflect.Type = make(map[string]reflect.Type)
	translators     map[string]Translator   = make(map[string]Translator)
)

// Translator interface defines translator that translates files and converts it to ir representation
type Translator interface {
	Init(tc translatortypes.Translator) error
	GetConfig() translatortypes.Translator

	BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error)
	PlanDetect(plantypes.Plan) ([]plantypes.Translator, error)

	Translate(serviceName string) map[string]translatortypes.Patch
}

func init() {
	translatorObjs := []Translator{new(classes.GoInterface)}
	for _, tt := range translatorObjs {
		t := reflect.TypeOf(tt)
		tn := t.Name()
		if ot, ok := translatorTypes[tn]; ok {
			log.Errorf("Two translator classes have the same name %s : %T, %T; Ignoring %T", tn, ot, t, t)
			continue
		}
		translatorTypes[tn] = t
	}
}

func Init(assetsPath string) error {
	filePaths, err := common.GetFilesByExt(assetsPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Warnf("Unable to fetch yaml files and recognize cf manifest yamls at path %q Error: %q", assetsPath, err)
		return err
	}
	translatorConfigs := make(map[string]translatortypes.Translator)
	for _, filePath := range filePaths {
		tc, err := getTranslatorConfig(filePath)
		if err != nil {
			log.Debugf("Unable to load %s as Translator config", filePath, err)
			continue
		}
		if ot, ok := translatorConfigs[tc.Name]; ok {
			log.Errorf("Found two conflicting translator Names %s : %s, %s. Ignoring %s.", tc.Name, ot.Spec.FilePath, tc.Spec.FilePath)
			continue
		}
		if _, ok := translatorTypes[tc.Spec.Class]; ok {
			translatorConfigs[tc.Name] = tc
			continue
		}
		log.Errorf("Unable to find suitable translator class for translator config at %s", filePath)
	}
	tns := make([]string, len(translatorConfigs))
	for tn := range translatorConfigs {
		tns = append(tns, tn)
	}
	translatorNames := qaengine.FetchMultiSelectAnswer(common.ConfigTranslatorTypesKey, "Select all translator types that you are interested in:", []string{"Services that don't support any of the translator types you are interested in will be ignored."}, tns, tns)
	for _, tn := range translatorNames {
		tc := translatorConfigs[tn]
		t := reflect.New(translatorTypes[tc.Spec.Class]).Interface().(Translator)
		if err := t.Init(tc); err != nil {
			log.Errorf("Unable to initialize translator %s : %s", tc.Name, err)
		}
	}
	return nil
}

func getTranslatorConfig(path string) (translatortypes.Translator, error) {
	tc := translatortypes.Translator{
		Spec: translatortypes.TranslatorSpec{
			Class:    reflect.TypeOf(classes.GoInterface{}).Name(),
			FilePath: path,
		},
	}
	if err := common.ReadMove2KubeYaml(path, &tc); err != nil {
		log.Debugf("Failed to read the translator metadata at path %q Error: %q", path, err)
		return tc, err
	}
	if tc.Kind != translatortypes.TranslatorKind {
		err := fmt.Errorf("the file at path %q is not a valid cluster metadata. Expected kind: %s Actual kind: %s", path, translatortypes.TranslatorKind, tc.Kind)
		log.Debug(err)
		return tc, err
	}
	return tc, nil
}

func GetTranslators() map[string]Translator {
	return translators
}

func GetServices(prjName string, dir string) (services map[string]plantypes.Service) {
	services = make(map[string]plantypes.Service)
	unservices := make([]plantypes.Translator, 0)
	log.Infoln("Planning Translation - Base Directory")
	for _, t := range translators {
		log.Infof("[%T] Planning translation", t)
		nservices, nunservices, err := t.BaseDirectoryDetect(dir)
		if err != nil {
			log.Warnf("[%T] Failed : %s", t, err)
		} else {
			services = plantypes.MergeServices(services, nservices)
			unservices = append(unservices, nunservices...)
			log.Infof("[%T] Done", t)
		}
	}
	log.Infoln("Translation planning - Base Directory done")
	log.Infoln("Planning Translation - Directory Walk")
	nservices, nunservices, err := walkForServices(dir, translators, services)
	if err != nil {
		log.Warnf("Translation planning - Directory Walk failed : %s", err)
	} else {
		services = plantypes.MergeServices(services, nservices)
		unservices = append(unservices, nunservices...)
		log.Infoln("Translation planning - Directory Walk done")
	}
	services = nameServices(prjName, nservices, unservices)
	log.Infoln("Planning Service Augmentors")
	for _, t := range translators {
		log.Debugf("[%T] Planning translation", t)
		for sn, s := range services {
			sts, err := t.ServiceAugmentDetect(sn, s)
			if err != nil {
				log.Warnf("[%T] Failed for service %s : %s", t, sn, err)
			} else {
				services[sn] = append(s, sts...)
			}
		}
		log.Debugf("[%T] Done", t)
	}
	log.Infoln("Service Augmentors planning - done")
	return
}

func GetPlanTranslators(plan plantypes.Plan) (suitableTranslators []plantypes.Translator, err error) {
	log.Infoln("Planning plan translators")
	for _, t := range translators {
		log.Infof("[%T] Planning translation", t)
		ts, err := t.PlanDetect(plan)
		if err != nil {
			log.Warnf("[%T] Failed : %s", t, err)
		} else {
			suitableTranslators = append(suitableTranslators, ts...)
			log.Infof("[%T] Done", t)
		}
	}
	log.Infoln("Plan translator planning - done")
	return suitableTranslators, nil
}

func walkForServices(inputPath string, ts map[string]Translator, bservices map[string]plantypes.Service) (services map[string]plantypes.Service, unservices []plantypes.Translator, err error) {
	services = bservices
	unservices = make([]plantypes.Translator, 0)
	ignoreDirectories, ignoreContents := getIgnorePaths(inputPath)
	knownProjectPaths := make([]string, 0)

	log.Infoln("Planning Translation - Known Directory")
	for sn, s := range bservices {
		for _, t := range s {
			if len(t.Paths) > 0 && len(t.Paths[plantypes.ProjectPathSourceArtifact]) > 0 {
				pps := t.Paths[plantypes.ProjectPathSourceArtifact]
				knownProjectPaths = common.MergeStringSlices(knownProjectPaths, pps...)
				for _, p := range pps {
					for _, t := range translators {
						log.Debugf("[%T] Planning translation", t)
						nservices, nunservices, err := t.KnownDirectoryDetect(p)
						if err != nil {
							log.Warnf("[%T] Failed : %s", t, err)
						} else {
							// TODO: Decide whether recurse on the nservices for project paths
							services = plantypes.MergeServices(services, nservices)
							services = plantypes.MergeServices(services, map[string]plantypes.Service{sn: nunservices})
							log.Debugf("[%T] Done", t)
						}
					}
				}
			}
		}
	}
	log.Infoln("Translation planning - Known Directory done")

	err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("Skipping path %q due to error. Error: %q", path, err)
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if common.IsStringPresent(knownProjectPaths, path) {
			return filepath.SkipDir //TODO: Should we go inside the directory in this case?
		}
		if common.IsStringPresent(ignoreDirectories, path) {
			if common.IsStringPresent(ignoreContents, path) {
				return filepath.SkipDir
			}
			return nil
		}
		log.Debugf("Planning dir translation - %s", path)
		found := false
		for _, t := range translators {
			log.Debugf("[%T] Planning translation", t, path)
			nservices, nunservices, err := t.DirectoryDetect(path)
			if err != nil {
				log.Warnf("[%T] Failed : %s", t, err)
			} else {
				plantypes.MergeServices(services, nservices)
				unservices = append(unservices, nunservices...)
				if len(nservices) > 0 || len(unservices) > 0 {
					found = true
				}
				log.Debugf("[%T] Done", t)
			}
		}
		log.Debugf("Dir translation done - %s", path)
		if !found {
			log.Debugf("No service found in directory %q", path)
			if common.IsStringPresent(ignoreContents, path) {
				return filepath.SkipDir
			}
			return nil
		}
		return filepath.SkipDir // Skip all subdirectories when base directory is a valid package
	})
	if err != nil {
		log.Errorf("Error occurred while walking through the directory at path %q Error: %q", inputPath, err)
	}
	return
}

type project struct {
	path       string
	pathsuffix string
}

func nameServices(projName string, nServices map[string]plantypes.Service, sts []plantypes.Translator) (services map[string]plantypes.Service) {
	services = make(map[string]plantypes.Service)
	// Collate services by project path or shared common base dir
	servicePaths := make(map[string][]plantypes.Translator)
	for _, st := range sts {
		pps, ok := st.Paths[plantypes.ProjectPathSourceArtifact]
		bpp := common.CleanAndFindCommonDirectory(pps)
		if !ok {
			paths := []string{}
			for _, p := range st.Paths {
				paths = append(paths, p...)
			}
			if len(paths) > 0 {
				bpp = common.CleanAndFindCommonDirectory(paths)
			} else {
				log.Errorf("No paths in the translator. Ignoring translator : %+v", st)
				continue
			}
		}
		if ts, ok := servicePaths[bpp]; ok {
			servicePaths[bpp] = append(ts, st)
		} else {
			servicePaths[bpp] = []plantypes.Translator{st}
		}
	}
	// Find if base dir is a git repo, and has only one service or many services
	gitRepoNames := make(map[string][]string) // [repoName][]basePath
	basePathRepos := make(map[string]string)
	for sp := range servicePaths {
		repoName, repoUrl, _, err := common.GetGitRepoName(sp)
		if err != nil {
			log.Debugf("Unable to find any git repo for directory %s : %s", sp, err)
			continue
		}
		if repoName == "" {
			log.Debugf("No repo name found for repo at %s", repoUrl)
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

	// TODO: Have temporarily adopted naming logic from old Dockerfile containerizer. Validate the approach.
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

func getIgnorePaths(inputPath string) (ignoreDirectories []string, ignoreContents []string) {
	filePaths, err := common.GetFilesByName(inputPath, []string{common.IgnoreFilename})
	if err != nil {
		log.Warnf("Unable to fetch .m2kignore files at path %q Error: %q", inputPath, err)
		return ignoreDirectories, ignoreContents
	}
	for _, filePath := range filePaths {
		file, err := os.Open(filePath)
		if err != nil {
			log.Warnf("Failed to open the .m2kignore file at path %q Error: %q", filePath, err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasSuffix(line, "*") {
				line = strings.TrimSuffix(line, "*")
				path := filepath.Join(filepath.Dir(filePath), line)
				ignoreContents = append(ignoreContents, path)
			} else {
				path := filepath.Join(filepath.Dir(filePath), line)
				ignoreDirectories = append(ignoreDirectories, path)
			}
		}
	}
	return ignoreDirectories, ignoreContents
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
		if len(parts) == 1 {
			prefix = ""
		} else if len(parts) > 1 {
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
