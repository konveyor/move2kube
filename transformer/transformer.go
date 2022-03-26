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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

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
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	initialized      = false
	transformerTypes = map[string]reflect.Type{}
	transformers     = []Transformer{}
	transformerMap   = map[string]Transformer{}
)

// Transformer interface defines transformer that transforms files and converts it to ir representation
type Transformer interface {
	Init(tc transformertypes.Transformer, env *environment.Environment) (err error)
	// GetConfig returns the transformer config
	GetConfig() (transformertypes.Transformer, *environment.Environment)
	DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error)
	Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error)
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
		new(dockerfilegenerator.RustDockerfileGenerator),
		new(dockerfilegenerator.DotNetCoreDockerfileGenerator),
		new(java.JarAnalyser),
		new(java.WarAnalyser),
		new(java.EarAnalyser),
		new(java.Tomcat),
		new(java.Liberty),
		new(java.Jboss),
		new(java.MavenAnalyser),
		new(java.GradleAnalyser),
		new(java.ZuulAnalyser),
		new(windows.WinConsoleAppDockerfileGenerator),
		new(windows.WinSilverLightWebAppDockerfileGenerator),
		new(windows.WinWebAppDockerfileGenerator),

		new(compose.ComposeAnalyser),
		new(compose.ComposeGenerator),

		new(CloudFoundry),

		new(containerimage.ContainerImagesPushScript),

		new(kubernetes.ClusterSelectorTransformer),
		new(kubernetes.Kubernetes),
		new(kubernetes.Knative),
		new(kubernetes.Tekton),
		new(kubernetes.BuildConfig),
		new(kubernetes.Parameterizer),
		new(kubernetes.KubernetesVersionChanger),

		new(ReadMeGenerator),
	}
	transformerTypes = common.GetTypesMap(transformerObjs)
}

// Init initializes the transformers
func Init(assetsPath, sourcePath string, selector labels.Selector, outputPath, projName string) (err error) {
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
	InitTransformers(transformerFiles, selector, sourcePath, outputPath, projName, false)
	return nil
}

