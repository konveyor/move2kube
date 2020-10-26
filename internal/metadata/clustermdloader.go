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

package metadata

import (
	"fmt"

	"github.com/konveyor/move2kube/internal/common"
	clustersmetadata "github.com/konveyor/move2kube/internal/metadata/clusters"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

//go:generate go run github.com/konveyor/move2kube/internal/common/generator clusters makemaps

// ClusterMDLoader Implements Loader interface
type ClusterMDLoader struct {
}

// UpdatePlan - output a plan based on the input directory contents
func (clusterMDLoader *ClusterMDLoader) UpdatePlan(inputPath string, plan *plantypes.Plan) error {
	filePaths, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Warnf("Failed to fetch the cluster metadata yamls at path %q Error: %q", inputPath, err)
		return err
	}
	for _, filePath := range filePaths {
		cm, err := clusterMDLoader.getClusterMetadata(filePath)
		if err != nil {
			continue
		}
		plan.Spec.Inputs.TargetInfoArtifacts[plantypes.K8sClusterArtifactType] = append(plan.Spec.Inputs.TargetInfoArtifacts[plantypes.K8sClusterArtifactType], filePath)

		// If we are targeting the default cluster type then change it to the custom cluster type the user provided.
		if plan.Spec.Outputs.Kubernetes.TargetCluster.Type == common.DefaultClusterType {
			plan.Spec.Outputs.Kubernetes.TargetCluster.Type = cm.Name
		}

		//If there is a cluster-metadata available from collect, then set below flag to true
		plan.Spec.Outputs.Kubernetes.IgnoreUnsupportedKinds = true
	}
	return nil
}

// LoadToIR loads target cluster in IR
func (clusterMDLoader *ClusterMDLoader) LoadToIR(plan plantypes.Plan, ir *irtypes.IR) error {
	clusters := clusterMDLoader.GetClusters(plan)
	target := plan.Spec.Outputs.Kubernetes.TargetCluster
	if target.Type == "" && target.Path == "" {
		log.Warnf("Neither type nor path is specified for the target cluster. Going with the default cluster type: %s", common.DefaultClusterType)
		target.Type = common.DefaultClusterType
	}
	if target.Type != "" && target.Path != "" {
		return fmt.Errorf("Only one of type or path should be specified for the target cluster. Target cluster: %v", target)
	}
	key := target.Type
	if target.Path != "" {
		key = target.Path
	}
	cm, ok := clusters[key]
	if !ok {
		return fmt.Errorf("The requested target cluster %v was not found", target)
	}
	ir.TargetClusterSpec = cm.Spec
	return nil
}

// GetClusters loads list of clusters
func (clusterMDLoader *ClusterMDLoader) GetClusters(plan plantypes.Plan) map[string]collecttypes.ClusterMetadata {
	clusters := map[string]collecttypes.ClusterMetadata{}

	// Load built-in cluster metadata.
	for name, metadata := range clustersmetadata.Constants {
		cm := collecttypes.ClusterMetadata{}
		if err := yaml.Unmarshal([]byte(metadata), &cm); err != nil {
			log.Warnf("Unable to unmarshal built-in cluster info %s Error: %q", name, err)
			continue
		}
		if len(cm.Spec.StorageClasses) == 0 {
			cm.Spec.StorageClasses = []string{common.DefaultStorageClassName}
			log.Debugf("No storage class in the cluster %s, adding [default] storage class", name)
		}
		clusters[cm.Name] = cm
	}

	// Load collected cluster metadata.
	clusterMDPaths := plan.Spec.Inputs.TargetInfoArtifacts[plantypes.K8sClusterArtifactType]
	for _, clusterMDPath := range clusterMDPaths {
		cm, err := clusterMDLoader.getClusterMetadata(clusterMDPath)
		if err != nil {
			log.Errorf("Failed to load the cluster metadata at path %q Error: %q", clusterMDPath, err)
			continue
		}
		if len(cm.Spec.StorageClasses) == 0 {
			cm.Spec.StorageClasses = []string{common.DefaultStorageClassName}
			log.Debugf("No storage class in the cluster %s at path %q, adding [default] storage class", cm.Name, clusterMDPath)
		}
		clusters[cm.Name] = cm
	}

	return clusters
}

func (*ClusterMDLoader) getClusterMetadata(path string) (collecttypes.ClusterMetadata, error) {
	cm := collecttypes.ClusterMetadata{}
	if err := common.ReadYaml(path, &cm); err != nil {
		log.Errorf("Failed to read the cluster metadata at path %q Error: %q", path, err)
		return cm, err
	}
	if cm.Kind != string(collecttypes.ClusterMetadataKind) {
		return cm, fmt.Errorf("The file at path %q is not a valid cluster metadata. Expected kind: %s Actual kind: %s", path, collecttypes.ClusterMetadataKind, cm.Kind)
	}
	return cm, nil
}
