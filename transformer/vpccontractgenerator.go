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

package transformer

import (
	"fmt"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/spf13/cast"
)

// VpcContractGenerator implements Transformer interface
type VpcContractGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// ContractTemplate to store vpc contract
type ContractTemplate struct {
	EnvType         string
	LogHostName     string
	IngestionKey    string
	LogPort         string
	EnvVolumes      map[string]map[string]string
	WorkloadType    string
	Auths           map[string]map[string]string
	ComposeArchive  string
	Images          map[string]map[string]string
	WorkloadVolumes map[string]map[string]string
	WorkloadEnvs    map[string]string
}

// Init initializes the translator
func (t *VpcContractGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the config of the transformer
func (t *VpcContractGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect executes detect in directories respecting the m2kignore
func (t *VpcContractGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *VpcContractGenerator) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	usesVPC := qaengine.FetchSelectAnswer(common.BaseKey+".ibmvpc", "Do you use IBM VPC?", []string{"We will generate contract file as part of the transformation."}, "No", []string{"Yes", "No"})

	if usesVPC == "No" {
		return nil, nil, nil
	}

	confTypes := qaengine.FetchMultiSelectAnswer(common.BaseKey+".ibmvpc.types", "What are the types?", nil, []string{}, []string{"workload", "env"})

	if len(confTypes) == 0 {
		return nil, nil, nil
	}

	data := ContractTemplate{}

	for _, confType := range confTypes {
		if confType == "env" {
			logHostName := qaengine.FetchStringAnswer(common.BaseKey+".ibmvpc.env.loghostname", "What is the log DNA hostname?", nil, "")
			ingestionKey := qaengine.FetchStringAnswer(common.BaseKey+".ibmvpc.env.ingestionkey", "What is the ingestion key?", nil, "")
			logPortStr := qaengine.FetchStringAnswer(common.BaseKey+".ibmvpc.env.logport", "What is the log port?", nil, "8080")
			volumeCountStr := qaengine.FetchStringAnswer(common.BaseKey+".ibmvpc.env.volumecount", "How many volumes?", nil, "0")
			volumeCount, _ := cast.ToIntE(volumeCountStr)

			volumes := map[string]map[string]string{}

			for i := 0; i < volumeCount; i++ {
				volumeName := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.env.volumes[%d].name", i), "What is volume name?", nil, "")
				volumeSeed := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.env.volumes[%d].seed", i), "What is volume seed?", nil, "")
				volumeMount := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.env.volumes[%d].mount", i), "What is volume mount?", nil, "")
				volumeFS := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.env.volumes[%d].fs", i), "What is volume filesystem?", nil, "")
				volume := map[string]string{
					"mount":      volumeMount,
					"seed":       volumeSeed,
					"filesystem": volumeFS,
				}
				volumes[volumeName] = volume
			}
			data.EnvType = confType
			data.LogHostName = logHostName
			data.IngestionKey = ingestionKey
			data.LogPort = logPortStr
			data.EnvVolumes = volumes

		} else if confType == "workload" {
			authsCountStr := qaengine.FetchStringAnswer(common.BaseKey+".ibmvpc.workload.authscount", "How many services?", nil, "0")
			authsCount, _ := cast.ToIntE(authsCountStr)
			auths := map[string]map[string]string{}

			for i := 0; i < authsCount; i++ {
				serviceAdress := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.workload.service[%d].address", i), "What is service address?", nil, "")
				serviceUserName := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.env.service[%d].username", i), "What is username?", nil, "")
				servicePass := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.env.service[%d].pass", i), "What is password?", nil, "")
				auth := map[string]string{
					"username": serviceUserName,
					"password": servicePass,
				}
				auths[serviceAdress] = auth
			}

			composeArchive := qaengine.FetchStringAnswer(common.BaseKey+".ibmvpc.workload.compose.archive", "What is the compose archive?", nil, "")

			imagesCountStr := qaengine.FetchStringAnswer(common.BaseKey+".ibmvpc.workload.imagescount", "How many images?", nil, "0")
			imagesCount, _ := cast.ToIntE(imagesCountStr)
			images := map[string]map[string]string{}

			for i := 0; i < imagesCount; i++ {
				registryAdress := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.workload.registry[%d].address", i), "What is registry address?", nil, "")
				registryNotary := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.env.registry[%d].notary", i), "What is notary?", nil, "")
				registryPublicKey := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.env.registry[%d].publickey", i), "What is publickey?", nil, "")
				image := map[string]string{
					"notary":    registryNotary,
					"publicKey": registryPublicKey,
				}
				images[registryAdress] = image
			}

			workloadVolumeCountStr := qaengine.FetchStringAnswer(common.BaseKey+".ibmvpc.workload.volumecount", "How many volumes for workload?", nil, "0")
			workloadVolumeCount, _ := cast.ToIntE(workloadVolumeCountStr)

			workloadVolumes := map[string]map[string]string{}

			for i := 0; i < workloadVolumeCount; i++ {
				volumeName := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.workload.volumes[%d].name", i), "What is volume name?", nil, "")
				volumeSeed := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.workload.volumes[%d].seed", i), "What is volume seed?", nil, "")
				volumeMount := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.workload.volumes[%d].mount", i), "What is volume mount?", nil, "")
				volumeFS := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.workload.volumes[%d].fs", i), "What is volume filesystem?", nil, "")
				volume := map[string]string{
					"mount":      volumeMount,
					"seed":       volumeSeed,
					"filesystem": volumeFS,
				}
				workloadVolumes[volumeName] = volume
			}

			workloadEnvsCountStr := qaengine.FetchStringAnswer(common.BaseKey+".ibmvpc.workload.envcount", "How many envs?", nil, "0")
			workloadEnvsCount, _ := cast.ToIntE(workloadEnvsCountStr)

			workloadEnvs := map[string]string{}

			for i := 0; i < workloadEnvsCount; i++ {
				envKey := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.workload.envs[%d].key", i), "What is env key?", nil, "")
				envValue := qaengine.FetchStringAnswer(common.BaseKey+fmt.Sprintf(".ibmvpc.workload.envs[%d].value", i), "What is env value?", nil, "")
				workloadEnvs[envKey] = envValue
			}

			data.WorkloadType = confType
			data.Auths = auths
			data.ComposeArchive = composeArchive
			data.Images = images
			data.WorkloadVolumes = workloadVolumes
			data.WorkloadEnvs = workloadEnvs
		}

	}

	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
		TemplateConfig: data,
	})
	return pathMappings, nil, nil
}
