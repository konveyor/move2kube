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

package apiresource

import (
	"fmt"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/internal/types/tekton"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	pipelineKind              = "Pipeline"
	defaultGitRepoBranch      = "main"
	gitRepoURLPlaceholder     = "<TODO: insert git repo url>"
	contextPathPlaceholder    = "<TODO: insert path to the directory containing Dockerfile>"
	dockerfilePathPlaceholder = "<TODO: insert path to the Dockerfile>"
)

// Pipeline handles all objects like a Tekton pipeline.
type Pipeline struct {
}

// GetSupportedKinds returns the kinds that this type supports.
func (*Pipeline) GetSupportedKinds() []string {
	return []string{pipelineKind}
}

// CreateNewResources creates the runtime objects from the intermediate representation.
func (p *Pipeline) CreateNewResources(ir irtypes.EnhancedIR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	// Since tekton is an extension, the tekton resources are put in a separate folder from the main application.
	// We ignore supported kinds because these resources are optional and it's upto the user to install the extension if they need it.
	irresources := ir.TektonResources.Pipelines
	for _, irresource := range irresources {
		objs = append(objs, p.createNewResource(irresource, ir))
	}
	return objs
}

func (*Pipeline) createNewResource(irpipeline tekton.Pipeline, ir irtypes.EnhancedIR) *v1beta1.Pipeline {
	pipeline := new(v1beta1.Pipeline)
	pipeline.TypeMeta = metav1.TypeMeta{
		Kind:       pipelineKind,
		APIVersion: v1beta1.SchemeGroupVersion.String(),
	}
	pipeline.ObjectMeta = metav1.ObjectMeta{Name: irpipeline.Name}
	pipeline.Spec.Params = []v1beta1.ParamSpec{
		{Name: "image-registry-url", Description: "registry-domain/namespace where the output image should be pushed.", Type: v1beta1.ParamTypeString},
	}
	pipeline.Spec.Workspaces = []v1beta1.PipelineWorkspaceDeclaration{
		{Name: irpipeline.WorkspaceName, Description: "This workspace will receive the cloned git repo and be passed to the kaniko task for building the image."},
	}
	tasks := []v1beta1.PipelineTask{}
	firstTask := true
	prevTaskName := ""
	for i, container := range ir.Containers {
		if !container.New {
			continue
		}
		if container.ContainerBuildType == plantypes.ManualContainerBuildTypeValue || container.ContainerBuildType == plantypes.ReuseContainerBuildTypeValue {
			log.Debugf("Manual or reuse containerization. We will skip this for CICD.")
			continue
		}
		if container.ContainerBuildType == plantypes.DockerFileContainerBuildTypeValue || container.ContainerBuildType == plantypes.ReuseDockerFileContainerBuildTypeValue {
			cloneTaskName := "clone-" + fmt.Sprint(i)
			gitRepoURL := container.RepoInfo.GitRepoURL
			branchName := container.RepoInfo.GitRepoBranch
			if gitRepoURL == "" {
				gitRepoURL = gitRepoURLPlaceholder
			}
			if branchName == "" {
				branchName = defaultGitRepoBranch
			}

			cloneTask := v1beta1.PipelineTask{
				Name:    cloneTaskName,
				TaskRef: &v1beta1.TaskRef{Name: "git-clone"},
				Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
					{Name: "output", Workspace: irpipeline.WorkspaceName},
				},
				Params: []v1beta1.Param{
					{Name: "url", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: gitRepoURL}},
					{Name: "revision", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: branchName}},
					{Name: "deleteExisting", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "true"}},
				},
			}
			if !firstTask {
				cloneTask.RunAfter = []string{prevTaskName}
			}

			imageName := container.ImageNames[0]
			// Assume there is no git repo. If there is no git repo we can't do CI/CD.
			dockerfilePath := dockerfilePathPlaceholder
			contextPath := contextPathPlaceholder
			// If there is a git repo, set the correct context and dockerfile paths.
			if container.RepoInfo.GitRepoDir != "" {
				relDockerfilePath, err := filepath.Rel(container.RepoInfo.GitRepoDir, container.RepoInfo.TargetPath)
				if err != nil {
					// TODO: Bump up the error after fixing abs path, rel path issues
					log.Debugf("ERROR: Failed to make the path %q relative to the path %q Error %q", container.RepoInfo.GitRepoDir, container.RepoInfo.TargetPath, err)
				} else {
					dockerfilePath = relDockerfilePath
					// We can't figure out the context from the source. So assume the context is the directory containing the dockerfile.
					contextPath = filepath.Dir(relDockerfilePath)
				}
			}

			buildPushTaskName := "build-push-" + fmt.Sprint(i)
			buildPushTask := v1beta1.PipelineTask{
				RunAfter: []string{cloneTaskName},
				Name:     buildPushTaskName,
				TaskRef:  &v1beta1.TaskRef{Name: "kaniko"},
				Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
					{Name: "source", Workspace: irpipeline.WorkspaceName},
				},
				Params: []v1beta1.Param{
					{Name: "IMAGE", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: "$(params.image-registry-url)/" + imageName}},
					{Name: "DOCKERFILE", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: dockerfilePath}},
					{Name: "CONTEXT", Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: contextPath}},
				},
			}
			tasks = append(tasks, cloneTask, buildPushTask)
			firstTask = false
			prevTaskName = buildPushTaskName
		} else if container.ContainerBuildType == plantypes.S2IContainerBuildTypeValue {
			// TODO: Implement support for S2I
			log.Debugf("S2I not yet supported for Tekton")
		} else if container.ContainerBuildType == plantypes.CNBContainerBuildTypeValue {
			// TODO: Implement support for CNB
			log.Debugf("CNB not yet supported for Tekton")
		} else {
			log.Errorf("Unknown containerization method: %v", container.ContainerBuildType)
		}
	}
	pipeline.Spec.Tasks = tasks
	return pipeline
}

// ConvertToClusterSupportedKinds converts the object to supported types if possible.
func (p *Pipeline) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR) ([]runtime.Object, bool) {
	supKinds := p.GetSupportedKinds()
	for _, supKind := range supKinds {
		if common.IsStringPresent(supportedKinds, supKind) {
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}
