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

package collector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/transformer/kubernetes/k8sschema"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cgdiscovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // See issue https://github.com/kubernetes/client-go/issues/345
	cgclientcmd "k8s.io/client-go/tools/clientcmd"
)

//ClusterCollector Implements Collector interface
type ClusterCollector struct {
	clusterCmd string
}

// GetAnnotations returns annotations on which this collector should be invoked
func (c ClusterCollector) GetAnnotations() []string {
	return []string{"k8s"}
}

//Collect gets the cluster metadata by querying the cluster. Assumes that the authentication with cluster is already done.
func (c *ClusterCollector) Collect(inputPath string, outputPath string) error {
	//Creating the output sub-directory if it does not exist
	outputPath = filepath.Join(outputPath, "clusters")
	err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission)
	if err != nil {
		logrus.Errorf("Unable to create output directory at path %q Error: %q", outputPath, err)
		return err
	}
	cmd := c.getClusterCommand()
	if cmd == "" {
		errStr := "no kubectl or oc in path. Add kubectl to path and rerun to collect data about the cluster in context"
		logrus.Warnf(errStr)
		return fmt.Errorf(errStr)
	}
	name, err := c.getClusterContextName()
	if err != nil {
		logrus.Warnf("Unable to access cluster in context : %s", err)
		return err
	}
	clusterMd := collecttypes.NewClusterMetadata(name)
	if clusterMd.Spec.StorageClasses, err = c.getStorageClasses(); err != nil {
		//If no storage classes, this will be an empty array
		clusterMd.Spec.StorageClasses = []string{}
	}

	clusterMd.Spec.APIKindVersionMap, err = c.collectUsingAPI()
	if err != nil {
		logrus.Warnf("Failed to collect using the API. Error: %q . Falling back to using the CLI.", err)
		clusterMd.Spec.APIKindVersionMap, err = c.collectUsingCLI()
		if err != nil {
			logrus.Warnf("Failed to collect using the CLI. Error: %q", err)
			return err
		}
	}

	c.groupOrderPolicy(&clusterMd.Spec.APIKindVersionMap)
	//c.VersionOrderPolicy(&clusterMd.APIKindVersionMap)

	outputPath = filepath.Join(outputPath, common.NormalizeForFilename(clusterMd.Name)+".yaml")
	return common.WriteYaml(outputPath, clusterMd)
}

func (c *ClusterCollector) getClusterCommand() string {
	if c.clusterCmd != "" {
		return c.clusterCmd
	}

	cmd := "kubectl"
	_, err := exec.LookPath(cmd)
	if err == nil {
		c.clusterCmd = cmd
		return c.clusterCmd
	}
	logrus.Warnf("Unable to find the %s command. Error: %q", cmd, err)

	cmd = "oc"
	_, err = exec.LookPath(cmd)
	if err == nil {
		c.clusterCmd = cmd
		return c.clusterCmd
	}
	logrus.Warnf("Unable to find the %s command. Error: %q", cmd, err)

	return ""
}

func (c *ClusterCollector) getClusterContextName() (string, error) {
	cmd := exec.Command(c.getClusterCommand(), "config", "current-context")
	name, err := cmd.Output()
	return strings.TrimSpace(string(name)), err
}

func (c *ClusterCollector) getStorageClasses() ([]string, error) {
	ccmd := c.getClusterCommand()
	cmd := exec.Command(ccmd, "get", "sc", "-o", "yaml")
	yamlOutput, err := cmd.CombinedOutput()
	if err != nil {
		errDesc := c.interpretError(string(yamlOutput))
		if errDesc != "" {
			logrus.Warnf("Error while running %s. %s", ccmd, errDesc)
		} else {
			logrus.Warnf("Error while fetching storage classes using command [%s]", cmd)
		}
		return nil, err
	}

	fileContents := map[string]interface{}{}
	err = yaml.Unmarshal(yamlOutput, &fileContents)
	if err != nil {
		logrus.Errorf("Error in unmarshalling yaml: %s. Skipping.", err)
		return nil, err
	}

	scArray := fileContents["items"].([]interface{})
	storageClasses := []string{}

	for _, sc := range scArray {
		if mapSC, ok := sc.(map[string]interface{}); ok {
			storageClasses = append(storageClasses, mapSC["metadata"].(map[string]interface{})["name"].(string))
		} else {
			logrus.Warnf("Unknown type detected in cluster metadata [%T]", mapSC)
		}
	}

	return storageClasses, nil
}

