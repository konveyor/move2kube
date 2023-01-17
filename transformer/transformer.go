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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	containertypes "github.com/konveyor/move2kube/environment/container"
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
	"github.com/konveyor/move2kube/types"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	graphtypes "github.com/konveyor/move2kube/types/graph"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// Transformer interface defines transformer that transforms files and converts it to ir representation
type Transformer interface {
	Init(tc transformertypes.Transformer, env *environment.Environment) (err error)
	// GetConfig returns the transformer config
	GetConfig() (transformertypes.Transformer, *environment.Environment)
	DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error)
	Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error)
}

type processType int

const (
	consume processType = iota
	passthrough
	dependency
)

const (
	// DEFAULT_SELECTED_LABEL is a label that can be used to remove a transformer from the list of transformers that are selected by default.
	DEFAULT_SELECTED_LABEL = types.GroupName + "/default-selected"
)

var (
	initialized                  = false
	transformerTypes             = map[string]reflect.Type{}
	transformers                 = []Transformer{}
	invokedByDefaultTransformers = []Transformer{}
	transformerMap               = map[string]Transformer{}
)

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
		new(CNBContainerizer),
		new(compose.ComposeAnalyser),
		new(compose.ComposeGenerator),

		new(CloudFoundry),

		new(containerimage.ContainerImagesPushScript),

		new(kubernetes.ClusterSelectorTransformer),
		new(kubernetes.Kubernetes),
		new(kubernetes.Knative),
		new(kubernetes.Tekton),
		new(kubernetes.ArgoCD),
		new(kubernetes.BuildConfig),
		new(kubernetes.Parameterizer),
		new(kubernetes.KubernetesVersionChanger),
		new(kubernetes.OperatorTransformer),

		new(ReadMeGenerator),
	}
	transformerTypes = common.GetTypesMap(transformerObjs)
}

// RegisterTransformer allows for adding transformers after initialization
func RegisterTransformer(tf Transformer) error {
	tval := reflect.ValueOf(tf)
	t := reflect.TypeOf(tval.Interface()).Elem()
	tn := t.Name()
	if ot, ok := transformerTypes[tn]; ok {
		logrus.Errorf("Two transformer classes have the same name '%s' : %T , %T; Ignoring %T", tn, ot, t, t)
		return fmt.Errorf("couldn't register transformer '%s' because a transformer with that name already exists", tn)
	}
	transformerTypes[tn] = t
	return nil
}

// Init initializes the transformers
func Init(assetsPath, sourcePath string, selector labels.Selector, outputPath, projName string) (map[string]string, error) {
	yamlPaths, err := common.GetFilesByExt(assetsPath, []string{".yml", ".yaml"})
	if err != nil {
		return nil, fmt.Errorf("failed to look for yaml files in the directory '%s' . Error: %w", assetsPath, err)
	}
	transformerYamlPaths := map[string]string{}
	for _, yamlPath := range yamlPaths {
		tc, err := getTransformerConfig(yamlPath)
		if err != nil {
			logrus.Debugf("failed to load the transformer config file at path '%s' . Error: %w", yamlPath, err)
			continue
		}
		if otc, ok := transformerYamlPaths[tc.Name]; ok {
			logrus.Warnf("Duplicate transformer configs with same name '%s' found. Ignoring '%s' in favor of '%s'", tc.Name, otc, yamlPath)
		}
		transformerYamlPaths[tc.Name] = yamlPath
	}
	deselectedTransformers, err := InitTransformers(transformerYamlPaths, selector, sourcePath, outputPath, projName, false, false)
	if err != nil {
		return deselectedTransformers, fmt.Errorf(
			"failed to initialize the transformers using the source path '%s' and the output path '%s' . Error: %w",
			sourcePath, outputPath, err,
		)
	}
	return deselectedTransformers, nil
}

