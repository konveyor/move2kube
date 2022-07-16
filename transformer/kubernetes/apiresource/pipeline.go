/*
 *  Copyright IBM Corporation 2021
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

package apiresource

import (
	"fmt"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/sirupsen/logrus"
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

// getSupportedKinds returns the kinds that this type supports.
func (*Pipeline) getSupportedKinds() []string {
	return []string{pipelineKind}
}

// createNewResources creates the runtime objects from the intermediate representation.
func (p *Pipeline) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	// Since tekton is an extension, the tekton resources are put in a separate folder from the main application.
	// We ignore supported kinds because these resources are optional and it's upto the user to install the extension if they need it.
	irresources := ir.TektonResources.Pipelines
	for _, irresource := range irresources {
		objs = append(objs, p.createNewResource(irresource, ir))
	}
	return objs
}

func (*Pipeline) createNewResource(irpipeline irtypes.Pipeline, ir irtypes.EnhancedIR) *v1beta1.Pipeline {
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
	i := 0
	for imageName, container := range ir.ContainerImages {
		if container.Build.ContainerBuildType == "" {
			continue
		}
		i++
		if container.Build.ContainerBuildType == irtypes.DockerfileContainerBuildType {
			_, repoDir, _, gitRepoURL, branchName, err := common.GatherGitInfo(container.Build.ContextPath)
			if err != nil {
				logrus.Debugf("Unable to identify git repo for %s : %s", container.Build.ContextPath, err)
			}
			cloneTaskName := "clone-" + fmt.Sprint(i)
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

			// Assume there is no git repo. If there is no git repo we can't do CI/CD.
			dockerfilePath := dockerfilePathPlaceholder
			contextPath := contextPathPlaceholder
			// If there is a git repo, set the correct context and dockerfile paths.
			if repoDir != "" {
				if len(container.Build.Artifacts) != 0 && len(container.Build.Artifacts[irtypes.DockerfileContainerBuildArtifactTypeValue]) != 0 {
					relDFPath, err := filepath.Rel(repoDir, container.Build.Artifacts[irtypes.DockerfileContainerBuildArtifactTypeValue][0])
					if err != nil {
						logrus.Errorf("Failed to make the df path %q relative to the path %q Error %q", repoDir, container.Build.ContextPath, err)
					} else {
						dockerfilePath = relDFPath
					}
				}
				relContextPath, err := filepath.Rel(repoDir, container.Build.ContextPath)
				if err != nil {
					logrus.Errorf("Failed to make the path %q relative to the path %q Error %q", repoDir, container.Build.ContextPath, err)
				} else {
					if dockerfilePath == dockerfilePathPlaceholder {
						dockerfilePath = filepath.Join(relContextPath, common.DefaultDockerfileName)
					}
					// We can't figure out the context from the source. So assume the context is the directory containing the dockerfile.
					contextPath = relContextPath
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
		} else if container.Build.ContainerBuildType == irtypes.S2IContainerBuildTypeValue {
			// TODO: Implement support for S2I
			logrus.Debugf("S2I not yet supported for Tekton")
		} else if container.Build.ContainerBuildType == irtypes.CNBContainerBuildTypeValue {
			// TODO: Implement support for CNB
			logrus.Debugf("CNB not yet supported for Tekton")
		} else {
			logrus.Errorf("Unknown containerization method: %v", container.Build.ContainerBuildType)
		}
	}
	pipeline.Spec.Tasks = tasks
	return pipeline
}

// convertToClusterSupportedKinds converts the object to supported types if possible.
func (p *Pipeline) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if common.IsPresent(p.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}