func (c *ClusterCollector) interpretError(cmdOutput string) string {
	errorTerms := []string{"Unauthorized", "Username"}

	for _, e := range errorTerms {
		if c.getClusterCommand() == "oc" && strings.Contains(cmdOutput, e) {
			return "Please login to cluster before running collect. (e.g. oc login <cluster url> --token=<token string>)"
		} else if c.getClusterCommand() == "kubectl" && strings.Contains(cmdOutput, e) {
			return "Please configure the cluster authentication with following instructions: [https://kubernetes.io/docs/reference/kubectl/cheatsheet/#kubectl-context-and-configuration]"
		}
	}

	return ""
}

func (c ClusterCollector) getGlobalGroupOrder() []string {
	return []string{`^.+\.k8s\.io$`, `^apps$`, `^policy$`, `^extensions$`, `^.+\.openshift\.io$`}
}

func (c *ClusterCollector) getAPI() (*cgdiscovery.DiscoveryClient, error) {
	rules := cgclientcmd.NewDefaultClientConfigLoadingRules()
	cfg, err := cgclientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &cgclientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		logrus.Warnf("Failed to get the default config for the cluster API client. Error: %q", err)
		return nil, err
	}
	return cgdiscovery.NewDiscoveryClientForConfig(cfg)
}

func (c *ClusterCollector) getPreferredResourceUsingAPI(api *cgdiscovery.DiscoveryClient) ([]schema.GroupVersion, error) {
	defer func() []schema.GroupVersion {
		if rErr := recover(); rErr != nil {
			logrus.Errorf("Recovered from error in getPreferredResourceUsingAPI [%s]", rErr)
			return nil
		}
		return []schema.GroupVersion{}
	}()
	debug.SetPanicOnFault(true)
	var gvList []schema.GroupVersion
	if api == nil {
		logrus.Errorf("API object is null")
		return nil, fmt.Errorf("API object is null")
	}
	apiGroupList, err := api.ServerGroups()
	if err != nil {
		logrus.Errorf("API request for server-group list failed")
		return nil, err
	}
	for _, group := range apiGroupList.Groups {
		preferredGV, err := schema.ParseGroupVersion(group.PreferredVersion.GroupVersion)
		if err != nil {
			continue
		}
		gvList = append(gvList, preferredGV)
		prioritizedGVList := scheme.Scheme.PrioritizedVersionsForGroup(group.Name)
		for _, prioritizedGV := range prioritizedGVList {
			if strings.Compare(group.PreferredVersion.GroupVersion, prioritizedGV.String()) == 0 {
				continue
			}
			for _, gvObj := range group.Versions {
				if strings.Compare(prioritizedGV.String(), gvObj.GroupVersion) == 0 {
					gv, _ := schema.ParseGroupVersion(gvObj.GroupVersion)
					gvList = append(gvList, gv)
					break
				}
			}
		}
		for _, version := range group.Versions {
			gv, _ := schema.ParseGroupVersion(version.GroupVersion)
			if gvExists(gvList, gv) {
				continue
			}
			gvList = append(gvList, gv)
		}
	}
	return gvList, nil
}