// InitTransformers initializes a subset of transformers
func InitTransformers(transformerYamlPaths map[string]string, selector labels.Selector, sourcePath, outputPath, projName string, logError, preExistingPlan bool) (map[string]string, error) {
	logrus.Trace("InitTransformers start")
	defer logrus.Trace("InitTransformers end")
	if initialized {
		logrus.Debug("already initialized")
		return nil, nil
	}
	transformerFilterString := qaengine.FetchStringAnswer(
		common.TransformerSelectorKey,
		"Specify a Kubernetes style selector to select only the transformers that you want to run.",
		[]string{"Leave empty to select everything. This is the default."},
		"",
		nil,
	)
	if transformerFilterString != "" {
		if transformerFilter, err := common.ConvertStringSelectorsToSelectors(transformerFilterString); err != nil {
			logrus.Errorf("failed to parse the transformer filter string: %s . Error: %q", transformerFilterString, err)
		} else {
			reqs, _ := transformerFilter.Requirements()
			selector = selector.Add(reqs...)
		}
	}
	transformerConfigs := getFilteredTransformers(transformerYamlPaths, selector, logError)
	deselectedTransformers := map[string]string{}
	for transformerName, transformerPath := range transformerYamlPaths {
		if _, ok := transformerConfigs[transformerName]; !ok {
			deselectedTransformers[transformerName] = transformerPath
		}
	}
	transformerNames := []string{}
	transformerNamesSelectedByDefault := []string{}
	for transformerName, t := range transformerConfigs {
		transformerNames = append(transformerNames, transformerName)
		if v, ok := t.ObjectMeta.Labels[DEFAULT_SELECTED_LABEL]; !ok || cast.ToBool(v) {
			transformerNamesSelectedByDefault = append(transformerNamesSelectedByDefault, transformerName)
		}
	}
	sort.Strings(transformerNames)
	selectedTransformerNames := qaengine.FetchMultiSelectAnswer(
		common.ConfigTransformerTypesKey,
		"Select all transformer types that you are interested in:",
		[]string{"Services that don't support any of the transformer types you are interested in will be ignored."},
		transformerNamesSelectedByDefault,
		transformerNames,
		nil,
	)
	for _, transformerName := range transformerNames {
		if !common.IsPresent(selectedTransformerNames, transformerName) {
			deselectedTransformers[transformerName] = transformerYamlPaths[transformerName]
		}
	}
	for _, selectedTransformerName := range selectedTransformerNames {
		transformerConfig, ok := transformerConfigs[selectedTransformerName]
		if !ok {
			logrus.Errorf("failed to find the transformer with the name: '%s'", selectedTransformerName)
			continue
		}
		transformerClass, ok := transformerTypes[transformerConfig.Spec.Class]
		if !ok {
			logrus.Errorf("failed to find the transformer class '%s' . Valid transformer classes are: %+v", transformerConfig.Spec.Class, transformerTypes)
			continue
		}
		transformer := reflect.New(transformerClass).Interface().(Transformer)
		transformerContextPath := filepath.Dir(transformerConfig.Spec.TransformerYamlPath)
		envInfo := environment.EnvInfo{
			Name:            transformerConfig.Name,
			ProjectName:     projName,
			Isolated:        transformerConfig.Spec.Isolated,
			Source:          sourcePath,
			Output:          outputPath,
			Context:         transformerContextPath,
			RelTemplatesDir: transformerConfig.Spec.TemplatesDir,
			EnvPlatformConfig: environmenttypes.EnvPlatformConfig{
				Container: environmenttypes.Container{},
				Platforms: []string{runtime.GOOS},
			},
		}
		for src, dest := range transformerConfig.Spec.ExternalFiles {
			if err := filesystem.Replicate(filepath.Join(transformerContextPath, src), filepath.Join(transformerContextPath, dest)); err != nil {
				logrus.Errorf(
					"failed to copy external files for transformer '%s' from source path '%s' to destination path '%s' . Error: %q",
					transformerConfig.Name, src, dest, err,
				)
			}
		}
		if preExistingPlan {
			if v, ok := transformerConfig.Labels["move2kube.konveyor.io/container-based"]; ok && cast.ToBool(v) {
				envInfo.SpawnContainers = true
			}
		}
		env, err := environment.NewEnvironment(envInfo, nil)
		if err != nil {
			return deselectedTransformers, fmt.Errorf("failed to create the environment %+v . Error: %w", envInfo, err)
		}
		if err := transformer.Init(transformerConfig, env); err != nil {
			if errors.Is(err, containertypes.ErrNoContainerRuntime) {
				logrus.Debugf("failed to initialize the transformer '%s' . Error: %q", transformerConfig.Name, err)
			} else {
				logrus.Errorf("failed to initialize the transformer '%s' . Error: %q", transformerConfig.Name, err)
			}
			continue
		}
		transformers = append(transformers, transformer)
		transformerMap[selectedTransformerName] = transformer
		if transformerConfig.Spec.InvokedByDefault.Enabled {
			invokedByDefaultTransformers = append(invokedByDefaultTransformers, transformer)
		}
	}
	initialized = true
	return deselectedTransformers, nil
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
func GetServices(prjName string, dir string) (map[string][]plantypes.PlanArtifact, error) {
	planServices := map[string][]plantypes.PlanArtifact{}
	logrus.Infoln("Planning started on the base directory")
	logrus.Debugf("Transformers: %+v", transformers)
	for _, transformer := range transformers {
		config, env := transformer.GetConfig()
		if err := env.Reset(); err != nil {
			logrus.Errorf("failed to reset the environment for the transformer %s . Error: %q", config.Name, err)
			continue
		}
		if config.Spec.DirectoryDetect.Levels != 1 {
			continue
		}
		logrus.Infof("[%s] Planning", config.Name)
		newServices, err := transformer.DirectoryDetect(env.Encode(dir).(string))
		if err != nil {
			logrus.Errorf("[%s] failed to look for services in the directory '%s' . Error: %q", config.Name, dir, err)
			continue
		}
		newPlanServices := getPlanArtifactsFromArtifacts(*env.Decode(&newServices).(*map[string][]transformertypes.Artifact), config)
		planServices = plantypes.MergeServices(planServices, newPlanServices)
		if len(newPlanServices) > 0 {
			logrus.Infof(getNamedAndUnNamedServicesLogMessage(newPlanServices))
		}
		common.PlanProgressNumBaseDetectTransformers++
		logrus.Infof("[%s] Done", config.Name)
	}
	logrus.Infof("[Base Directory] %s", getNamedAndUnNamedServicesLogMessage(planServices))
	logrus.Infoln("Planning finished on the base directory")
	logrus.Infoln("Planning started on its sub directories")
	nservices, err := walkForServices(dir, planServices)
	if err != nil {
		logrus.Errorf("Transformation planning - Directory Walk failed : %s", err)
	} else {
		planServices = nservices
		logrus.Infoln("Planning finished on its sub directories")
	}
	logrus.Infof("[Directory Walk] %s", getNamedAndUnNamedServicesLogMessage(planServices))
	planServices = nameServices(prjName, planServices)
	logrus.Infof("[Named Services] Identified %d named services", len(planServices))
	return planServices, nil
}

func walkForServices(inputPath string, bservices map[string][]plantypes.PlanArtifact) (map[string][]plantypes.PlanArtifact, error) {
	services := bservices
	ignoreDirectories, ignoreContents := getIgnorePaths(inputPath)
	knownServiceDirPaths := []string{}

	err := filepath.WalkDir(inputPath, func(path string, info os.DirEntry, err error) error {
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
		if common.IsPresent(knownServiceDirPaths, path) {
			return filepath.SkipDir // TODO: Should we go inside the directory in this case?
		}
		if common.IsPresent(ignoreDirectories, path) {
			if common.IsPresent(ignoreContents, path) {
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
			numfound += len(newPlanServices)
			if len(newPlanServices) > 0 {
				msg := getNamedAndUnNamedServicesLogMessage(newPlanServices)
				relpath, err := filepath.Rel(inputPath, path)
				if err != nil {
					logrus.Errorf("failed to make the directory %s relative to the input directory %s . Error: %q", path, inputPath, err)
					logrus.Infof("%s in %s", msg, path)
					continue
				}
				logrus.Infof("%s in %s", msg, relpath)
			}
		}
		logrus.Debugf("planning finished for the directory %s and %d services were detected", path, numfound)
		if skipThisDir || common.IsPresent(ignoreContents, path) {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return services, fmt.Errorf("failed to walk through the directory at path %s . Error: %q", inputPath, err)
	}
	return services, nil
}

func summarizeArtifacts(artifacts []transformertypes.Artifact) []string {
	arts := []string{}
	for _, a := range artifacts {
		arts = append(arts, fmt.Sprintf("%s - %s", a.Name, a.Type))
	}
	return arts
}

func summarizePathMappings(pathMappings []transformertypes.PathMapping) string {
	paths := []string{}
	for _, pathMapping := range pathMappings {
		paths = append(paths, fmt.Sprintf("(%s, %s, %s)", pathMapping.Type, pathMapping.SrcPath, pathMapping.DestPath))
	}
	return strings.Join(paths, "\n")
}

// Transform transforms as per the plan
func Transform(planArtifacts []plantypes.PlanArtifact, sourceDir, outputPath string) error {
	var allArtifacts []transformertypes.Artifact
	newArtifactsToProcess := []transformertypes.Artifact{}
	pathMappings := []transformertypes.PathMapping{}
	defaultNewArtifactsToProcess := []transformertypes.Artifact{}
	iteration := 1
	// transform default transformers
	graph := graphtypes.NewGraph()
	startVertexId := graph.AddVertex("start", iteration, nil)
	for _, invokedByDefaultTransformer := range invokedByDefaultTransformers {
		tDefaultConfig, defaultEnv := invokedByDefaultTransformer.GetConfig()
		newPathMappings, defaultArtifacts, err := runSingleTransform(nil, nil, invokedByDefaultTransformer, tDefaultConfig, defaultEnv, graph, iteration)
		if err != nil {
			logrus.Errorf("failed to transform using the transformer %s. Error: %q", tDefaultConfig.Name, err)
		}
		defaultNewArtifactsToProcess = append(defaultNewArtifactsToProcess, defaultArtifacts...)
		pathMappings = append(pathMappings, newPathMappings...)
	}
	logrus.Infof("Iteration %d", iteration)
	for _, planArtifact := range planArtifacts {
		planArtifact.ProcessWith = *metav1.AddLabelToSelector(&planArtifact.ProcessWith, transformertypes.LabelName, string(planArtifact.TransformerName))
		if planArtifact.Type == "" {
			planArtifact.Type = artifacts.ServiceArtifactType
		}
		if planArtifact.Name == "" {
			planArtifact.Name = planArtifact.ServiceName
		}
		serviceConfig := artifacts.ServiceConfig{
			ServiceName: planArtifact.ServiceName,
		}
		if planArtifact.Configs == nil {
			planArtifact.Configs = map[transformertypes.ConfigType]interface{}{}
		}
		planArtifact.Configs[artifacts.ServiceConfigType] = serviceConfig
		newArtifactsToProcess = append(newArtifactsToProcess, planArtifact.Artifact)
	}

	// logging
	for _, artifact := range newArtifactsToProcess {
		artifact.Configs[graphtypes.GraphSourceVertexKey] = startVertexId
	}
	newArtifactsToProcess = append(newArtifactsToProcess, defaultNewArtifactsToProcess...)
	allArtifacts = newArtifactsToProcess
	// logging

	for {
		iteration++
		logrus.Infof("Iteration %d - %d artifacts to process", iteration, len(newArtifactsToProcess))
		newPathMappings, newArtifacts, _ := transform(newArtifactsToProcess, allArtifacts, consume, nil, graph, iteration)
		pathMappings = append(pathMappings, newPathMappings...)
		if err := os.RemoveAll(outputPath); err != nil {
			return fmt.Errorf("failed to remove the output directory %s . Error: %q", outputPath, err)
		}
		if err := processPathMappings(pathMappings, sourceDir, outputPath); err != nil {
			return fmt.Errorf("failed to process the path mappings: %+v . Error: %q", pathMappings, err)
		}
		if len(newArtifacts) == 0 {
			break
		}
		logrus.Infof("Created %d pathMappings and %d artifacts. Total Path Mappings : %d. Total Artifacts : %d.", len(newPathMappings), len(newArtifacts), len(pathMappings), len(allArtifacts))
		allArtifacts = append(allArtifacts, newArtifacts...)
		newArtifactsToProcess = newArtifacts
	}

	// logging
	{
		graphFilePath := graphtypes.GraphFileName
		graphFile, err := os.Create(graphFilePath)
		if err != nil {
			logrus.Errorf("failed to create a %s file to write to the graph. Error: %q", graphFilePath, err)
		} else {
			enc := json.NewEncoder(graphFile)
			enc.SetIndent("", "    ")
			if err := enc.Encode(graph); err != nil {
				logrus.Errorf("failed to encode the graph as json. Error: %q", err)
			}
		}
	}
	// logging

	return nil
}

func transform(newArtifactsToProcess, allArtifacts []transformertypes.Artifact, pt processType, depSel labels.Selector, graph *graphtypes.Graph, iteration int) (pathMappings []transformertypes.PathMapping, newArtifactsCreated, updatedArtifacts []transformertypes.Artifact) {
	logrus.Trace("transform start")
	defer logrus.Trace("transform end")
	if pt == dependency && (depSel == nil || depSel.String() == "") {
		return nil, nil, newArtifactsToProcess
	}
	for _, transformer := range transformers {
		tConfig, env := transformer.GetConfig()
		if pt == dependency && !depSel.Matches(labels.Set(tConfig.Labels)) {
			continue
		}
		artifactsToProcess, artifactsToNotProcess := getArtifactsToProcess(newArtifactsToProcess, allArtifacts, tConfig, pt)
		if len(artifactsToProcess) == 0 {
			continue
		}

		logrus.Debugf("Transformer %s will be processing %d artifacts in %d mode", tConfig.Name, len(artifactsToProcess), pt)

		// Dependency processing
		dependencyCreatedNewPathMappings, dependencyCreatedNewArtifacts, dependencyUpdatedArtifacts := transform(artifactsToProcess, allArtifacts, dependency, tConfig.Spec.DependencySelector, graph, iteration)
		pathMappings = append(pathMappings, dependencyCreatedNewPathMappings...)
		// Dependency processing

		artifactsToConsume, artifactsToNotConsume := getArtifactsToProcess(dependencyUpdatedArtifacts, allArtifacts, tConfig, pt)
		if len(artifactsToNotConsume) != 0 {
			logrus.Errorf("Artifacts to not consume: %d. This should have been 0.", len(artifactsToNotConsume))
		}

		logrus.Infof("Transformer %s processing %d artifacts", tConfig.Name, len(artifactsToConsume))

		producedNewPathMappings, producedNewArtifacts, err := runSingleTransform(artifactsToConsume, allArtifacts, transformer, tConfig, env, graph, iteration)
		if err != nil {
			logrus.Errorf("failed to run a single transformation using the transformer %+v on the artifacts %+v . Error: %q", tConfig, artifactsToConsume, err)
			continue
		}
		pathMappings = append(pathMappings, producedNewPathMappings...)
		artifactsToPassThrough := []transformertypes.Artifact{}
		artifactsAlreadyPassedThrough := []transformertypes.Artifact{}
		if pt == consume {
			artifactsToPassThrough = append(dependencyCreatedNewArtifacts, producedNewArtifacts...)
		} else if pt == passthrough || pt == dependency {
			for _, a := range producedNewArtifacts {
				if c, ok := tConfig.Spec.ConsumedArtifacts[a.Type]; ok &&
					(c.Mode != transformertypes.MandatoryPassThrough && c.Mode != transformertypes.OnDemandPassThrough) {
					artifactsToPassThrough = append(artifactsToPassThrough, a)
				} else {
					artifactsAlreadyPassedThrough = append(artifactsAlreadyPassedThrough, a)
				}
			}
		}

		passedThroughPathMappings, passedThroughNewArtifactsCreated, passedThroughUpdatedArtifacts := transform(artifactsToPassThrough, allArtifacts, passthrough, nil, graph, iteration)

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
		logrus.Infof("Transformer %s Done", tConfig.Name)
	}
	if pt == passthrough || pt == dependency {
		logrus.Debugf("Created %d pathMappings, %d artifacts, %d updated artifacts from transform while passing through/dependency.", len(pathMappings), len(newArtifactsCreated), len(newArtifactsToProcess))
		return pathMappings, newArtifactsCreated, newArtifactsToProcess
	}
	logrus.Debugf("Created %d pathMappings and %d artifacts from transform.", len(pathMappings), len(newArtifactsCreated))
	return pathMappings, newArtifactsCreated, nil
}

func runSingleTransform(artifactsToProcess, allArtifacts []transformertypes.Artifact, transformer Transformer, tconfig transformertypes.Transformer, env *environment.Environment, graph *graphtypes.Graph, iteration int) (newPathMappings []transformertypes.PathMapping, newArtifacts []transformertypes.Artifact, err error) {
	logrus.Trace("runSingleTransform start")
	defer logrus.Trace("runSingleTransform end")
	if err := env.Reset(); err != nil {
		return nil, nil, fmt.Errorf("failed to reset the environment: %+v Error: %q", env, err)
	}
	newPathMappings, newArtifacts, err = transformer.Transform(
		*env.Encode(&artifactsToProcess).(*[]transformertypes.Artifact),
		*env.Encode(&allArtifacts).(*[]transformertypes.Artifact),
	)
	// logging
	{
		vertexName := fmt.Sprintf("iteration: %d\nclass: %s\nname: %s", iteration, tconfig.Spec.Class, tconfig.Name)
		targetVertexId := graph.AddVertex(
			vertexName,
			iteration,
			map[string]interface{}{
				"consumedArtifacts": summarizeArtifacts(artifactsToProcess),
				"producedArtifacts": summarizeArtifacts(newArtifacts),
				"pathMappings":      summarizePathMappings(newPathMappings),
			},
		)
		// transformers that are invoked by default has source vertex as start
		if tconfig.Spec.InvokedByDefault.Enabled {
			edgeName := fmt.Sprintf("%d -> %d (invoked by default)", 0, targetVertexId)
			graph.AddEdge(graph.SourceVertexId, targetVertexId, edgeName, nil)
		}
		for _, artifact := range artifactsToProcess {
			sourceVertexId, ok := artifact.Configs[graphtypes.GraphSourceVertexKey].(int)
			if !ok {
				logrus.Errorf("the artifact is missing a source vertex id. Actual %+v", artifact)
				continue
			}
			edgeName := fmt.Sprintf("%d -> %d", sourceVertexId, targetVertexId)
			if processVertexId, ok := artifact.Configs[graphtypes.GraphProcessVertexKey].(int); ok {
				sourceVertexId = processVertexId
				edgeName = fmt.Sprintf("%d -> %d", processVertexId, targetVertexId)
			}
			graph.AddEdge(sourceVertexId, targetVertexId, edgeName, map[string]interface{}{"newArtifact": summarizeArtifacts([]transformertypes.Artifact{artifact})})
		}
		for i, newArtifact := range newArtifacts {
			if newArtifact.Configs == nil {
				newArtifact.Configs = map[transformertypes.ConfigType]interface{}{}
			}
			if _, ok := newArtifact.Configs[graphtypes.GraphSourceVertexKey]; ok {
				newArtifact.Configs[graphtypes.GraphProcessVertexKey] = targetVertexId
				continue
			}
			newArtifact.Configs[graphtypes.GraphSourceVertexKey] = targetVertexId
			newArtifacts[i] = newArtifact
		}
	}
	// logging

	if err != nil {
		return newPathMappings, newArtifacts, fmt.Errorf("failed to transform artifacts using the transformer %s . Error: %q", tconfig.Name, err)
	}

	filteredArtifacts := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if ps, ok := tconfig.Spec.ProducedArtifacts[newArtifact.Type]; ok && !ps.Disabled {
			filteredArtifacts = append(filteredArtifacts, newArtifact)
		} else {
			logrus.Warnf("Ignoring artifact %s of type %s in transformer %s", newArtifact.Name, newArtifact.Type, tconfig.Name)
		}
	}
	newArtifacts = filteredArtifacts
	newPathMappings = env.ProcessPathMappings(newPathMappings)
	newPathMappings = *env.DownloadAndDecode(&newPathMappings, true).(*[]transformertypes.PathMapping)
	if err := processPathMappings(newPathMappings, env.Source, env.Output); err != nil {
		return newPathMappings, newArtifacts, fmt.Errorf("failed to process the path mappings: %+v . Error: %q", newPathMappings, err)
	}
	newArtifacts = *env.DownloadAndDecode(&newArtifacts, false).(*[]transformertypes.Artifact)
	newArtifacts = postProcessArtifacts(newArtifacts, tconfig)
	return newPathMappings, newArtifacts, nil
}

func getArtifactsToProcess(newArtifactsToProcess, allArtifacts []transformertypes.Artifact, tConfig transformertypes.Transformer, pt processType) ([]transformertypes.Artifact, []transformertypes.Artifact) {
	artifactsToProcess := []transformertypes.Artifact{}
	artifactsToNotProcess := []transformertypes.Artifact{}

	for _, newArtifact := range newArtifactsToProcess {
		processSpec, ok := tConfig.Spec.ConsumedArtifacts[newArtifact.Type]
		if !ok || processSpec.Disabled {
			artifactsToNotProcess = append(artifactsToNotProcess, newArtifact)
			continue
		}
		switch pt {
		case passthrough:
			if processSpec.Mode != transformertypes.MandatoryPassThrough {
				artifactsToNotProcess = append(artifactsToNotProcess, newArtifact)
				continue
			}
		case dependency:
			if processSpec.Mode != transformertypes.OnDemandPassThrough {
				artifactsToNotProcess = append(artifactsToNotProcess, newArtifact)
				continue
			}
		default:
			if processSpec.Mode == transformertypes.MandatoryPassThrough || processSpec.Mode == transformertypes.OnDemandPassThrough {
				artifactsToNotProcess = append(artifactsToNotProcess, newArtifact)
				continue
			}
		}
		selected := true
		if pt != passthrough && pt != dependency && newArtifact.ProcessWith.String() != "" {
			isSelected, err := selectTransformer(newArtifact.ProcessWith, tConfig)
			if err != nil {
				logrus.Errorf("failed to process the selector %+v against the transformer %+v . Error: %q", newArtifact.ProcessWith, tConfig, err)
				artifactsToNotProcess = append(artifactsToNotProcess, newArtifact)
				continue
			}
			if !isSelected {
				selected = false
			}
		}
		if !selected {
			artifactsToNotProcess = append(artifactsToNotProcess, newArtifact)
			continue
		}
		if processSpec.Merge {
			artifactsToProcess = mergeArtifacts(append(artifactsToProcess, updatedArtifacts(allArtifacts, newArtifact)...))
		}
		artifactsToProcess = append(artifactsToProcess, newArtifact)

	}

	return artifactsToProcess, artifactsToNotProcess
}
