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

package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

type context struct {
	RootDir          string
	ShouldConvert    bool
	Convert          func(string, string) (string, error)
	MapKeysToConvert []string
	CurrentMapKey    reflect.Value
}

const (
	structTag           = "m2kpath"
	structTagNormal     = "normal"
	structTagMapKeys    = "keys"
	structTagCondition  = "if"
	structTagCheckField = "in"
)

func processTag(structT reflect.Type, structV reflect.Value, i int, oldCtx context) (context, error) {
	ctx := context{
		RootDir: oldCtx.RootDir,
		Convert: oldCtx.Convert,
	}
	tag, ok := structT.Field(i).Tag.Lookup(structTag)
	if !ok {
		ctx.ShouldConvert = false
		return ctx, nil
	}
	if tag == structTagNormal {
		ctx.ShouldConvert = true
		return ctx, nil
	}
	// Special cases.
	tagParts := strings.Split(tag, ":")
	if len(tagParts) == 0 {
		return ctx, fmt.Errorf("the m2kpath struct tag has an invalid format. Actual tag: %s", tag)
	}
	// Special case for converting subset of map.
	if len(tagParts) == 2 && tagParts[0] == structTagMapKeys {
		ctx.ShouldConvert = true
		ctx.MapKeysToConvert = strings.Split(tagParts[1], ",")
		return ctx, nil
	}
	// Special case for conditional conversion.
	if len(tagParts) == 4 && tagParts[0] == structTagCondition && tagParts[2] == structTagCheckField {
		targetField := structV.FieldByName(tagParts[1]).String()
		validValues := strings.Split(tagParts[3], ",")
		ctx.ShouldConvert = common.IsStringPresent(validValues, targetField)
		return ctx, nil
	}
	return ctx, fmt.Errorf("failed to process the tag. Actual tag: %s", tag)
}

func recurse(value reflect.Value, ctx context) error {
	//logrus.Infof("type [%v] ctx [%v]\n", value.Type(), ctx)
	switch value.Kind() {
	case reflect.String:
		if !ctx.ShouldConvert {
			break
		}
		if len(ctx.MapKeysToConvert) > 0 {
			if ctx.CurrentMapKey.Kind() != reflect.String {
				return fmt.Errorf("map keys are not of kind string. Actual kind: %v", ctx.CurrentMapKey.Kind())
			}
			if !common.IsStringPresent(ctx.MapKeysToConvert, ctx.CurrentMapKey.String()) {
				break
			}
		}
		s, err := ctx.Convert(value.String(), ctx.RootDir)
		if err != nil {
			return fmt.Errorf("failed to convert %s Error: %q", value.String(), err)
		}
		value.SetString(s)
	case reflect.Struct:
		structV := value
		structT := value.Type()
		for i := 0; i < structV.NumField(); i++ {
			ctx, err := processTag(structT, structV, i, ctx)
			if err != nil {
				return err
			}
			if err := recurse(structV.Field(i), ctx); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if err := recurse(value.Index(i), ctx); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			ctx.CurrentMapKey = iter.Key()
			v := iter.Value()
			if v.Kind() != reflect.String {
				if err := recurse(v, ctx); err != nil {
					return err
				}
				continue
			}
			if !ctx.ShouldConvert {
				continue
			}
			if len(ctx.MapKeysToConvert) > 0 {
				if ctx.CurrentMapKey.Kind() != reflect.String {
					return fmt.Errorf("map keys are not of kind string. Actual kind: %v", ctx.CurrentMapKey.Kind())
				}
				if !common.IsStringPresent(ctx.MapKeysToConvert, ctx.CurrentMapKey.String()) {
					continue
				}
			}
			s, err := ctx.Convert(v.String(), ctx.RootDir)
			if err != nil {
				return fmt.Errorf("failed to convert %s Error: %q", v.String(), err)
			}
			value.SetMapIndex(ctx.CurrentMapKey, reflect.ValueOf(s))
		}
	default:
		//fmt.Println("default. Actual kind:", value.Kind())
	}
	return nil
}

func ConvertPathsDecode(obj interface{}, rootDir string) (absRootDir string, err error) {
	absRootDir, err = filepath.Abs(rootDir)
	if err != nil {
		logrus.Errorf("Failed to make the root directory path %q absolute. Error %q", rootDir, err)
		return absRootDir, err
	}
	ctx := context{
		RootDir: absRootDir,
		Convert: GetAbsolutePath,
	}
	planV := reflect.ValueOf(obj).Elem()
	return absRootDir, recurse(planV, ctx)
}

