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

package transformer

import (
	"os"
	"path/filepath"
	"reflect"

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
	transformerTypes map[string]reflect.Type = make(map[string]reflect.Type)
	transformers     map[string]Transformer  = make(map[string]Transformer)
)

// Transformer interface defines transformer that transforms files and converts it to ir representation
type Transformer interface {
	Init(tc transformertypes.Transformer, env environment.Environment) (err error)
	GetConfig() (transformertypes.Transformer, environment.Environment)

	BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error)
	DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error)

	Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error)
}

func init() {
	transformerObjs := []Transformer{new(analysers.ComposeAnalyser), new(generators.ComposeGenerator), new(generators.Kubernetes), new(generators.Knative), new(generators.Tekton), new(generators.BuildConfig), new(external.SimpleExecutable), new(analysers.CNBContainerizer), new(generators.CNBGenerator), new(analysers.CloudFoundry)}
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

func Init(assetsPath, sourcePath string) (err error) {
	filePaths, err := common.GetFilesByExt(assetsPath, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Unable to fetch yaml files and recognize cf manifest yamls at path %q Error: %q", assetsPath, err)
		return err
	}
	transformerFiles := make(map[string]string)
	for _, filePath := range filePaths {
		tc, err := getTransformerConfig(filePath)
		if err != nil {
			logrus.Debugf("Unable to load %s as Transformer config", filePath, err)
			continue
		}
		transformerFiles[tc.Name] = filePath
	}
	InitTransformers(transformerFiles, sourcePath, false)
	return nil
}

func InitTransformers(transformerToInit map[string]string, sourcePath string, warn bool) error {
	transformerConfigs := make(map[string]transformertypes.Transformer)
	for tn, tfilepath := range transformerToInit {
		tc, err := getTransformerConfig(tfilepath)
		if err != nil {
			if warn {
				logrus.Errorf("Unable to load %s as Transformer config", tfilepath, err)
			} else {
				logrus.Debugf("Unable to load %s as Transformer config", tfilepath, err)
			}
			continue
		}
		if ot, ok := transformerConfigs[tc.Name]; ok {
			logrus.Errorf("Found two conflicting transformer Names %s : %s, %s. Ignoring %s.", tc.Name, ot.Spec.FilePath, tc.Spec.FilePath)
			continue
		}
		if _, ok := transformerTypes[tc.Spec.Class]; ok {
			transformerConfigs[tc.Name] = tc
			continue
		}
		transformerConfigs[tn] = tc
	}
	tns := make([]string, 0)
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
			env, err := environment.NewEnvironment(tc.Name, sourcePath, filepath.Dir(tc.Spec.FilePath), environmenttypes.Container{})
			if err != nil {
				logrus.Errorf("Unable to create environment : %s", err)
				return err
			}
			if err := t.Init(tc, env); err != nil {
				logrus.Errorf("Unable to initialize transformer %s : %s", tc.Name, err)
			} else {
				transformers[tn] = t
			}
		}
	}
	return nil
}

func Destroy() {
	for _, t := range transformers {
		_, env := t.GetConfig()
		if err := env.Destroy(); err != nil {
			logrus.Errorf("Unable to destroy environment : %s", err)
		}
	}
}

func GetTransformers() map[string]Transformer {
	return transformers
}

func GetServices(prjName string, dir string) (services map[string]plantypes.Service, err error) {
	services = make(map[string]plantypes.Service)
	unservices := make([]plantypes.Transformer, 0)
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
			unservices = setTransformerInfoForTransformers(*env.Decode(&unservices).(*[]plantypes.Transformer), config)
			services = plantypes.MergeServices(services, nservices)
			unservices = append(unservices, nunservices...)
			logrus.Infof("Identified %d namedservices and %d unnamedservices", len(nservices), len(nunservices))
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
		iteration += 1
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
			logrus.Infof("Transformer %s", config.Name)
			newPathMappings, newArtifacts, err := t.Transform(*env.Encode(&artifactsToProcess).(*[]transformertypes.Artifact), *env.Encode(&artifacts).(*[]transformertypes.Artifact))
			if err != nil {
				logrus.Errorf("Unable to transform artifacts using %s : %s", tn, err)
				continue
			}
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
	unservices = make([]plantypes.Transformer, 0)
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