func (c *ClusterCollector) getKindsForGroups(api *cgdiscovery.DiscoveryClient) (map[string][]schema.GroupVersion, error) {
	defer func() map[string][]schema.GroupVersion {
		if rErr := recover(); rErr != nil {
			logrus.Errorf("Recovered from error in getKindsForGroups [%s]", rErr)
			return nil
		}
		var emptyMap map[string][]schema.GroupVersion
		return emptyMap
	}()

	mapKind := map[string][]schema.GroupVersion{}

	_, apiResourceList, err := api.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	for _, rscListObj := range apiResourceList {
		gvObj, err := schema.ParseGroupVersion(rscListObj.GroupVersion)
		for err != nil {
			logrus.Warnf("Ignoring group-version [%s]. Could not parse it", rscListObj.GroupVersion)
			continue
		}

		for _, rscObj := range rscListObj.APIResources {
			if gvList, ok := mapKind[rscObj.Kind]; ok {
				if !gvExists(gvList, gvObj) {
					gvList = append(gvList, gvObj)
				}
				mapKind[rscObj.Kind] = gvList
			} else {
				mapKind[rscObj.Kind] = []schema.GroupVersion{gvObj}

			}
		}
	}

	return mapKind, nil
}

func (c *ClusterCollector) sortGroupVersionByPreference(prefGVList []schema.GroupVersion, mapKind *map[string][]schema.GroupVersion) {
	for kind, gvList := range *mapKind {
		var gvOrderedList []schema.GroupVersion
		unorderedList := []string{}
		for _, pGV := range prefGVList {
			if gvExists(gvList, pGV) {
				gvOrderedList = append(gvOrderedList, pGV)
			} else {
				unorderedList = append(unorderedList, pGV.String())
			}
		}

		unorderedList = c.clusterByGroupsAndSortVersions(unorderedList)
		for _, gvStr := range unorderedList {
			gvObj, err := schema.ParseGroupVersion(gvStr)
			if err == nil {
				continue
			}
			gvOrderedList = append(gvOrderedList, gvObj)
		}
		(*mapKind)[kind] = gvOrderedList
	}
}

func (c *ClusterCollector) collectUsingAPI() (map[string][]string, error) {
	api, err := c.getAPI()
	if err != nil {
		logrus.Warnf("Failed to api handle for cluster")
		return nil, err
	}

	gvList, err := c.getPreferredResourceUsingAPI(api)
	errStr := "Failed to retrieve preferred group information from cluster"
	if err != nil {
		logrus.Warnf(errStr)
		return nil, err
	} else if len(gvList) == 0 {
		logrus.Warnf(errStr)
		return nil, fmt.Errorf(errStr)
	}

	mapKind, err := c.getKindsForGroups(api)
	errStr = "Failed to retrieve <kind, group-version> information from cluster"
	if err != nil {
		logrus.Warnf(errStr)
		return nil, err
	} else if len(mapKind) == 0 {
		logrus.Warnf(errStr)
		return nil, fmt.Errorf(errStr)
	}

	c.sortGroupVersionByPreference(gvList, &mapKind)

	APIKindVersionMap := map[string][]string{}

	for kind, gvList := range mapKind {
		gvStrList := make([]string, len(gvList))
		for i, gv := range gvList {
			gvStrList[i] = gv.String()
		}
		APIKindVersionMap[kind] = gvStrList
	}

	return APIKindVersionMap, nil
}

func (c *ClusterCollector) getAllGVMatchingGroup(groupRegex string, gvList []string) []string {
	var filtered []string

	for _, gv := range gvList {
		gvObj, err := schema.ParseGroupVersion(gv)
		if err != nil {
			continue
		}

		if gvObj.Group == "" {
			continue
		}

		re := regexp.MustCompile(groupRegex)
		if re.MatchString(gvObj.Group) {
			filtered = append(filtered, gv)
		}
	}

	return filtered
}

func (c *ClusterCollector) groupOrderPolicy(mapKindGV *map[string][]string) {
	globalOrder := c.getGlobalGroupOrder()
	for kind, gvList := range *mapKindGV {
		sortedGV := []string{}

		//First priority is for known groups
		for _, groupKey := range globalOrder {
			subsetOfGV := c.getAllGVMatchingGroup(groupKey, gvList)
			sortedGV = append(sortedGV, subsetOfGV...)
		}

		//Second priority is for unknown groups (which are not empty string)
		for _, gv := range gvList {
			gvObj, err := schema.ParseGroupVersion(gv)
			if err != nil {
				continue
			}

			if common.IsPresent(sortedGV, gv) {
				continue
			}

			if strings.Compare(gvObj.Group, "") != 0 {
				sortedGV = append(sortedGV, gv)
			}
		}

		//Third priority is for empty groups
		for _, gv := range gvList {
			gvObj, err := schema.ParseGroupVersion(gv)
			if err != nil {
				continue
			}

			if strings.Compare(gvObj.Group, "") == 0 {
				sortedGV = append(sortedGV, gv)
			}
		}

		if len(sortedGV) > 0 {
			(*mapKindGV)[kind] = sortedGV
		} else {
			(*mapKindGV)[kind] = gvList
		}
	}
}

