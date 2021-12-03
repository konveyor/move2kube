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

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/filesystem"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer/compose"
	"github.com/konveyor/move2kube/transformer/containerimage"
	"github.com/konveyor/move2kube/transformer/dockerfile"
	"github.com/konveyor/move2kube/transformer/dockerfilegenerator"
	"github.com/konveyor/move2kube/transformer/dockerfilegenerator/java"
	"github.com/konveyor/move2kube/transformer/dockerfilegenerator/windows"
	"github.com/konveyor/move2kube/transformer/external"
	"github.com/konveyor/move2kube/transformer/kubernetes"
	collectiontypes "github.com/konveyor/move2kube/types/collection"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	initialized      = false
	transformerTypes = map[string]reflect.Type{}
	transformers     = map[string]Transformer{}
)

// Transformer interface defines transformer that transforms files and converts it to ir representation
type Transformer interface {
	Init(tc transformertypes.Transformer, env *environment.Environment) (err error)
	// GetConfig returns the transformer config
	GetConfig() (transformertypes.Transformer, *environment.Environment)
	DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error)
	Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error)
}

func init() {
	transformerObjs := []Transformer{
		new(external.Starlark),
		new(external.Executable),

		new(Router),

		new(dockerfile.DockerfileDetector),
		new(dockerfile.DockerfileParser),
		new(dockerfile.DockerfileImageBuildScript),
		new(dockerfilegenerator.NodejsDockerfileGenerator),
		new(dockerfilegenerator.GolangDockerfileGenerator),
		new(dockerfilegenerator.PHPDockerfileGenerator),
		new(dockerfilegenerator.PythonDockerfileGenerator),
		new(dockerfilegenerator.RubyDockerfileGenerator),
		new(dockerfilegenerator.DotNet5DockerfileGenerator),
		new(java.JarAnalyser),
		new(java.WarAnalyser),
		new(java.Tomcat),
		new(java.MavenAnalyser),
		new(java.ZuulAnalyser),
		new(java.EurekaReplaceEngine),
		new(windows.WinConsoleAppDockerfileGenerator),
		new(windows.WinSilverLightWebAppDockerfileGenerator),
		new(windows.WinWebAppDockerfileGenerator),

		new(compose.ComposeAnalyser),
		new(compose.ComposeGenerator),

		new(CloudFoundry),

		new(containerimage.ContainerImagesPushScript),
		new(containerimage.ContainerImagesBuildScript),

		new(kubernetes.Kubernetes),
		new(kubernetes.Knative),
		new(kubernetes.Tekton),
		new(kubernetes.BuildConfig),
		new(kubernetes.Parameterizer),

		new(ReadMeGenerator),
	}
	transformerTypes = common.GetTypesMap(transformerObjs)
}

// Init initializes the transformers
func Init(assetsPath, sourcePath string, selector labels.Selector, targetCluster collectiontypes.ClusterMetadata, outputPath, projName string) (err error) {
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
			logrus.Warnf("Duplicate transformer configs with same name %s found. Ignoring %s in favor of %s", tc.Name, otc, filePath)
		}
		transformerFiles[tc.Name] = filePath
	}
	InitTransformers(transformerFiles, selector, targetCluster, sourcePath, outputPath, projName, false)
	return nil
}

