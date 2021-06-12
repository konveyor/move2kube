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

package classes

/*
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// ReuseDockerfileContainerizer uses its own containerization interface
type ReuseDockerfileContainerizer struct {
}

// GetContainerBuildStrategy returns the containerization build strategy
func (d *ReuseDockerfileContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.ReuseDockerFileContainerBuildTypeValue
}

// GetContainer returns the container for the service
func (d *ReuseDockerfileContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	container := irtypes.NewContainer(d.GetContainerBuildStrategy(), service.Image, true)

	if len(service.ContainerizationOptions) == 0 {
		err := fmt.Errorf("Failed to reuse the Dockerfile. The service %s doesn't have any containerization target options", service.ServiceName)
		logrus.Debug(err)
		return container, err
	}

	dockerfilePath := service.ContainerizationOptions[0]
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) { // TODO: What about other types of errors?
		logrus.Errorf("Unable to find the Dockerfile at path %q Error: %q", dockerfilePath, err)
		logrus.Errorf("Will assume the dockerfile will be copied and will proceed.") // TODO: is this correct? shouldn't we return here?
	}

	dockerfileDir := filepath.Dir(dockerfilePath)
	buildScriptFilename := service.ServiceName + "-docker-build.sh"
	buildScriptPath := filepath.Join(dockerfileDir, buildScriptFilename)

	relContextPath := "."
	if sourceCodeDirs, ok := service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType]; ok {
		if len(sourceCodeDirs) > 0 {
			sourceCodeDir := sourceCodeDirs[0]
			logrus.Debugf("Using %q as the context path for Dockerfile at path %q", sourceCodeDir, dockerfilePath)
			newRelContextPath, err := filepath.Rel(dockerfileDir, sourceCodeDir)
			if err != nil {
				logrus.Errorf("Failed to make the context path %q relative to the directory containing the Dockerfile %q Error: %q", sourceCodeDir, dockerfileDir, err)
				return container, err
			}
			relContextPath = newRelContextPath
		}
	}

	dockerBuildScript, err := common.GetStringFromTemplate(scripts.Dockerbuild_sh, struct {
		Dockerfilename string
		ImageName      string
		Context        string
	}{
		Dockerfilename: filepath.Base(dockerfilePath),
		ImageName:      service.Image,
		Context:        relContextPath,
	})
	if err != nil {
		logrus.Warnf("Unable to translate template to string : %s", scripts.Dockerbuild_sh)
	}

	relBuildScriptPath, err := plan.GetRelativePath(buildScriptPath)
	if err != nil {
		logrus.Errorf("Failed to make the build script path %q relative to the root directory %q Error: %q", buildScriptPath, plan.Spec.Inputs.RootDir, err)
		return container, err
	}

	container.AddFile(relBuildScriptPath, dockerBuildScript)
	return container, nil
}

// DockerfileContainerizer implements Containerizer interface
type DockerfileContainerizer struct {
	dfcontainerizers map[string]containerizer.DockerfileContainerizer
}

// GetContainerBuildStrategy returns the ContaierBuildStrategy
func (*DockerfileContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.DockerFileContainerBuildTypeValue
}

// Init initializes docker file containerizer
func (d *DockerfileContainerizer) Init(path string) {
	if d.dfcontainerizers == nil {
		d.dfcontainerizers = map[string]containerizer.DockerfileContainerizer{}
	}
	filePaths, err := common.GetFilesByExt(path, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Unable to find dockerfile containerizers : %s", err)
	}

	for _, filePath := range filePaths {
		c, err := d.readContainerizerMetadata(filePath)
		if err != nil {
			continue
		}
		c.Spec.FilePath = filePath
		if oc, ok := d.dfcontainerizers[c.Name]; ok {
			logrus.Errorf("Found duplicate containerizer for %s, ignoring %+v", c.Name, oc)
		}
		d.dfcontainerizers[c.Name] = c
	}
	logrus.Debugf("Detected Dockerfile containerization options : %v", d.dfcontainerizers)
}

func (d *DockerfileContainerizer) readContainerizerMetadata(path string) (containerizer.DockerfileContainerizer, error) {
	cm := containerizer.DockerfileContainerizer{}
	if err := common.ReadMove2KubeYaml(path, &cm); err != nil {
		logrus.Debugf("Failed to read the containerizer metadata at path %q Error: %q", path, err)
		return cm, err
	}
	if cm.Kind != string(containerizer.DockerfileContainerizerTypeKind) {
		err := fmt.Errorf("The file at path %q is not a valid containerizer metadata. Expected kind: %s Actual kind: %s", path, containerizer.DockerfileContainerizerTypeKind, cm.Kind)
		logrus.Debug(err)
		return cm, err
	}
	return cm, nil
}

// GetTargetOptions returns the target options for a path
func (d *DockerfileContainerizer) GetTargetOptions(_ plantypes.Plan, path string) []string {
	targetOptions := []string{}
	for _, dfcontainerizer := range d.dfcontainerizers {
		output, err := d.detect(dfcontainerizer, path)
		if err != nil {
			logrus.Debugf("%s detector cannot containerize %s Error: %q", dfcontainerizer, path, err)
			continue
		}
		logrus.Debugf("Output of Dockerfile containerizer detect script %s : %s", dfcontainerizer, output)
		targetOptions = append(targetOptions, dfcontainerizer.Name)
	}
	return targetOptions
}

func (*DockerfileContainerizer) detect(scriptDir string, directory string) (string, error) {
	scriptPath := filepath.Join(scriptDir, dockerfileDetectScript)
	cmd := exec.Command(scriptPath, directory)
	cmd.Dir = scriptDir
	cmd.Stderr = os.Stderr
	logrus.Debugf("Executing detect script %s on %s : %s", scriptDir, directory, cmd)
	outputBytes, err := cmd.Output()
	return string(outputBytes), err
}

// GetContainer returns the container for a service
func (d *DockerfileContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	// TODO: Fix exposed ports too
	if service.ContainerBuildType != d.GetContainerBuildStrategy() || len(service.ContainerizationOptions) == 0 {
		return irtypes.Container{}, fmt.Errorf("Unsupported service type for containerization or insufficient information in service")
	}
	container := irtypes.NewContainer(d.GetContainerBuildStrategy(), service.Image, true)
	container.RepoInfo = service.RepoInfo // TODO: instead of passing this in from plan phase, we should gather git info here itself.
	containerizerDir := service.ContainerizationOptions[0]
	sourceCodeDir := service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0] // TODO: what about the other source artifacts?

	relOutputPath, err := plan.GetRelativePath(sourceCodeDir)
	if err != nil {
		logrus.Errorf("Failed to make the source code directory %q relative to the root directory %q Error: %q", sourceCodeDir, plan.Spec.Inputs.RootDir, err)
		return container, err
	}

	//　1. Execute detect to obtain the json response from the .sh file
	output, err := d.detect(containerizerDir, sourceCodeDir)
	if err != nil {
		logrus.Errorf("Detect using Dockerfile containerizer at path %q on the source code at path %q failed. Error: %q", containerizerDir, sourceCodeDir, err)
		return container, err
	}
	logrus.Debugf("The Dockerfile containerizer at path %q produced the following output: %q", containerizerDir, output)

	// 2. Parse json output
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(output), &m); err != nil {
		logrus.Errorf("Unable to unmarshal the output of the detect script at path %q Output: %q Error: %q", containerizerDir, output, err)
		return container, err
	}

	// Final multiline string containing the generated Dockerfile will be stored here
	dockerfileContents := ""
	// Filled segments will be stored here
	segmentSlice := []string{}

	// 2.1 Obtain segmentRecords slice for both cases (segment-based, df)
	segmentRecords := []map[string]interface{}{}

	if _, ok := m["type"]; ok {
		// segment-based
		if _, segmentsFound := m["segments"]; segmentsFound {
			if segments, ok := m["segments"].([]interface{}); ok {
				for _, segment := range segments {
					if rec, ok := segment.(map[string]interface{}); ok {
						segmentRecords = append(segmentRecords, rec)
					}
				}
			}
		}

	} else {
		// Add the fixed segment id associated to the full Dockerfile and append to segmentRecords
		m["segment_id"] = "Dockerfile"
		segmentRecords = append(segmentRecords, m)
	}

	// 2.2 Iterate over segmentRecords slice
	for _, segmentRecord := range segmentRecords {

		// is "segment_id" present ?
		if val, ok := segmentRecord["segment_id"]; ok {

			strPath, ok := val.(string)

			if !ok {
				logrus.Warnf("Segment id is not a valid string: %v", val)
				continue
			}

			// Get the path to the segment and read it as a template
			dockerfileSegmentTemplatePath := filepath.Join(containerizerDir, strPath)
			dockerfileSegmentTemplateBytes, err := ioutil.ReadFile(dockerfileSegmentTemplatePath)
			if err != nil {
				logrus.Errorf("Unable to read the Dockerfile segment template at path %q Error: %q", dockerfileSegmentTemplatePath, err)
				//return container, err
			}
			dockerfileSegmentTemplate := string(dockerfileSegmentTemplateBytes)
			dockerfileSegmentContents := dockerfileSegmentTemplate

			// Fill the segment template with the corresponding data
			dockerfileSegmentContents, err = common.GetStringFromTemplate(dockerfileSegmentTemplate, segmentRecord)
			if err != nil {
				logrus.Warnf("Skipping segment %s due to error while filling template. Error: %s", dockerfileSegmentTemplatePath, err)
				continue
			}

			// Append filled segment template to segmentSlice
			segmentSlice = append(segmentSlice, dockerfileSegmentContents)
		}

		// is "port" present ?
		if val, ok := segmentRecord["port"]; ok {
			portToExpose := int(val.(float64)) // Type assert to float64 because json numbers are floats.
			container.AddExposedPort(portToExpose)
		}

		// is "files_to_copy" present ?
		if pathsToCopySlice, ok := segmentRecord["files_to_copy"]; ok {

			logrus.Debugf("listing files to copy:")

			// Iterate over the paths
			for _, relPathToCopy := range pathsToCopySlice.([]string) {

				// Generate the absolute path
				pathToCopy := filepath.Join(containerizerDir, relPathToCopy)

				logrus.Debugf("copying file/folder at path %s", pathToCopy)

				// Get path info to determine if it is file/dir
				fileInfo, err := os.Stat(pathToCopy)
				if err != nil {
					logrus.Warnf("Cannot determine if the path %s is a file or folder. Error: %q", pathToCopy, err)
					continue
				}

				if fileInfo.IsDir() {
					logrus.Debugf("The path %s is a directory", pathToCopy)

					err := filepath.Walk(pathToCopy, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							logrus.Warnf("Skipping path %s due to error. Error: %q", path, err)
							return nil
						}

						if info.IsDir() {
							return nil
						}
						// At this point it means we have a file
						logrus.Debugf("copying the file %s", path)

						// Obtain the relative path of the file wrt the containerizerDir
						relFilePath, err := filepath.Rel(containerizerDir, path)
						if err != nil {
							logrus.Warnf("Failed to make the path %s relative to the containerizer directory %s Error: %q", path, containerizerDir, err)
							return nil
						}
						logrus.Debugf("relative file path is %s", relFilePath)

						// Get file contents
						contentBytes, err := ioutil.ReadFile(path)
						if err != nil {
							logrus.Warnf("Failed to read the file at path %s Error: %q", path, err)
							return nil
						}

						// Add file contents and relative path to the container object
						container.AddFile(filepath.Join(relOutputPath, relFilePath), string(contentBytes))
						return nil
					})
					if err != nil {
						logrus.Warnf("Error in walking through files at path %q Error: %q", containerizerDir, err)
						continue
					}

				} else {
					logrus.Debugf("The path %s is a file", pathToCopy)
					// Get content and add it to the container object
					contentBytes, err := ioutil.ReadFile(pathToCopy)
					if err != nil {
						logrus.Warnf("Failed to read the file at path %s Error: %q", pathToCopy, err)
						continue
					}
					container.AddFile(filepath.Join(relOutputPath, relPathToCopy), string(contentBytes))
				}
			}
		}
	}
	// 3. Merge the filled segments into dockerfileContents
	dockerfileContents = strings.Join(segmentSlice, "\n")

	// 4. Add result to the container object
	dockerfileName := "Dockerfile." + service.ServiceName
	dockerfilePath := filepath.Join(relOutputPath, dockerfileName)
	container.AddFile(dockerfilePath, dockerfileContents)

	// 5. Create the docker build script.
	dockerBuildScriptContents, err := common.GetStringFromTemplate(scripts.Dockerbuild_sh, struct {
		Dockerfilename string
		ImageName      string
		Context        string
	}{
		Dockerfilename: dockerfileName,
		ImageName:      service.Image,
		Context:        ".",
	})
	if err != nil {
		logrus.Errorf("Failed to fill the docker build script template %s Error: %q", scripts.Dockerbuild_sh, err)
		return container, err
	}

	dockerBuildScriptPath := filepath.Join(relOutputPath, service.ServiceName+"-docker-build.sh")
	container.AddFile(dockerBuildScriptPath, dockerBuildScriptContents)
	container.RepoInfo.TargetPath, err = plan.GetAbsolutePath(dockerfilePath)
	if err != nil {
		logrus.Errorf("Failed to make the relative dockerfile path %s absolute using the plan's root directory. Error: %q", dockerfilePath, err)
		return container, err
	}

	return container, nil
}

// GetServiceOptions - output a plan based on the input directory contents
func (dockerfileTranslator *DockerfileTranslator) GetServiceOptions(inputPath string, plan plantypes.Plan) (map[string]plantypes.Service, error) {
	services := map[string]plantypes.Service{}
	sdfs, err := getDockerfileServices(inputPath, plan.Name)
	if err != nil {
		logrus.Errorf("Unable to get Dockerfiles : %s", err)
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
				logrus.Warnf("Error while parsing the git repo at path %q Error: %q", dfs[0].path, err)
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
		logrus.Debugf("The service %s has no containerization target options. Skipping.", serviceName)
		continue
	}
	logrus.Debugf("Translating %s", serviceName)
	irContainer, err := new(containerizer.ReuseDockerfileContainerizer).GetContainer(plan, service)
	if err != nil {
		logrus.Warnf("Unable to get reuse the Dockerfile for service %s even though build parameters are present. Error: %q", serviceName, err)
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
		logrus.Debugf("Unable to open file %s : %s", path, err)
		return false, err
	}
	defer f.Close()
	res, err := dockerparser.Parse(f)
	if err != nil {
		logrus.Debugf("Unable to parse file %s as Docker files : %s", path, err)
		return false, err
	}
	for _, dfchild := range res.AST.Children {
		if dfchild.Value == "from" {
			r := regexp.MustCompile(`(?i)FROM\s+(--platform=[^\s]+)?[^\s]+(\s+AS\s+[^\s]+)?\s*(#.+)?$`)
			if r.MatchString(dfchild.Original) {
				logrus.Debugf("Identified a docker file : " + path)
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
		logrus.Warnf("Error in walking through files due to : %s", err)
		return sDockerfiles, err
	} else if !info.IsDir() {
		logrus.Warnf("The path %q is not a directory.", inputpath)
	}
	files := []string{}
	err = filepath.Walk(inputpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("Skipping path %s due to error: %s", path, err)
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
		logrus.Warnf("Error in walking through files due to : %s", err)
	}
	logrus.Debugf("No of dockerfiles identified : %d", len(files))
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
			logrus.Errorf("There is no file at path %s Error: %q", path, err)
			return false
		}
		logrus.Errorf("There was an error accessing the file at path %s Error: %q", path, err)
		return false
	}
	if finfo.IsDir() {
		logrus.Errorf("The path %s points to a directory. Expected a Dockerfile.", path)
		return false
	}
	return true
}
*/