func (c *ClusterCollector) sortVersionList(vList *[]string) {
	srcVersionKeys := []string{"alpha", "beta"}
	trVersionKeys := []string{"-alpha.", "-beta."}
	regex := []string{`\-alpha\.`, `\-beta\.`}

	for index, version := range *vList {
		for i, vKey := range srcVersionKeys {
			re := regexp.MustCompile(vKey)
			if re.MatchString(version) {
				//Tranforming the string to the format suitable for semver pkg
				(*vList)[index] = re.ReplaceAllString(version, trVersionKeys[i])
				break
			}
		}
	}

	svObjList := make([]*semver.Version, len(*vList))
	for index, versionStr := range *vList {
		svObj, err := semver.NewVersion(versionStr)
		if err != nil {
			logrus.Warnf("Skipping Version: %s", versionStr)
			continue
		}

		svObjList[index] = svObj
	}

	sort.Sort(sort.Reverse(semver.Collection(svObjList)))

	for index, svObj := range svObjList {
		transfVersionStr := svObj.Original()
		noMatches := true
		for i, vKey := range regex {
			re := regexp.MustCompile(vKey)
			if re.MatchString(transfVersionStr) {
				(*vList)[index] = re.ReplaceAllString(transfVersionStr, srcVersionKeys[i])
				noMatches = false
				break
			}
		}
		if noMatches {
			(*vList)[index] = transfVersionStr
		}
	}
}

func (c *ClusterCollector) clusterByGroupsAndSortVersions(gvList []string) []string {
	gvMap := map[string][]string{}
	for _, gvStr := range gvList {
		gvObj, err := schema.ParseGroupVersion(gvStr)
		if err != nil {
			logrus.Debugf("Error parting group version [%s]", gvStr)
			continue
		}

		if vList, ok := gvMap[gvObj.Group]; ok {
			vList = append(vList, gvObj.Version)
			gvMap[gvObj.Group] = vList
		} else {
			vList = []string{gvObj.Version}
			gvMap[gvObj.Group] = vList
		}
	}

	for _, vList := range gvMap {
		c.sortVersionList(&vList)
	}

	sortedGVList := []string{}
	for group, vList := range gvMap {
		for _, v := range vList {
			gvObj := schema.GroupVersion{Group: group, Version: v}
			sortedGVList = append(sortedGVList, gvObj.String())
		}
	}

	return sortedGVList
}

func (c *ClusterCollector) collectUsingCLI() (map[string][]string, error) {
	cmd := exec.Command("bash", "-c", c.getClusterCommand()+" api-resources -o name")
	output, err := cmd.Output()
	if err != nil {
		logrus.Errorf("Error while running kubectl api-resources: %s", err)
		return nil, err
	}
	logrus.Debugf("Got kind information for cluster")
	nameList := strings.Split(string(output), "\n")
	mapKind := map[string][]schema.GroupVersion{}
	for _, name := range nameList {
		tmpArray := strings.Split(name, ".")
		if len(tmpArray) > 0 {
			name := tmpArray[0]
			kind, gvStr, err := c.getGVKUsingNameCLI(name)
			if err != nil {
				logrus.Debugf("Erroring parsing kind from CLI output")
				continue
			}
			group := ""
			for i, tmp := range tmpArray {
				if i == 1 {
					group = tmp
				} else if i > 1 {
					tmp = strings.TrimSpace(tmp)
					group = group + "." + tmp
				}
			}

			if group != "" {
				if groupArray, ok := mapKind[kind]; ok {
					groupArray = append(groupArray, schema.GroupVersion{Group: group, Version: ""})
					mapKind[kind] = groupArray
				} else {
					mapKind[kind] = []schema.GroupVersion{{Group: group, Version: ""}}
				}
			} else {
				mapKind[kind] = []schema.GroupVersion{{Group: "", Version: gvStr}}
			}
		}
	}

	apiMd := map[string][]string{}

	for kind, availableGroupList := range mapKind {
		if len(availableGroupList) == 1 {
			singletonObj := availableGroupList[0]
			if strings.Compare(singletonObj.Group, "") == 0 {
				apiMd[kind] = []string{singletonObj.Version}
				continue
			}
		}
		if len(availableGroupList) > 0 {
			gvList := c.getPreferredGVUsingCLI(kind, availableGroupList)
			apiMd[kind] = gvList
		} else {
			logrus.Warnf("Empty group for kind [%s]", kind)
		}
	}

	return apiMd, nil
}

