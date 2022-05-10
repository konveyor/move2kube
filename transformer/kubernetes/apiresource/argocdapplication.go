package apiresource

import (
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/konveyor/move2kube/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ArgoCDApplication struct{}

const (
	argoCDNameSpace     = "argocd"
	defaultGitRepoRef   = "HEAD"
	deployToSameCluster = "https://kubernetes.default.svc"
	placeHolderRepoURL  = "<TODO: fill in the git/helm repo url>"
	placeHolderRepoPath = "<TODO: path to deploy/yamls containing manifest yamls>"
)

func (*ArgoCDApplication) getSupportedKinds() []string {
	return []string{v1alpha1.ApplicationSchemaGroupVersionKind.Kind}
}

func (a *ArgoCDApplication) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	// Since ArgoCD is an extension, the ArgoCD resources are put in a separate folder from the main application.
	// We ignore supported kinds because these resources are optional and it's upto the user to install the extension if they need it.
	irresources := ir.ArgoCDResources.Applications
	for _, irresource := range irresources {
		objs = append(objs, a.createNewResource(irresource, targetCluster))
	}
	return objs
}

// createNewResources creates the runtime objects from the intermediate representation.
func (*ArgoCDApplication) createNewResource(irApplication irtypes.Application, targetCluster collecttypes.ClusterMetadata) *v1alpha1.Application {
	repoURL := irApplication.RepoURL
	if repoURL == "" {
		repoURL = placeHolderRepoURL
	}
	repoPath := irApplication.RepoPath
	if repoPath == "" {
		repoPath = placeHolderRepoPath
	}
	repoRef := irApplication.RepoRef
	if repoPath == "" {
		repoRef = defaultGitRepoRef
	}
	clusterServer := irApplication.ClusterServer
	if clusterServer == "" {
		clusterServer = deployToSameCluster
	}
	appGVK := v1alpha1.ApplicationSchemaGroupVersionKind
	return &v1alpha1.Application{
		TypeMeta:   metav1.TypeMeta{APIVersion: appGVK.GroupVersion().String(), Kind: appGVK.Kind},
		ObjectMeta: metav1.ObjectMeta{Name: irApplication.Name, Namespace: argoCDNameSpace},
		Spec: v1alpha1.ApplicationSpec{
			Source: v1alpha1.ApplicationSource{
				RepoURL:        repoURL,
				TargetRevision: repoRef,
				Path:           repoPath,
			},
			Destination: v1alpha1.ApplicationDestination{
				Server:    clusterServer,
				Namespace: irApplication.DestNamespace,
			},
		},
	}
}

// convertToClusterSupportedKinds converts the object to supported types if possible.
func (a *ArgoCDApplication) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, enhancedIR irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if common.IsStringPresent(a.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}
