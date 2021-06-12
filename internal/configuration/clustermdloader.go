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

package configuration

import (
	"fmt"

	"github.com/konveyor/move2kube/internal/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
)

// ClusterMDLoader Implements Loader interface
type ClusterMDLoader struct {
}

// UpdatePlan - output a plan based on the input directory contents
func (clusterMDLoader *ClusterMDLoader) UpdatePlan(plan *plantypes.Plan) error {
	filePaths, err := common.GetFilesByExt(common.AssetsPath, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Failed to fetch the cluster metadata yamls at path %q Error: %q", common.AssetsPath, err)
		return err
	}
	for _, filePath := range filePaths {
		cm, err := clusterMDLoader.GetClusterMetadata(filePath)
		if err != nil {
			continue
		}
		plan.Spec.Configuration.TargetClusters[cm.Name] = filePath

		// If we are targeting the default cluster type then change it to the custom cluster type the user provided.
		if plan.Spec.TargetCluster.Type == common.DefaultClusterType {
			plan.Spec.TargetCluster.Type = cm.Name
		}
	}
	return nil
}

// GetClusters loads list of clusters
func (clusterMDLoader *ClusterMDLoader) GetClusters(plan plantypes.Plan) map[string]collecttypes.ClusterMetadata {
	clusters := map[string]collecttypes.ClusterMetadata{}

	// Load collected cluster metadata.
	clusterMDPaths := plan.Spec.Configuration.TargetClusters
	for _, clusterMDPath := range clusterMDPaths {
		cm, err := clusterMDLoader.GetClusterMetadata(clusterMDPath)
		if err != nil {
			logrus.Errorf("Failed to load the cluster metadata at path %q Error: %q", clusterMDPath, err)
			continue
		}
		if len(cm.Spec.StorageClasses) == 0 {
			cm.Spec.StorageClasses = []string{common.DefaultStorageClassName}
			logrus.Debugf("No storage class in the cluster %s at path %q, adding [default] storage class", cm.Name, clusterMDPath)
		}
		clusters[cm.Name] = cm
	}

	return clusters
}

func (*ClusterMDLoader) GetClusterMetadata(path string) (collecttypes.ClusterMetadata, error) {
	cm := collecttypes.ClusterMetadata{}
	if err := common.ReadMove2KubeYaml(path, &cm); err != nil {
		logrus.Debugf("Failed to read the cluster metadata at path %q Error: %q", path, err)
		return cm, err
	}
	if cm.Kind != string(collecttypes.ClusterMetadataKind) {
		err := fmt.Errorf("the file at path %q is not a valid cluster metadata. Expected kind: %s Actual kind: %s", path, collecttypes.ClusterMetadataKind, cm.Kind)
		logrus.Debug(err)
		return cm, err
	}
	return cm, nil
}

func (c *ClusterMDLoader) GetTargetClusterMetadataForPlan(plan plantypes.Plan) (targetCluster collecttypes.ClusterMetadata, err error) {
	if plan.Spec.TargetCluster.Path != "" {
		targetCluster, err = c.GetClusterMetadata(plan.Spec.TargetCluster.Path)
		if err != nil {
			logrus.Errorf("Unable to load cluster metadata from %s : %s", plan.Spec.TargetCluster.Path, err)
			return targetCluster, err
		}
	} else if plan.Spec.TargetCluster.Type != "" {
		var ok bool
		targetCluster, ok = c.GetClusters(plan)[plan.Spec.TargetCluster.Type]
		if !ok {
			err = fmt.Errorf("unable to load cluster metadata from %s", plan.Spec.TargetCluster.Type)
			logrus.Errorf("%s", err)
			return targetCluster, err
		}
	} else {
		err := fmt.Errorf("unable to find target cluster : %+v", plan.Spec.TargetCluster)
		logrus.Errorf("%s", err)
		return targetCluster, err
	}
	return targetCluster, nil
}