// InitTransformers initializes a subset of transformers
func InitTransformers(transformerToInit map[string]string, selector labels.Selector, targetCluster collectiontypes.ClusterMetadata, sourcePath string, outputPath, projName string, logError bool) error {
	if initialized {
		return nil
	}
	transformerFilterString := qaengine.FetchStringAnswer(common.TransformerSelectorKey, "", []string{"Set the transformer selector config."}, "")
	if transformerFilterString != "" {
		transformerFilter, err := common.ConvertStringSelectorsToSelectors(transformerFilterString)
		if err != nil {
			logrus.Errorf("Unable to parse transformer filter string : %s", err)
		} else {
			reqs, _ := transformerFilter.Requirements()
			selector = selector.Add(reqs...)
		}
	}
	transformerConfigs := getFilteredTransformers(transformerToInit, selector, logError)
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
			transformerContextPath := filepath.Dir(tc.Spec.FilePath)
			envInfo := environment.EnvInfo{
				Name:            tc.Name,
				ProjectName:     projName,
				TargetCluster:   targetCluster,
				Source:          sourcePath,
				Output:          outputPath,
				Context:         transformerContextPath,
				RelTemplatesDir: tc.Spec.TemplatesDir,
			}
			for src, dest := range tc.Spec.ExternalFiles {
				err := filesystem.Replicate(filepath.Join(transformerContextPath, src), filepath.Join(transformerContextPath, dest))
				if err != nil {
					logrus.Errorf("Error while copying external files in transformer %s (%s:%s) : %s", tc.Name, src, dest, err)
				}
			}
			env, err := environment.NewEnvironment(envInfo, nil, environmenttypes.Container{})
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

// GetInitializedTransformers returns the list of initialized transformers
func GetInitializedTransformers() map[string]Transformer {
	return transformers
}

// GetInitializedTransformersF returns the list of initialized transformers after filtering
func GetInitializedTransformersF(filters labels.Selector) map[string]Transformer {
	filteredTransformers := map[string]Transformer{}
	for tn, t := range GetInitializedTransformers() {
		tc, _ := t.GetConfig()
		if tc.ObjectMeta.Labels == nil {
			tc.ObjectMeta.Labels = map[string]string{}
		}
		if !filters.Matches(labels.Set(tc.ObjectMeta.Labels)) {
			continue
		}
		filteredTransformers[tn] = t
	}
	return filteredTransformers
}

// GetServices returns the list of services detected in a directory
func GetServices(prjName string, dir string) (services map[string][]transformertypes.Artifact, err error) {
	services = map[string][]transformertypes.Artifact{}
	logrus.Infoln("Planning Transformation - Base Directory")
	logrus.Debugf("Transformers : %+v", transformers)
	for tn, t := range transformers {
		config, env := t.GetConfig()
		env.Reset()
		if config.Spec.DirectoryDetect.Levels != 1 {
			continue
		}
		logrus.Infof("[%s] Planning transformation", tn)
		nservices, err := t.DirectoryDetect(env.Encode(dir).(string))
		if err != nil {
			logrus.Errorf("[%s] Failed : %s", tn, err)
		} else {
			nservices = setTransformerInfoForServices(*env.Decode(&nservices).(*map[string][]transformertypes.Artifact), config)
			services = plantypes.MergeServices(services, nservices)
			if len(nservices) > 0 {
				logrus.Infof(getNamedAndUnNamedServicesLogMessage(nservices))
			}
			common.PlanProgressNumBaseDetectTransformers++
			logrus.Infof("[%s] Done", tn)
		}
	}
	logrus.Infof("[Base Directory] %s", getNamedAndUnNamedServicesLogMessage(services))
	logrus.Infoln("Transformation planning - Base Directory done")
	logrus.Infoln("Planning Transformation - Directory Walk")
	nservices, err := walkForServices(dir, transformers, services)
	if err != nil {
		logrus.Errorf("Transformation planning - Directory Walk failed : %s", err)
	} else {
		services = nservices
		logrus.Infoln("Transformation planning - Directory Walk done")
	}
	logrus.Infof("[Directory Walk] %s", getNamedAndUnNamedServicesLogMessage(services))
	services = nameServices(prjName, services)
	logrus.Infof("[Named Services] Identified %d namedservices", len(services))
	return
}

// Transform transforms as per the plan
func Transform(plan plantypes.Plan, outputPath string) (err error) {
	allartifacts := []transformertypes.Artifact{}
	newArtifactsToProcess := []transformertypes.Artifact{}
	pathMappings := []transformertypes.PathMapping{}
	iteration := 1
	logrus.Infof("Iteration %d", iteration)
	for sn, sas := range plan.Spec.Services {
		if len(sas) == 0 {
			logrus.Errorf("Ignoring service %s, since it has no artifacts", sn)
			continue
		}
		a := sas[0]
		a.Name = sn
		serviceConfig := artifacts.ServiceConfig{
			ServiceName: sn,
		}
		if a.Configs == nil {
			a.Configs = map[string]interface{}{}
		}
		a.Configs[artifacts.ServiceConfigType] = serviceConfig
		newArtifactsToProcess = append(newArtifactsToProcess, a)
	}
	allartifacts = newArtifactsToProcess
	for {
		iteration++
		newArtifactsCreated := []transformertypes.Artifact{}
		logrus.Infof("Iteration %d", iteration)
		for tn, t := range transformers {
			logrus.Debugf("Starting processing for Transformer %s", tn)
			config, env := t.GetConfig()
			env.Reset()
			artifactsToProcess := []transformertypes.Artifact{}
			for _, na := range newArtifactsToProcess {
				if preprocessSpec, ok := config.Spec.ArtifactsToProcess[na.Artifact]; ok || na.Artifact == tn {
					if preprocessSpec.Merge {
						artifactsToProcess = mergeArtifacts(append(artifactsToProcess, updatedArtifacts(allartifacts, na)...))
					}
					artifactsToProcess = append(artifactsToProcess, na)
				}
			}
			logrus.Debugf("Transformer %s will be processing %d artifacts", config.Name, len(artifactsToProcess))
			if len(artifactsToProcess) == 0 {
				continue
			}
			logrus.Infof("Transformer %s processing %d artifacts", config.Name, len(artifactsToProcess))
			newPathMappings, newArtifacts, err := t.Transform(*env.Encode(&artifactsToProcess).(*[]transformertypes.Artifact), *env.Encode(&allartifacts).(*[]transformertypes.Artifact))
			if err != nil {
				logrus.Errorf("Unable to transform artifacts using %s : %s", tn, err)
				continue
			}
			newPathMappings = env.ProcessPathMappings(newPathMappings)
			newPathMappings = *env.DownloadAndDecode(&newPathMappings, true).(*[]transformertypes.PathMapping)
			err = processPathMappings(newPathMappings, plan.Spec.RootDir, outputPath)
			if err != nil {
				logrus.Errorf("Unable to process path mappings")
			}
			newArtifacts = *env.DownloadAndDecode(&newArtifacts, false).(*[]transformertypes.Artifact)
			pathMappings = append(pathMappings, newPathMappings...)
			newArtifactsCreated = append(newArtifactsCreated, newArtifacts...)
			logrus.Infof("Created %d pathMappings and %d artifacts. Total Path Mappings : %d. Total Artifacts : %d.", len(newPathMappings), len(newArtifacts), len(pathMappings), len(allartifacts))
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
		newArtifactsToProcess = newArtifactsCreated
		allartifacts = append(allartifacts, newArtifactsToProcess...)
	}
	return nil
}

func walkForServices(inputPath string, ts map[string]Transformer, bservices map[string][]transformertypes.Artifact) (services map[string][]transformertypes.Artifact, err error) {
	services = bservices
	ignoreDirectories, ignoreContents := getIgnorePaths(inputPath)
	knownProjectPaths := []string{}

	err = filepath.WalkDir(inputPath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			logrus.Warnf("Skipping path %q due to error. Error: %q", path, err)
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		for _, dirRegExp := range common.DefaultIgnoreDirRegexps {
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
		common.PlanProgressNumDirectories++
		logrus.Debugf("Planning dir transformation - %s", path)
		found := false
		for _, t := range transformers {
			config, env := t.GetConfig()
			logrus.Debugf("[%s] Planning transformation in %s", config.Name, path)
			env.Reset()
			if config.Spec.DirectoryDetect.Levels == 1 || config.Spec.DirectoryDetect.Levels == 0 {
				continue
			}
			nservices, err := t.DirectoryDetect(env.Encode(path).(string))
			if err != nil {
				logrus.Warnf("[%s] Failed : %s", config.Name, err)
			} else {
				nservices = setTransformerInfoForServices(*env.Decode(&nservices).(*map[string][]transformertypes.Artifact), config)
				services = plantypes.MergeServices(services, nservices)
				logrus.Debugf("[%s] Done", config.Name)
				if len(nservices) > 0 {
					found = true
					relpath, _ := filepath.Rel(inputPath, path)
					logrus.Infof("%s in %s", getNamedAndUnNamedServicesLogMessage(nservices), relpath)
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
