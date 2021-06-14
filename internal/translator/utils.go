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

package translator

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/filesystem"
	"github.com/konveyor/move2kube/internal/common"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/sirupsen/logrus"
)

func getTranslatorConfig(path string) (translatortypes.Translator, error) {
	tc := translatortypes.Translator{
		Spec: translatortypes.TranslatorSpec{
			FilePath: path,
		},
	}
	if err := common.ReadMove2KubeYaml(path, &tc); err != nil {
		logrus.Debugf("Failed to read the translator metadata at path %q Error: %q", path, err)
		return tc, err
	}
	if tc.Kind != translatortypes.TranslatorKind {
		err := fmt.Errorf("the file at path %q is not a valid cluster metadata. Expected kind: %s Actual kind: %s", path, translatortypes.TranslatorKind, tc.Kind)
		logrus.Debug(err)
		return tc, err
	}
	return tc, nil
}

func walkForServices(inputPath string, ts map[string]Translator, bservices map[string]plantypes.Service) (services map[string]plantypes.Service, unservices []plantypes.Translator, err error) {
	services = bservices
	unservices = make([]plantypes.Translator, 0)
	ignoreDirectories, ignoreContents := getIgnorePaths(inputPath)
	knownProjectPaths := make([]string, 0)

	err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("Skipping path %q due to error. Error: %q", path, err)
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
		logrus.Debugf("Planning dir translation - %s", path)
		found := false
		for _, t := range translators {
			logrus.Debugf("[%T] Planning translation", t, path)
			_, env := t.GetConfig()
			env.Sync()
			nservices, nunservices, err := t.DirectoryDetect(env.EncodePath(path))
			if err != nil {
				logrus.Warnf("[%T] Failed : %s", t, err)
			} else {
				nservices = postProcessServices(nservices, t)
				unservices = postProcessTranslators(unservices, t)
				plantypes.MergeServices(services, nservices)
				unservices = append(unservices, nunservices...)
				if len(nservices) > 0 || len(unservices) > 0 {
					found = true
				}
				logrus.Debugf("[%T] Done", t)
			}
		}
		logrus.Debugf("Dir translation done - %s", path)
		if !found {
			logrus.Debugf("No service found in directory %q", path)
			if common.IsStringPresent(ignoreContents, path) {
				return filepath.SkipDir
			}
			return nil
		}
		return filepath.SkipDir // Skip all subdirectories when base directory is a valid package
	})
	if err != nil {
		logrus.Errorf("Error occurred while walking through the directory at path %q Error: %q", inputPath, err)
	}
	return
}

type project struct {
	path       string
	pathsuffix string
}

