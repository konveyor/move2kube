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
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

type context struct {
	ShouldConvert    bool
	Convert          func(string) (string, error)
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
	ctx := context{Convert: oldCtx.Convert}
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
		return ctx, fmt.Errorf("The m2kpath struct tag has an invalid format. Actual tag: %s", tag)
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
	return ctx, fmt.Errorf("Failed to process the tag. Actual tag: %s", tag)
}

func recurse(value reflect.Value, ctx context) error {
	// fmt.Printf("type [%v] ctx [%v]\n", value.Type(), ctx)
	switch value.Kind() {
	case reflect.String:
		if !ctx.ShouldConvert {
			break
		}
		if len(ctx.MapKeysToConvert) > 0 {
			if ctx.CurrentMapKey.Kind() != reflect.String {
				return fmt.Errorf("Map keys are not of kind string. Actual kind: %v", ctx.CurrentMapKey.Kind())
			}
			if !common.IsStringPresent(ctx.MapKeysToConvert, ctx.CurrentMapKey.String()) {
				break
			}
		}
		s, err := ctx.Convert(value.String())
		if err != nil {
			return fmt.Errorf("Failed to convert %s Error: %q", value.String(), err)
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
			if err := recurse(iter.Value(), ctx); err != nil {
				return err
			}
		}
	default:
		//fmt.Println("default. Actual kind:", value.Kind())
	}
	return nil
}

func convertPathsDecode(plan *Plan) error {
	rootDir, err := filepath.Abs(plan.Spec.Inputs.RootDir)
	if err != nil {
		log.Errorf("Failed to make the root directory path %q absolute. Error %q", plan.Spec.Inputs.RootDir, err)
		return err
	}
	plan.Spec.Inputs.RootDir = rootDir

	ctx := context{Convert: plan.GetAbsolutePath}
	planV := reflect.ValueOf(plan).Elem()
	return recurse(planV, ctx)
}

func convertPathsEncode(plan *Plan) error {
	ctx := context{Convert: plan.GetRelativePath}
	planV := reflect.ValueOf(plan).Elem()
	if err := recurse(planV, ctx); err != nil {
		log.Errorf("Error while converting absolute paths to relative. Error: %q", err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Failed to get the current working directory. Error %q", err)
		return err
	}
	rootDir, err := filepath.Rel(pwd, plan.Spec.Inputs.RootDir)
	if err != nil {
		log.Errorf("Failed to make the root directory path %q relative to the current working directory %q Error %q", rootDir, pwd, err)
		return err
	}
	plan.Spec.Inputs.RootDir = rootDir
	return nil
}

// ReadPlan decodes the plan from yaml converting relative paths to absolute.
func ReadPlan(path string) (Plan, error) {
	plan := Plan{}
	if err := common.ReadYaml(path, &plan); err != nil {
		log.Errorf("Failed to load the plan file at path %q Error %q", path, err)
		return plan, err
	}

	if err := convertPathsDecode(&plan); err != nil {
		return plan, err
	}
	return plan, nil
}

// Copy makes a copy of the plan.
func (plan *Plan) Copy() (Plan, error) {
	copy := Plan{}
	planBytes, err := yaml.Marshal(plan)
	if err != nil {
		log.Errorf("Failed to marshal the plan to yaml. Error: %q", err)
		return copy, err
	}
	err = yaml.Unmarshal(planBytes, &copy)
	return copy, err
}

// WritePlan encodes the plan to yaml converting absolute paths to relative.
func WritePlan(path string, plan Plan) error {
	copy, err := plan.Copy()
	if err != nil {
		log.Errorf("Failed to create a copy of the plan before writing. Error: %q", err)
		return err
	}
	if err := convertPathsEncode(&copy); err != nil {
		return err
	}
	return common.WriteYaml(path, copy)
}

// IsAssetsPath returns true if it is a m2kassets path.
func IsAssetsPath(path string) bool {
	pathParts := strings.Split(path, string(os.PathSeparator))
	if filepath.IsAbs(path) {
		return common.IsStringPresent(pathParts, common.AssetsDir)
	}
	return len(pathParts) > 0 && pathParts[0] == common.AssetsDir
}

// SetRootDir changes the root directory of the plan.
// The `rootDir` must be an cleaned absolute path.
func (plan *Plan) SetRootDir(rootDir string) error {
	oldRootDir := plan.Spec.Inputs.RootDir

	convert := func(oldPath string) (string, error) {
		if oldPath == "" || !filepath.IsAbs(oldPath) || IsAssetsPath(oldPath) {
			return oldPath, nil
		}
		relPath, err := filepath.Rel(oldRootDir, oldPath)
		if err != nil {
			return "", err
		}
		return filepath.Join(rootDir, relPath), nil
	}

	ctx := context{Convert: convert}
	planV := reflect.ValueOf(plan).Elem()
	err := recurse(planV, ctx)
	if err != nil {
		return err
	}

	plan.Spec.Inputs.RootDir = rootDir
	return nil
}

// GetRelativePath returns a path relative to the root directory of the plan
func (plan *Plan) GetRelativePath(absPath string) (string, error) {
	if absPath == "" {
		return absPath, nil
	}
	if !filepath.IsAbs(absPath) {
		log.Debugf("The input path %q is not an absolute path. Cannot make it relative to the root directory.", absPath)
		return absPath, nil
	}
	if IsAssetsPath(absPath) {
		return filepath.Rel(common.TempPath, absPath)
	}
	return filepath.Rel(plan.Spec.Inputs.RootDir, absPath)
}

// GetAbsolutePath takes a path relative to the plan's root directory or
// assets path and makes it absolute.
func (plan *Plan) GetAbsolutePath(relPath string) (string, error) {
	if relPath == "" {
		return relPath, nil
	}
	if filepath.IsAbs(relPath) {
		log.Debugf("The input path %q is not an relative path. Cannot make it absolute.", relPath)
		return relPath, nil
	}
	if IsAssetsPath(relPath) {
		return filepath.Join(common.TempPath, relPath), nil
	}
	return filepath.Join(plan.Spec.Inputs.RootDir, relPath), nil
}