// InitTransformers initializes a subset of transformers
func InitTransformers(transformerToInit map[string]string, selector labels.Selector, sourcePath string, outputPath, projName string, logError bool) error {
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
	sort.Strings(tns)
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
				Isolated:        tc.Spec.Isolated,
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
				transformers = append(transformers, t)
				transformerMap[tn] = t
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
func GetInitializedTransformers() []Transformer {
	return transformers
}

// GetTransformerByName returns the transformer chosen by name
func GetTransformerByName(name string) (t Transformer, err error) {
	if t, ok := transformerMap[name]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("no transformer found")
}

// GetInitializedTransformersF returns the list of initialized transformers after filtering
func GetInitializedTransformersF(filters labels.Selector) []Transformer {
	filteredTransformers := []Transformer{}
	for _, t := range GetInitializedTransformers() {
		tc, _ := t.GetConfig()
		if tc.ObjectMeta.Labels == nil {
			tc.ObjectMeta.Labels = map[string]string{}
		}
		if !filters.Matches(labels.Set(tc.ObjectMeta.Labels)) {
			continue
		}
		filteredTransformers = append(filteredTransformers, t)
	}
	return filteredTransformers
}

// GetServices returns the list of services detected in a directory
func GetServices(prjName string, dir string) (services map[string][]plantypes.PlanArtifact, err error) {
	services = map[string][]plantypes.PlanArtifact{}
	logrus.Infoln("Planning Transformation - Base Directory")
	logrus.Debugf("Transformers : %+v", transformers)
	for _, t := range transformers {
		config, env := t.GetConfig()
		if err := env.Reset(); err != nil {
			logrus.Errorf("failed to reset the environment for the transformer %s . Error: %q", config.Name, err)
			continue
		}
		if config.Spec.DirectoryDetect.Levels != 1 {
			continue
		}
		logrus.Infof("[%s] Planning transformation", config.Name)
		nservices, err := t.DirectoryDetect(env.Encode(dir).(string))
		if err != nil {
			logrus.Errorf("[%s] Failed : %s", config.Name, err)
		} else {
			nservices := getPlanArtifactsFromArtifacts(*env.Decode(&nservices).(*map[string][]transformertypes.Artifact), config)
			services = plantypes.MergeServices(services, nservices)
			if len(nservices) > 0 {
				logrus.Infof(getNamedAndUnNamedServicesLogMessage(nservices))
			}
			common.PlanProgressNumBaseDetectTransformers++
			logrus.Infof("[%s] Done", config.Name)
		}
	}
	logrus.Infof("[Base Directory] %s", getNamedAndUnNamedServicesLogMessage(services))
	logrus.Infoln("Transformation planning - Base Directory done")
	logrus.Infoln("Planning Transformation - Directory Walk")
	nservices, err := walkForServices(dir, services)
	if err != nil {
		logrus.Errorf("Transformation planning - Directory Walk failed : %s", err)
	} else {
		services = nservices
		logrus.Infoln("Transformation planning - Directory Walk done")
	}
	logrus.Infof("[Directory Walk] %s", getNamedAndUnNamedServicesLogMessage(services))
	services = nameServices(prjName, services)
	logrus.Infof("[Named Services] Identified %d named services", len(services))
	return
}

func walkForServices(inputPath string, bservices map[string][]plantypes.PlanArtifact) (services map[string][]plantypes.PlanArtifact, err error) {
	services = bservices
	ignoreDirectories, ignoreContents := getIgnorePaths(inputPath)
	knownServiceDirPaths := []string{}

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
		if common.IsStringPresent(knownServiceDirPaths, path) {
			return filepath.SkipDir //TODO: Should we go inside the directory in this case?
		}
		if common.IsStringPresent(ignoreDirectories, path) {
			if common.IsStringPresent(ignoreContents, path) {
				return filepath.SkipDir
			}
			return nil
		}
		common.PlanProgressNumDirectories++
		logrus.Debugf("Planning in directory %s", path)
		numfound := 0
		skipThisDir := false
		for _, transformer := range transformers {
			config, env := transformer.GetConfig()
			logrus.Debugf("[%s] Planning in directory %s", config.Name, path)
			if err := env.Reset(); err != nil {
				logrus.Errorf("failed to reset the environment for the transformer %s . Error: %q", config.Name, err)
				continue
			}
			if config.Spec.DirectoryDetect.Levels == 1 || config.Spec.DirectoryDetect.Levels == 0 {
				continue
			}
			newServicesToArtifacts, err := transformer.DirectoryDetect(env.Encode(path).(string))
			if err != nil {
				logrus.Warnf("[%s] directory detect failed. Error: %q", config.Name, err)
				continue
			}
			for _, newServiceArtifacts := range newServicesToArtifacts {
				for _, newServiceArtifact := range newServiceArtifacts {
					knownServiceDirPaths = append(knownServiceDirPaths, newServiceArtifact.Paths[artifacts.ServiceDirPathType]...)
					for _, serviceDirPath := range newServiceArtifact.Paths[artifacts.ServiceDirPathType] {
						if serviceDirPath == path {
							skipThisDir = true
							break
						}
					}
				}
			}
			newPlanServices := getPlanArtifactsFromArtifacts(*env.Decode(&newServicesToArtifacts).(*map[string][]transformertypes.Artifact), config)
			services = plantypes.MergeServices(services, newPlanServices)
			logrus.Debugf("[%s] Done", config.Name)
			numfound += len(newServicesToArtifacts)
			if len(newServicesToArtifacts) > 0 {
				relpath, _ := filepath.Rel(inputPath, path)
				logrus.Infof("%s in %s", getNamedAndUnNamedServicesLogMessage(newPlanServices), relpath)
			}
		}
		logrus.Debugf("planning finished for the directory %s and %d services were detected", path, numfound)
		if skipThisDir || common.IsStringPresent(ignoreContents, path) {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("failed to walk through the directory at path %s . Error: %q", inputPath, err)
	}
	return services, err
}

type processType int

const (
	consume processType = iota
	passthrough
	dependency
)

// Transform transforms as per the plan
func Transform(planServices []plantypes.PlanArtifact, sourceDir, outputPath string) (err error) {
	var allArtifacts []transformertypes.Artifact
	newArtifactsToProcess := []transformertypes.Artifact{}
	pathMappings := []transformertypes.PathMapping{}
	iteration := 1
	logrus.Infof("Iteration %d", iteration)
	for _, a := range planServices {
		a.ProcessWith = *metav1.AddLabelToSelector(&a.ProcessWith, transformertypes.LabelName, string(a.TransformerName))
		if a.Type == "" {
			a.Type = artifacts.ServiceArtifactType
		}
		if a.Name == "" {
			a.Name = a.ServiceName
		}
		serviceConfig := artifacts.ServiceConfig{
			ServiceName: a.ServiceName,
		}
		if a.Configs == nil {
			a.Configs = map[transformertypes.ConfigType]interface{}{}
		}
		a.Configs[artifacts.ServiceConfigType] = serviceConfig
		newArtifactsToProcess = append(newArtifactsToProcess, a.Artifact)
	}
	allArtifacts = newArtifactsToProcess
	for {
		iteration++
		logrus.Infof("Iteration %d - %d artifacts to process", iteration, len(newArtifactsToProcess))
		newPathMappings, newArtifacts, _ := transform(newArtifactsToProcess, allArtifacts, consume, nil)
		pathMappings = append(pathMappings, newPathMappings...)
		if err = os.RemoveAll(outputPath); err != nil {
			logrus.Errorf("Unable to delete %s : %s", outputPath, err)
		}
		err = processPathMappings(pathMappings, sourceDir, outputPath)
		if err != nil {
			logrus.Errorf("Unable to process path mappings")
		}
		if len(newArtifacts) == 0 {
			break
		}
		logrus.Infof("Created %d pathMappings and %d artifacts. Total Path Mappings : %d. Total Artifacts : %d.", len(newPathMappings), len(newArtifacts), len(pathMappings), len(allArtifacts))
		allArtifacts = append(allArtifacts, newArtifacts...)
		newArtifactsToProcess = newArtifacts
	}
	return nil
}

func transform(newArtifactsToProcess, allArtifacts []transformertypes.Artifact, pt processType, depSel labels.Selector) (pathMappings []transformertypes.PathMapping, newArtifactsCreated, updatedArtifacts []transformertypes.Artifact) {
	if pt == dependency && (depSel == nil || depSel.String() == "") {
		return nil, nil, newArtifactsToProcess
	}
	for _, transformer := range transformers {
		tconfig, env := transformer.GetConfig()
		if pt == dependency && !depSel.Matches(labels.Set(tconfig.Labels)) {
			continue
		}
		artifactsToProcess, artifactsToNotProcess := getArtifactsToProcess(newArtifactsToProcess, allArtifacts, tconfig, pt)
		if len(artifactsToProcess) == 0 {
			continue
		}
		logrus.Debugf("Transformer %s will be processing %d artifacts in %d mode", tconfig.Name, len(artifactsToProcess), pt)
		// Dependency processing
		dependencyCreatedNewPathMappings, dependencyCreatedNewArtifacts, dependencyUpdatedArtifacts := transform(artifactsToProcess, allArtifacts, dependency, tconfig.Spec.DependencySelector)
		logrus.Debugf("Dependency processing resulted in %d pathmappings, %d new artifacts and %d updated artifacts", len(dependencyCreatedNewPathMappings), len(dependencyCreatedNewArtifacts), len(dependencyUpdatedArtifacts))
		pathMappings = append(pathMappings, dependencyCreatedNewPathMappings...)
		artifactsToConsume, artifactsToNotConsume := getArtifactsToProcess(dependencyUpdatedArtifacts, allArtifacts, tconfig, pt)
		if len(artifactsToNotConsume) != 0 {
			logrus.Errorf("Artifacts to not consume : %d. This should have been 0.", len(artifactsToNotConsume))
		}
		producedNewPathMappings, producedNewArtifacts, err := runSingleTransform(artifactsToConsume, allArtifacts, transformer, tconfig, env)
		if err != nil {
			continue
		}
		pathMappings = append(pathMappings, producedNewPathMappings...)
		artifactsToPassThrough := []transformertypes.Artifact{}
		artifactsAlreadyPassedThrough := []transformertypes.Artifact{}
		if pt == consume {
			artifactsToPassThrough = append(dependencyCreatedNewArtifacts, producedNewArtifacts...)
		} else if pt == passthrough || pt == dependency {
			for _, a := range producedNewArtifacts {
				if c, ok := tconfig.Spec.ConsumedArtifacts[a.Type]; ok && (c.Mode != transformertypes.MandatoryPassThrough && c.Mode != transformertypes.OnDemandPassThrough) {
					artifactsToPassThrough = append(artifactsToPassThrough, a)
				} else {
					artifactsAlreadyPassedThrough = append(artifactsAlreadyPassedThrough, a)
				}
			}
		}
		passedThroughPathMappings, passedThroughNewArtifactsCreated, passedThroughUpdatedArtifacts := transform(artifactsToPassThrough, allArtifacts, passthrough, nil)
		pathMappings = append(pathMappings, passedThroughPathMappings...)
		newArtifactsCreated = append(newArtifactsCreated, passedThroughNewArtifactsCreated...)
		if pt == consume {
			newArtifactsCreated = append(newArtifactsCreated, passedThroughUpdatedArtifacts...)
		}
		updatedArtifacts = append(updatedArtifacts, passedThroughUpdatedArtifacts...)
		if pt == passthrough || pt == dependency {
			newArtifactsToProcess = artifactsToNotProcess
			newArtifactsToProcess = append(newArtifactsToProcess, passedThroughUpdatedArtifacts...)
			newArtifactsToProcess = append(newArtifactsToProcess, artifactsAlreadyPassedThrough...)
		}
		logrus.Infof("Transformer %s Done", tconfig.Name)
	}
	if pt == passthrough || pt == dependency {
		logrus.Debugf("Created %d pathMappings, %d artifacts, %d updated artifacts from transform while passing through/dependency.", len(pathMappings), len(newArtifactsCreated), len(newArtifactsToProcess))
		return pathMappings, newArtifactsCreated, newArtifactsToProcess
	}
	logrus.Debugf("Created %d pathMappings and %d artifacts from transform.", len(pathMappings), len(newArtifactsCreated))
	return pathMappings, newArtifactsCreated, nil
}

func runSingleTransform(artifactsToProcess, allArtifacts []transformertypes.Artifact, t Transformer, tconfig transformertypes.Transformer, env *environment.Environment) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	logrus.Infof("Transformer %s processing %d artifacts", tconfig.Name, len(artifactsToProcess))
	env.Reset()
	producedNewPathMappings, producedNewArtifacts, err := t.Transform(*env.Encode(&artifactsToProcess).(*[]transformertypes.Artifact), *env.Encode(&allArtifacts).(*[]transformertypes.Artifact))
	if err != nil {
		logrus.Errorf("Unable to transform artifacts using %s : %s", tconfig.Name, err)
		return producedNewPathMappings, producedNewArtifacts, err
	}
	filteredArtifacts := []transformertypes.Artifact{}
	for _, na := range producedNewArtifacts {
		if ps, ok := tconfig.Spec.ProducedArtifacts[na.Type]; ok && !ps.Disabled {
			filteredArtifacts = append(filteredArtifacts, na)
		} else {
			logrus.Warnf("Ignoring artifact %s of type %s in transformer %s", na.Name, na.Type, tconfig.Name)
		}
	}
	producedNewArtifacts = filteredArtifacts
	producedNewPathMappings = env.ProcessPathMappings(producedNewPathMappings)
	producedNewPathMappings = *env.DownloadAndDecode(&producedNewPathMappings, true).(*[]transformertypes.PathMapping)
	err = processPathMappings(producedNewPathMappings, env.Source, env.Output)
	if err != nil {
		logrus.Errorf("Unable to process path mappings")
	}
	producedNewArtifacts = *env.DownloadAndDecode(&producedNewArtifacts, false).(*[]transformertypes.Artifact)
	producedNewArtifacts = postProcessArtifacts(producedNewArtifacts, tconfig)
	return producedNewPathMappings, producedNewArtifacts, nil
}