func nameServices(projName string, nServices map[string]plantypes.Service, sts []plantypes.Translator) (services map[string]plantypes.Service) {
	services = nServices
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
				logrus.Errorf("No paths in the translator. Ignoring translator : %+v", st)
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
		logrus.Warnf("Unable to fetch .m2kignore files at path %q Error: %q", inputPath, err)
		return ignoreDirectories, ignoreContents
	}
	for _, filePath := range filePaths {
		file, err := os.Open(filePath)
		if err != nil {
			logrus.Warnf("Failed to open the .m2kignore file at path %q Error: %q", filePath, err)
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

func convertPatchToPathMappings(p translatortypes.Patch, t Translator) []translatortypes.PathMapping {
	pathMappings := []translatortypes.PathMapping{}
	npm, err := processPathMappings(p.PathMappings, t, p.Config)
	if err != nil {
		logrus.Errorf("Unable to process path mappings : %s", err)
		return npm
	}
	return pathMappings
}

func processPathMappings(pms []translatortypes.PathMapping, t Translator, templateConfig interface{}) ([]translatortypes.PathMapping, error) {
	pathMappings := []translatortypes.PathMapping{}
	for _, pm := range pms {
		npm, err := processPathMapping(pm, t, templateConfig)
		if err != nil {
			logrus.Errorf("Unable to process path mapping : %s", err)
			continue
		}
		pathMappings = append(pathMappings, npm)
	}
	return pathMappings, nil
}

func processPathMapping(pm translatortypes.PathMapping, t Translator, templateConfig interface{}) (translatortypes.PathMapping, error) {
	config, env := t.GetConfig()
	switch pm.Type {
	case translatortypes.SourcePathMappingType:
		pm.SrcPath = env.DecodePath(pm.SrcPath)
		return pm, nil
	case translatortypes.ModifiedSourcePathMappingType:
		orgSrcPath := env.GetSourcePath()
		srcPath := orgSrcPath
		if filepath.IsAbs(pm.SrcPath) {
			var err error
			srcPath, err = filepath.Rel(orgSrcPath, pm.SrcPath)
			if err != nil {
				logrus.Errorf("Invalid sourcePath %s in translator %+v. Ignoring.", pm.SrcPath, pm)
				return pm, err
			}
		}
		pm.SrcPath = env.SyncOutput(pm.SrcPath)
		newTempDir, err := ioutil.TempDir(common.TempPath, "modifiedsource-*")
		if err != nil {
			logrus.Errorf("Unable to create temporary directory for templates in pathMapping %+v. Ingoring.", pm)
			return pm, err
		}
		if srcPath != "" {
			err := ioutil.WriteFile(filepath.Join(newTempDir, "sourcerelativepath.config"), []byte(srcPath), 0666)
			if err != nil {
				logrus.Errorf("Unable to persist source relative path to file")
			}
		}
		if err := filesystem.GenerateDelta(filepath.Join(orgSrcPath, srcPath), pm.SrcPath, newTempDir); err == nil {
			pm.SrcPath = newTempDir
		} else {
			logrus.Errorf("Error while copying modified sourcepath for %+v. Ignoring.", pm)
			return pm, err
		}
		pm.SrcPath = newTempDir
		return pm, nil
	case translatortypes.TemplatePathMappingType:
		if !filepath.IsAbs(pm.SrcPath) {
			pm.SrcPath = filepath.Join(config.Spec.FilePath, pm.SrcPath)
		}
		pm.SrcPath = env.SyncOutput(pm.SrcPath)
		newTempDir, err := ioutil.TempDir(common.TempPath, "templates-*")
		if err != nil {
			logrus.Errorf("Unable to create temporary directory for templates in pathMapping %+v. Ingoring.", pm)
			return pm, err
		}
		if err := filesystem.TemplateCopy(pm.SrcPath, newTempDir, templateConfig); err != nil {
			logrus.Errorf("Error while copying sourcepath for %+v. Ignoring.", pm)
			return pm, err
		}
		pm.Type = translatortypes.DefaultPathMappingType
		pm.SrcPath = newTempDir
		return pm, nil
	default:
		if !filepath.IsAbs(pm.SrcPath) {
			pm.SrcPath = filepath.Join(config.Spec.FilePath, pm.SrcPath)
		} else {
			pm.SrcPath = env.SyncOutput(pm.SrcPath)
		}
		return pm, nil
	}
}

func createOutput(pathMappings []translatortypes.PathMapping, sourcePath, outputPath string) error {
	for _, pm := range pathMappings {
		switch pm.Type {
		case translatortypes.SourcePathMappingType:
			if err := filesystem.Replicate(filepath.Join(sourcePath, pm.SrcPath), filepath.Join(outputPath, pm.DestPath)); err != nil {
				logrus.Errorf("Error while copying sourcepath for %+v", pm)
			}
		case translatortypes.ModifiedSourcePathMappingType:
			logrus.Errorf("Modified source path mapping type copy yet to be implemented")
			//TODO: Implement
			continue
		default:
			srcPath := pm.SrcPath
			if !filepath.IsAbs(pm.SrcPath) {
				srcPath = filepath.Join(pm.SrcPath)
			}
			if err := filesystem.Replicate(srcPath, filepath.Join(outputPath, pm.DestPath)); err != nil {
				logrus.Errorf("Error while copying sourcepath for %+v", pm)
			}
		}
	}
	return nil
}

func postProcessServices(services map[string]plantypes.Service, t Translator) map[string]plantypes.Service {
	for sn, s := range services {
		services[sn] = postProcessTranslators(s, t)
	}
	return services
}

func postProcessTranslators(translators []plantypes.Translator, t Translator) []plantypes.Translator {
	for sti, st := range translators {
		config, env := t.GetConfig()
		srcPath := env.GetSourcePath()
		st.Name = config.Name
		err := plantypes.ChangePaths(&st, srcPath, env.DecodePath(srcPath))
		if err != nil {
			logrus.Errorf("Unable to encode of translator obj %+v. Ignoring : %s", t, err)
			continue
		}
		translators[sti] = st
	}
	return translators
}

func preProcessTranslators(translators []plantypes.Translator, t Translator) []plantypes.Translator {
	for sti, st := range translators {
		_, env := t.GetConfig()
		srcPath := env.GetSourcePath()
		err := plantypes.ChangePaths(&st, env.DecodePath(srcPath), srcPath)
		if err != nil {
			logrus.Errorf("Unable to decode of translator obj %+v. Ignoring : %s", t, err)
			continue
		}
		translators[sti] = st
	}
	return translators
}

func preProcessTranslator(p plantypes.Translator, t Translator) plantypes.Translator {
	_, env := t.GetConfig()
	srcPath := env.GetSourcePath()
	err := plantypes.ChangePaths(&p, env.DecodePath(srcPath), srcPath)
	if err != nil {
		logrus.Errorf("Unable to decode of translator obj %+v. Ignoring : %s", t, err)
	}
	return p
}

func preProcessPlanObj(p interface{}, t Translator) interface{} {
	_, env := t.GetConfig()
	srcPath := env.GetSourcePath()
	err := plantypes.ChangePaths(&p, env.DecodePath(srcPath), srcPath)
	if err != nil {
		logrus.Errorf("Unable to decode of plan obj %+v. Ignoring : %s", t, err)
	}
	return p
}

func postProcessPlanObj(p interface{}, t Translator) interface{} {
	_, env := t.GetConfig()
	srcPath := env.GetSourcePath()
	err := plantypes.ChangePaths(&p, srcPath, env.DecodePath(srcPath))
	if err != nil {
		logrus.Errorf("Unable to decode of plan obj %+v. Ignoring : %s", t, err)
	}
	return p
}