func ConvertPathsEncode(obj interface{}, rootDir string) (relRootDir string, err error) {
	ctx := context{
		RootDir: rootDir,
		Convert: GetRelativePath,
	}
	planV := reflect.ValueOf(obj).Elem()
	if err := recurse(planV, ctx); err != nil {
		logrus.Errorf("Error while converting absolute paths to relative. Error: %q", err)
		return "", err
	}
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Failed to get the current working directory. Error %q", err)
		return "", err
	}
	relRootDir, err = filepath.Rel(pwd, rootDir)
	if err != nil {
		logrus.Errorf("Failed to make the root directory path %q relative to the current working directory %q Error %q", rootDir, pwd, err)
	}
	return relRootDir, err
}

func ChangePaths(obj interface{}, currRoot, newRoot string) (err error) {
	_, err = ConvertPathsEncode(obj, currRoot)
	if err != nil {
		logrus.Errorf("Unable to encode of translator obj %+v. Ignoring : %s", obj, err)
		return err
	}
	_, err = ConvertPathsDecode(obj, newRoot)
	if err != nil {
		logrus.Errorf("Unable to decode of translator obj %+v. Ignoring : %s", obj, err)
		return err
	}
	return nil
}

// ReadPlan decodes the plan from yaml converting relative paths to absolute.
func ReadPlan(path string) (Plan, error) {
	plan := Plan{}
	var err error
	if err = common.ReadMove2KubeYaml(path, &plan); err != nil {
		logrus.Errorf("Failed to load the plan file at path %q Error %q", path, err)
		return plan, err
	}

	if plan.Spec.RootDir, err = ConvertPathsDecode(&plan, plan.Spec.RootDir); err != nil {
		return plan, err
	}
	return plan, nil
}

// CopyObj makes a copy of the serializable Object.
func CopyObj(obj interface{}, copy interface{}) error {
	planBytes, err := yaml.Marshal(obj)
	if err != nil {
		logrus.Errorf("Failed to marshal the plan to yaml. Error: %q", err)
		return err
	}
	err = yaml.Unmarshal(planBytes, copy)
	if err != nil {
		logrus.Errorf("Failed to unmarshal the yaml to plan. Error: %q", err)
		return err
	}
	return nil
}

// WritePlan encodes the plan to yaml converting absolute paths to relative.
func WritePlan(path string, plan Plan) error {
	copy := Plan{}
	err := CopyObj(plan, &copy)
	if err != nil {
		logrus.Errorf("Failed to create a copy of the plan before writing. Error: %q", err)
		return err
	}
	if copy.Spec.RootDir, err = ConvertPathsEncode(&copy, copy.Spec.RootDir); err != nil {
		return err
	}
	return common.WriteYaml(path, copy)
}

// IsAssetsPath returns true if it is a m2kassets path.
func IsAssetsPath(path string) bool {
	if filepath.IsAbs(path) {
		return strings.HasPrefix(path, filepath.Join(common.TempPath, common.AssetsDir))
	}
	pathParts := strings.Split(path, string(os.PathSeparator))
	return len(pathParts) > 0 && pathParts[0] == common.AssetsDir
}

// SetRootDir changes the root directory of the plan.
// The `rootDir` must be an cleaned absolute path.
func (plan *Plan) SetRootDir(rootDir string) error {
	oldRootDir := plan.Spec.RootDir

	convert := func(oldPath, rootDir string) (string, error) {
		if oldPath == "" || !filepath.IsAbs(oldPath) || IsAssetsPath(oldPath) {
			return oldPath, nil
		}
		relPath, err := filepath.Rel(oldRootDir, oldPath)
		if err != nil {
			return "", err
		}
		return filepath.Join(rootDir, relPath), nil
	}

	ctx := context{
		RootDir: plan.Spec.RootDir,
		Convert: convert,
	}
	planV := reflect.ValueOf(plan).Elem()
	err := recurse(planV, ctx)
	if err != nil {
		return err
	}

	plan.Spec.RootDir = rootDir
	return nil
}

// GetRelativePath returns a path relative to the root directory of the plan
func GetRelativePath(absPath, rootDir string) (string, error) {
	if absPath == "" {
		return absPath, nil
	}
	if !filepath.IsAbs(absPath) {
		err := fmt.Errorf("the input path %q is not an absolute path. Cannot make it relative to the root directory (%s)", absPath, rootDir)
		logrus.Errorf("%s", err)
		return absPath, err
	}
	if IsAssetsPath(absPath) {
		return filepath.Rel(common.TempPath, absPath)
	}
	return filepath.Rel(rootDir, absPath)
}

// GetAbsolutePath takes a path relative to the plan's root directory or
// assets path and makes it absolute.
func GetAbsolutePath(relPath, rootDir string) (string, error) {
	if relPath == "" {
		return relPath, nil
	}
	if filepath.IsAbs(relPath) {
		logrus.Debugf("The input path %q is not an relative path. Cannot make it absolute.", relPath)
		return relPath, nil
	}
	if IsAssetsPath(relPath) {
		return filepath.Join(common.TempPath, relPath), nil
	}
	return filepath.Join(rootDir, relPath), nil
}
