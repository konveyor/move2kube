/*
 *  Copyright IBM Corporation 2020, 2021
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
	"os"
	"path/filepath"
	"reflect"
	"regexp"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/transformer/classes/analysers"
	"github.com/konveyor/move2kube/internal/transformer/classes/external"
	"github.com/konveyor/move2kube/internal/transformer/classes/generators"
	"github.com/konveyor/move2kube/qaengine"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

var (
	initialized                                     = false
	transformerTypes        map[string]reflect.Type = map[string]reflect.Type{}
	transformers            map[string]Transformer  = map[string]Transformer{}
	defaultIgnoreDirRegexps                         = []*regexp.Regexp{regexp.MustCompile("[.].*")}
)

// Transformer interface defines transformer that transforms files and converts it to ir representation
type Transformer interface {
	Init(tc transformertypes.Transformer, env *environment.Environment) (err error)
	// GetConfig returns the transformer config
	GetConfig() (transformertypes.Transformer, *environment.Environment)

	BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error)
	DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error)

	Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error)
}

func init() {
	transformerObjs := []Transformer{
		new(analysers.ComposeAnalyser),
		new(analysers.CNBContainerizer),
		new(analysers.CloudFoundry),
		new(analysers.DockerfileDetector),
		new(analysers.SpringbootAnalyser),
		new(analysers.ZuulAnalyser),
		new(analysers.EurekaReplaceEngine),

		new(generators.ComposeGenerator),
		new(generators.Kubernetes),
		new(generators.Knative),
		new(generators.Tekton),
		new(generators.BuildConfig),
		new(generators.CNBGenerator),
		new(generators.S2IGenerator),
		new(generators.DockerfileImageBuildScript),
		new(generators.ContainerImagesPushScript),
		new(generators.ContainerImagesBuildScript),
		new(generators.ReadMeGenerator),

		new(external.Starlark),
		new(external.SimpleExecutable),
	}
	for _, tt := range transformerObjs {
		t := reflect.TypeOf(tt).Elem()
		tn := t.Name()
		if ot, ok := transformerTypes[tn]; ok {
			logrus.Errorf("Two transformer classes have the same name %s : %T, %T; Ignoring %T", tn, ot, t, t)
			continue
		}
		transformerTypes[tn] = t
	}
}

// Init initializes the transformers
func Init(assetsPath, sourcePath, outputPath, projName string) (err error) {
	filePaths, err := common.GetFilesByExt(assetsPath, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Unable to fetch yaml files and recognize cf manifest yamls at path %q Error: %q", assetsPath, err)
		return err
	}
	transformerFiles := map[string]string{}
	for _, filePath := range filePaths {
		tc, err := getTransformerConfig(filePath)
		if err != nil {
			logrus.Debugf("Unable to load %s as Transformer config : %s", filePath, err)
			continue
		}
		if otc, ok := transformerFiles[tc.Name]; ok {
			logrus.Warnf("Duplicate transformer configs with same name %s found. Ignoring %s in favor of %s", otc, filePath)
		}
		transformerFiles[tc.Name] = filePath
	}
	InitTransformers(transformerFiles, sourcePath, outputPath, projName, false)
	return nil
}

// InitTransformers initializes a subset of transformers
func InitTransformers(transformerToInit map[string]string, sourcePath string, outputPath, projName string, warn bool) error {
	if initialized {
		return nil
	}
	transformerConfigs := map[string]transformertypes.Transformer{}
	for tn, tfilepath := range transformerToInit {
		tc, err := getTransformerConfig(tfilepath)
		if err != nil {
			if warn {
				logrus.Errorf("Unable to load %s as Transformer config : %s", tfilepath, err)
			} else {
				logrus.Debugf("Unable to load %s as Transformer config : %s", tfilepath, err)
			}
			continue
		}
		if ot, ok := transformerConfigs[tc.Name]; ok {
			logrus.Errorf("Found two conflicting transformer Names %s : %s, %s. Ignoring %s.", tc.Name, ot.Spec.FilePath, tc.Spec.FilePath, ot.Spec.FilePath)
		}
		if _, ok := transformerTypes[tc.Spec.Class]; ok {
			transformerConfigs[tc.Name] = tc
			continue
		}
		transformerConfigs[tn] = tc
	}
	tns := []string{}
	for tn := range transformerConfigs {
		tns = append(tns, tn)
	}
	transformerNames := qaengine.FetchMultiSelectAnswer(common.ConfigTransformerTypesKey, "Select all transformer types that you are interested in:", []string{"Services that don't support any of the transformer types you are interested in will be ignored."}, tns, tns)
	for _, tn := range transformerNames {
		tc := transformerConfigs[tn]
		if c, ok := transformerTypes[tc.Spec.Class]; !ok {
			logrus.Errorf("Unable to find Transformer class %s in %+v", tc.Spec.Class, transformerTypes)
		} else {
			t := reflect.New(c).Interface().(Transformer)
			env, err := environment.NewEnvironment(tc.Name, projName, sourcePath, outputPath, filepath.Dir(tc.Spec.FilePath), tc.Spec.TemplatesDir, nil, environmenttypes.Container{})
			if err != nil {
				logrus.Errorf("Unable to create environment : %s", err)
				return err
			}
			if err := t.Init(tc, env); err != nil {
				if _, ok := err.(*transformertypes.TransformerDisabledError); ok {
					logrus.Debugf("Unable to initialize transformer %s : %s", tc.Name, err)
				} else {
					logrus.Errorf("Unable to initialize transformer %s : %s", tc.Name, err)
				}
			} else {
				transformers[tn] = t
			}
		}
	}
	initialized = true
	return nil
}

// Destroy destroys the transformers
func Destroy() {
	for _, t := range transformers {
		_, env := t.GetConfig()
		if err := env.Destroy(); err != nil {
			logrus.Errorf("Unable to destroy environment : %s", err)
		}
	}
}

// GetTransformers returns the list of initialized transformers
func GetTransformers() map[string]Transformer {
	return transformers
}

// GetServices returns the list of services detected in a directory
func GetServices(prjName string, dir string) (services map[string]plantypes.Service, err error) {
	services = map[string]plantypes.Service{}
	unservices := []plantypes.Transformer{}
	logrus.Infoln("Planning Transformation - Base Directory")
	logrus.Debugf("Transformers : %+v", transformers)
	for tn, t := range transformers {
		config, env := t.GetConfig()
		env.Reset()
		logrus.Infof("[%s] Planning transformation", tn)
		nservices, nunservices, err := t.BaseDirectoryDetect(env.Encode(dir).(string))
		if err != nil {
			logrus.Errorf("[%s] Failed : %s", tn, err)
		} else {
			nservices = setTransformerInfoForServices(*env.Decode(&nservices).(*map[string]plantypes.Service), config)
			nunservices = setTransformerInfoForTransformers(*env.Decode(&nunservices).(*[]plantypes.Transformer), config)
			services = plantypes.MergeServices(services, nservices)
			unservices = append(unservices, nunservices...)
			if len(nservices) > 0 || len(nunservices) > 0 {
				logrus.Infof("Identified %d namedservices and %d unnamedservices", len(nservices), len(nunservices))
			}
			logrus.Infof("[%s] Done", tn)
		}
	}
	logrus.Infof("[Base Directory] Identified %d namedservices and %d unnamedservices", len(services), len(unservices))
	logrus.Infoln("Transformation planning - Base Directory done")
	logrus.Infoln("Planning Transformation - Directory Walk")
	nservices, nunservices, err := walkForServices(dir, transformers, services)
	if err != nil {
		logrus.Errorf("Transformation planning - Directory Walk failed : %s", err)
	} else {
		services = nservices
		unservices = append(unservices, nunservices...)
		logrus.Infoln("Transformation planning - Directory Walk done")
	}
	logrus.Infof("[Directory Walk] Identified %d namedservices and %d unnamedservices", len(services), len(unservices))
	services = nameServices(prjName, services, unservices)
	logrus.Infof("[Named Services] Identified %d namedservices", len(services))
	return
}

// Transform transforms as per the plan
func Transform(plan plantypes.Plan, outputPath string) (err error) {
	artifacts := []transformertypes.Artifact{}
	pathMappings := []transformertypes.PathMapping{}
	iteration := 1
	logrus.Infof("Iteration %d", iteration)
	for serviceName, service := range plan.Spec.Services {
		for _, transformer := range service {
			logrus.Infof("Transformer %s for service %s", transformer.Name, serviceName)
			t := transformers[transformer.Name]
			_, env := t.GetConfig()
			env.Reset()
			a := getArtifactForTransformerPlan(serviceName, transformer, plan)
			newPathMappings, newArtifacts, err := t.Transform([]transformertypes.Artifact{*env.Encode(&a).(*transformertypes.Artifact)}, *env.Encode(&artifacts).(*[]transformertypes.Artifact))
			if err != nil {
				logrus.Errorf("Unable to transform service %s using %s : %s", serviceName, transformer.Name, err)
				continue
			}
			newPathMappings = env.ProcessPathMappings(newPathMappings)
			newPathMappings = *env.DownloadAndDecode(&newPathMappings, true).(*[]transformertypes.PathMapping)
			newArtifacts = *env.DownloadAndDecode(&newArtifacts, false).(*[]transformertypes.Artifact)
			pathMappings = append(pathMappings, newPathMappings...)
			artifacts = mergeArtifacts(append(artifacts, newArtifacts...))
			logrus.Infof("Created %d pathMappings and %d artifacts. Total Path Mappings : %d. Total Artifacts : %d.", len(newPathMappings), len(newArtifacts), len(pathMappings), len(artifacts))
			logrus.Infof("Transformer %s Done for service %s", transformer.Name, serviceName)
		}
	}
	err = processPathMappings(pathMappings, plan.Spec.RootDir, outputPath)
	if err != nil {
		logrus.Errorf("Unable to process path mappings")
	}
	newArtifactsToProcess := artifacts
	for {
		iteration++
		newArtifactsCreated := []transformertypes.Artifact{}
		logrus.Infof("Iteration %d", iteration)
		for tn, t := range transformers {
			config, env := t.GetConfig()
			env.Reset()
			artifactsToProcess := []transformertypes.Artifact{}
			for _, na := range newArtifactsToProcess {
				if common.IsStringPresent(config.Spec.ArtifactsToProcess, string(na.Artifact)) {
					artifactsToProcess = append(artifactsToProcess, na)
				}
			}
			if len(artifactsToProcess) == 0 {
				continue
			}
			logrus.Infof("Transformer %s processing %d artifacts", config.Name, len(artifactsToProcess))
			newPathMappings, newArtifacts, err := t.Transform(*env.Encode(&artifactsToProcess).(*[]transformertypes.Artifact), *env.Encode(&artifacts).(*[]transformertypes.Artifact))
			if err != nil {
				logrus.Errorf("Unable to transform artifacts using %s : %s", tn, err)
				continue
			}
			newPathMappings = env.ProcessPathMappings(newPathMappings)
			newPathMappings = *env.DownloadAndDecode(&newPathMappings, true).(*[]transformertypes.PathMapping)
			newArtifacts = *env.DownloadAndDecode(&newArtifacts, false).(*[]transformertypes.Artifact)
			pathMappings = append(pathMappings, newPathMappings...)
			newArtifactsCreated = append(newArtifactsCreated, newArtifacts...)
			logrus.Infof("Created %d pathMappings and %d artifacts. Total Path Mappings : %d. Total Artifacts : %d.", len(newPathMappings), len(newArtifacts), len(pathMappings), len(artifacts))
			logrus.Infof("Transformer %s Done", config.Name)
		}
		if err = os.RemoveAll(outputPath); err != nil {
			logrus.Errorf("Unable to delete %s : %s", outputPath, err)
		}
		err = processPathMappings(pathMappings, plan.Spec.RootDir, outputPath)
		if err != nil {
			logrus.Errorf("Unable to process path mappings")
		}
		if len(newArtifactsCreated) == 0 {
			break
		}
		newArtifactsToProcess = mergeArtifacts(append(newArtifactsCreated, updatedArtifacts(artifacts, newArtifactsCreated)...))
		artifacts = mergeArtifacts(append(artifacts, newArtifactsToProcess...))
	}
	return nil
}

func walkForServices(inputPath string, ts map[string]Transformer, bservices map[string]plantypes.Service) (services map[string]plantypes.Service, unservices []plantypes.Transformer, err error) {
	services = bservices
	unservices = []plantypes.Transformer{}
	ignoreDirectories, ignoreContents := getIgnorePaths(inputPath)
	knownProjectPaths := []string{}

	err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("Skipping path %q due to error. Error: %q", path, err)
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		for _, dirRegExp := range defaultIgnoreDirRegexps {
			if dirRegExp.Match([]byte(filepath.Base(path))) {
				return filepath.SkipDir
			}
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
		logrus.Debugf("Planning dir transformation - %s", path)
		found := false
		for _, t := range transformers {
			config, env := t.GetConfig()
			logrus.Debugf("[%s] Planning transformation in %s", config.Name, path)
			env.Reset()
			nservices, nunservices, err := t.DirectoryDetect(env.Encode(path).(string))
			if err != nil {
				logrus.Warnf("[%s] Failed : %s", config.Name, err)
			} else {
				nservices = setTransformerInfoForServices(*env.Decode(&nservices).(*map[string]plantypes.Service), config)
				nunservices = setTransformerInfoForTransformers(*env.Decode(&nunservices).(*[]plantypes.Transformer), config)
				services = plantypes.MergeServices(services, nservices)
				unservices = append(unservices, nunservices...)
				logrus.Debugf("[%s] Done", config.Name)
				if len(nservices) > 0 || len(nunservices) > 0 {
					found = true
					relpath, _ := filepath.Rel(inputPath, path)
					logrus.Infof("Found %d named services and %d unnamed transformer success in %s", len(nservices), len(nunservices), relpath)
				}
			}
		}
		logrus.Debugf("Dir transformation done - %s", path)
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