func (c *ClusterCollector) getPreferredGVUsingCLI(kind string, availableGroupList []schema.GroupVersion) []string {
	scheme := k8sschema.GetSchema()
	var gvList []string
	for _, gvObj := range availableGroupList {
		prioritizedGVList := scheme.PrioritizedVersionsForGroup(gvObj.Group)
		if len(prioritizedGVList) > 0 {
			for _, gv := range prioritizedGVList {
				isSupported, err := c.isSupportedGV(kind, gv.String())
				if isSupported {
					gvList = append(gvList, gv.String())
				} else {
					logrus.Debugf("Group version not found by CLI for kind [%s] : %s", kind, err)
				}
			}
		} else {
			_, gvStr, err := c.getGVKUsingNameCLI(kind)
			if err == nil {
				gvList = append(gvList, gvStr)
			}
		}
	}

	return gvList
}

func (c *ClusterCollector) isSupportedGV(kind string, gvStr string) (bool, error) {
	cmd := exec.Command("bash", "-c", c.getClusterCommand()+" explain "+kind+" --api-version="+gvStr+" --recursive")
	output, err := cmd.Output()
	if err != nil {
		logrus.Debugf("Error while running %s for verifying [%s]\n", c.getClusterCommand(), gvStr)
		return false, err
	}

	lines := strings.Split(string(output), "\n")

	if len(lines) < 2 {
		return false, fmt.Errorf("description incomplete")
	}

	if strings.Contains(lines[1], "VERSION") {
		return true, nil
	}

	return false, fmt.Errorf("GV [%s] not found", gvStr)
}

func (c *ClusterCollector) getGVKUsingNameCLI(name string) (string, string, error) {
	cmd := exec.Command("bash", "-c", c.getClusterCommand()+" explain "+name)
	output, err := cmd.Output()
	if err != nil {
		//logrus.Errorf("Error while running kubectl: %s\n", err)
		return "", "", err
	}

	var gvk schema.GroupVersionKind

	lines := strings.Split(string(output), "\n")

	if len(lines) < 2 {
		return "", "", fmt.Errorf("description incomplete")
	}

	if strings.Contains(lines[0], "KIND") {
		tmpArray := strings.Split(lines[0], ":")
		gvk.Kind = strings.TrimSpace(tmpArray[1])
	} else {
		return "", "", err
	}

	if strings.Contains(lines[1], "VERSION") {
		tmpArray := strings.Split(lines[1], ":")
		tmpGV := strings.TrimSpace(tmpArray[1])
		tmpGVLines := strings.Split(tmpGV, "/")
		if len(tmpGVLines) == 2 {
			gvk.Group = tmpGVLines[0]
			gvk.Version = tmpGVLines[1]
		} else {
			gvk.Group = ""
			gvk.Version = tmpGVLines[0]
		}
	}

	return gvk.Kind, gvk.GroupVersion().String(), nil
}

//GVExists looks up group version from list
func gvExists(gvList []schema.GroupVersion, gvKey schema.GroupVersion) bool {
	for _, gv := range gvList {
		if gv.String() == gvKey.String() {
			return true
		}
	}
	return false
}