func getArtifactsToProcess(newArtifactsToProcess, allArtifacts []transformertypes.Artifact, tconfig transformertypes.Transformer, pt processType) (artifactsToProcess, artifactsNotProcessed []transformertypes.Artifact) {
	artifactsNotProcessed = []transformertypes.Artifact{}
	artifactsToProcess = []transformertypes.Artifact{}
	for _, na := range newArtifactsToProcess {
		processConfig := tconfig.Spec.ConsumedArtifacts
		if processSpec, ok := processConfig[na.Type]; ok && !processSpec.Disabled {
			switch pt {
			case passthrough:
				if processSpec.Mode != transformertypes.MandatoryPassThrough {
					artifactsNotProcessed = append(artifactsNotProcessed, na)
					continue
				}
			case dependency:
				if processSpec.Mode != transformertypes.OnDemandPassThrough {
					artifactsNotProcessed = append(artifactsNotProcessed, na)
					continue
				}
			default:
				if processSpec.Mode == transformertypes.MandatoryPassThrough || processSpec.Mode == transformertypes.OnDemandPassThrough {
					artifactsNotProcessed = append(artifactsNotProcessed, na)
					continue
				}
			}
			selected := true
			if na.ProcessWith.String() != "" && pt != passthrough && pt != dependency {
				s, err := selectTransformer(na.ProcessWith, tconfig)
				if err != nil {
					logrus.Errorf("Unable to process selector for transformer %s : %s", tconfig.Name, err)
				} else if !s {
					selected = false
				}
			}
			if !selected {
				artifactsNotProcessed = append(artifactsNotProcessed, na)
				continue
			}
			if processSpec.Merge {
				artifactsToProcess = mergeArtifacts(append(artifactsToProcess, updatedArtifacts(allArtifacts, na)...))
			}
			artifactsToProcess = append(artifactsToProcess, na)
		} else {
			artifactsNotProcessed = append(artifactsNotProcessed, na)
		}
	}
	return
}
