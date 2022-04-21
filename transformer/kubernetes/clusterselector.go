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

package kubernetes

import (
	"fmt"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	transformertypes "github.com/konveyor/move2kube/types/transformer"

	"github.com/sirupsen/logrus"
)

const (
	// clusterTypeKey is the key for QA ID
	clusterTypeKey = "clustertype"
	// defaultClusterSelectorSuffix defines the default suffix for the QA IDs
	defaultClusterSelectorSuffix = "cs"
	// defaultStorageClassName defines the default storage class to be used
	defaultStorageClassName = "default"
	// defaultClusterType defines the default cluster type chosen by plan
	defaultClusterType = "Kubernetes"
	// ClusterMetadata config stores cluster configuration of selected cluster
	ClusterMetadata transformertypes.ConfigType = "ClusterMetadata"
)

// ClusterSelectorTransformer implements Transformer interface
type ClusterSelectorTransformer struct {
	Config   transformertypes.Transformer
	Env      *environment.Environment
	Clusters map[string]collecttypes.ClusterMetadata
	CSConfig *ClusterSelectorConfig
}

// ClusterSelectorConfig represents the configuration of the cluster selector
type ClusterSelectorConfig struct {
	QaSuffix string `yaml:"qasuffix"`
}

// Init Initializes the transformer
func (t *ClusterSelectorTransformer) Init(tc transformertypes.Transformer, e *environment.Environment) error {
	t.Config = tc
	t.Env = e
	t.CSConfig = &ClusterSelectorConfig{}
	filePaths, err := common.GetFilesByExt(e.Context, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Failed to fetch the cluster metadata yamls at path %q Error: %q", common.AssetsPath, err)
		return err
	}
	t.Clusters = map[string]collecttypes.ClusterMetadata{}
	for _, filePath := range filePaths {
		cm, err := t.GetClusterMetadata(filePath)
		if err != nil {
			continue
		}
		if _, ok := t.Clusters[cm.Name]; ok {
			logrus.Errorf("Two cluster configs with same name : %s", cm.Name)
			continue
		}
		t.Clusters[cm.Name] = cm
	}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.CSConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.CSConfig, err)
		return err
	}
	if t.CSConfig.QaSuffix == "" {
		t.CSConfig.QaSuffix = defaultClusterSelectorSuffix
	}
	logrus.Errorf("CLUSTER-SELECTOR --> QA SUFFIX = %s", t.CSConfig.QaSuffix)
	return nil
}

// GetConfig returns the transformer config
func (t *ClusterSelectorTransformer) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each subdirectory
func (t *ClusterSelectorTransformer) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms artifacts
func (t *ClusterSelectorTransformer) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	clusterTypeList := []string{}
	for c := range t.Clusters {
		clusterTypeList = append(clusterTypeList, c)
	}
	if len(clusterTypeList) == 0 {
		err = fmt.Errorf("no cluster configuration available")
		logrus.Errorf("%s", err)
		return nil, nil, err
	}
	def := defaultClusterType
	if !common.IsStringPresent(clusterTypeList, def) {
		def = clusterTypeList[0]
	}
	qaId := common.ConfigTargetKey + common.Delim + t.CSConfig.QaSuffix + common.Delim + clusterTypeKey
	clusterType := qaengine.FetchSelectAnswer(qaId, "Choose the cluster type:", []string{"Choose the cluster type you would like to target"}, def, clusterTypeList)
	for ai := range newArtifacts {
		if newArtifacts[ai].Configs == nil {
			newArtifacts[ai].Configs = make(map[transformertypes.ConfigType]interface{})
		}
		newArtifacts[ai].Configs[ClusterMetadata] = t.Clusters[clusterType]
	}
	return nil, newArtifacts, nil
}

// GetClusterMetadata returns the Cluster Metadata
func (t *ClusterSelectorTransformer) GetClusterMetadata(path string) (collecttypes.ClusterMetadata, error) {
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
	if len(cm.Spec.StorageClasses) == 0 {
		cm.Spec.StorageClasses = []string{defaultStorageClassName}
		logrus.Debugf("No storage class in the cluster %s at path %q, adding [default] storage class", cm.Name, path)
	}
	return cm, nil
}
