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

package pathconverters

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
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
	ctx := context{
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
		ctx.ShouldConvert = common.IsPresent(validValues, targetField)
		return ctx, nil
	}
	return ctx, fmt.Errorf("failed to process the tag. Actual tag: %s", tag)
}

func process(value reflect.Value, ctx context) error {
	//logrus.Debugf("type [%v] ctx [%v]\n", value.Type(), ctx)
	switch value.Kind() {
	case reflect.Invalid:
		logrus.Debugf("pathconverters.process got an invalid reflect.Value. Ignoring")
	case reflect.String:
		if !ctx.ShouldConvert {
			break
		}
		if len(ctx.MapKeysToConvert) > 0 {
			if ctx.CurrentMapKey.Kind() != reflect.String {
				return fmt.Errorf("map keys are not of kind string. Actual kind: %v", ctx.CurrentMapKey.Kind())
			}
			if !common.IsPresent(ctx.MapKeysToConvert, ctx.CurrentMapKey.String()) {
				break
			}
		}
		s, err := ctx.Convert(value.String())
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
			if err := process(structV.Field(i), ctx); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if err := process(value.Index(i), ctx); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			ctx.CurrentMapKey = iter.Key()
			v := iter.Value()
			if v.Kind() != reflect.String {
				if err := process(v, ctx); err != nil {
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
				if !common.IsPresent(ctx.MapKeysToConvert, ctx.CurrentMapKey.String()) {
					continue
				}
			}
			s, err := ctx.Convert(v.String())
			if err != nil {
				return fmt.Errorf("failed to convert %s Error: %q", v.String(), err)
			}
			value.SetMapIndex(ctx.CurrentMapKey, reflect.ValueOf(s))
		}
	case reflect.Interface, reflect.Pointer:
		if !value.IsNil() {
			if err := process(value.Elem(), ctx); err != nil {
				return err
			}
		}
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Bool:
		// Ignore
	default:
		logrus.Debugf("default. Actual kind: %s", value.Kind())
	}
	return nil
}

// MakePlanPathsAbsolute converts all paths in a plan to absolute path from relative path
func MakePlanPathsAbsolute(obj interface{}, sourcePath, assetsPath string) (err error) {
	function := func(relPath string) (string, error) {
		if relPath == "" {
			return relPath, nil
		}
		if filepath.IsAbs(relPath) {
			logrus.Debugf("The input path %q is not an relative path. Cannot make it absolute.", relPath)
			return relPath, nil
		}
		pathParts := strings.Split(relPath, string(os.PathSeparator))
		if len(pathParts) > 0 && pathParts[0] == common.AssetsDir {
			return filepath.Join(assetsPath, relPath), nil
		}
		return filepath.Join(sourcePath, relPath), nil
	}
	return ProcessPaths(obj, function)
}

// ChangePaths changes paths which are based out of one root to another root
func ChangePaths(obj interface{}, mapping map[string]string) error {
	for parent := range mapping {
		for child := range mapping {
			if parent != child {
				if common.IsParent(child, parent) {
					err := fmt.Errorf("the provided source paths %s is child of %s.", child, parent)
					logrus.Errorf("%s", err)
					return err
				}
			}
		}
	}
	function := func(path string) (string, error) {
		if path == "" {
			return path, nil
		}
		if !filepath.IsAbs(path) {
			err := fmt.Errorf("the input path %q is not an absolute path", path)
			logrus.Errorf("%s", err)
			return path, err
		}
		for src, dest := range mapping {
			if common.IsParent(path, src) {
				rel, err := filepath.Rel(src, path)
				if err != nil {
					err := fmt.Errorf("unable to make path (%s) relative to root (%s)", path, src)
					return path, err
				}
				return filepath.Join(dest, rel), nil
			}
		}
		return path, fmt.Errorf("unable to find proper root for %s", path)
	}
	return ProcessPaths(obj, function)
}

// ProcessPaths calls the process function for each path in an object
func ProcessPaths(obj interface{}, processfunc func(string) (string, error)) (err error) {
	ctx := context{Convert: processfunc}
	objV := reflect.ValueOf(obj).Elem()
	if err := process(objV, ctx); err != nil {
		return fmt.Errorf("failed to process the paths for the object of type %T and value %+v . Error: %q", obj, obj, err)
	}
	return nil
}
