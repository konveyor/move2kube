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
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	common "github.com/konveyor/move2kube/internal/common"
	clustersmetadata "github.com/konveyor/move2kube/internal/metadata/clusters"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

//go:generate go run github.com/konveyor/move2kube/internal/common/generator clusters makemaps

// ClusterMDLoader Implements Loader interface
type ClusterMDLoader struct {
}

// UpdatePlan - output a plan based on the input directory contents
func (i ClusterMDLoader) UpdatePlan(inputPath string, plan *plantypes.Plan) error {
	files, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Warnf("Unable to fetch yaml files and recognize cluster metadata yamls : %s", err)
	}
	for _, path := range files {
		cm := new(collecttypes.ClusterMetadata)
		if common.ReadYaml(path, &cm) == nil && cm.Kind == string(collecttypes.ClusterMetadataKind) {
			relpath, _ := plan.GetRelativePath(path)
			plan.Spec.Inputs.TargetInfoArtifacts[plantypes.K8sClusterArtifactType] = append(plan.Spec.Inputs.TargetInfoArtifacts[plantypes.K8sClusterArtifactType], relpath)
			if plan.Spec.Outputs.Kubernetes.ClusterType == common.DefaultClusterType {
				plan.Spec.Outputs.Kubernetes.ClusterType = cm.Name
			}

			//If there is a cluster-metadata available from collect, then set below flag to true
			plan.Spec.Outputs.Kubernetes.IgnoreUnsupportedKinds = true
		}
	}
	return nil
}

// LoadToIR loads target cluster in IR
func (i ClusterMDLoader) LoadToIR(p plantypes.Plan, ir *irtypes.IR) error {
	clusters := i.GetClusters(p)
	ir.TargetClusterSpec = clusters[p.Spec.Outputs.Kubernetes.ClusterType].Spec
	return nil
}

// GetClusters loads list of clusters
func (i ClusterMDLoader) GetClusters(p plantypes.Plan) map[string]collecttypes.ClusterMetadata {
	clusters := map[string]collecttypes.ClusterMetadata{}
	for fname, clustermd := range clustersmetadata.Constants {
		cm := collecttypes.ClusterMetadata{}
		err := yaml.Unmarshal([]byte(clustermd), &cm)
		if err != nil {
			log.Warnf("Unable to marshal inbuilt cluster info : %s", fname)
			continue
		}
		if len(cm.Spec.StorageClasses) == 0 {
			cm.Spec.StorageClasses = []string{common.DefaultStorageClassName}
			log.Debugf("No storage class in the cluster, adding [default]")
		}
		clusters[cm.Name] = cm
	}
	for _, clusterfilepath := range p.Spec.Inputs.TargetInfoArtifacts[plantypes.K8sClusterArtifactType] {
		cm := new(collecttypes.ClusterMetadata)
		if common.ReadYaml(p.GetFullPath(clusterfilepath), &cm) == nil && cm.Kind == string(collecttypes.ClusterMetadataKind) {
			if len(cm.Spec.StorageClasses) == 0 {
				cm.Spec.StorageClasses = []string{common.DefaultStorageClassName}
				log.Debugf("No storage class in the cluster, adding [default]")
			}
			clusters[cm.Name] = *cm
			clusters[clusterfilepath] = *cm
		}
	}
	return clusters
}
