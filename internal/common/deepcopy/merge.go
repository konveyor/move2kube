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

package deepcopy

import (
	"reflect"

	"github.com/sirupsen/logrus"
)

// MergeInterface can be implemented by types to have custom deep copy logic.
type MergeInterface interface {
	Merge(interface{}) interface{}
}

// Merge returns a merge of x and y.
// Supports everything except Chan, Func and UnsafePointer.
func Merge(x interface{}, y interface{}) interface{} {
	return mergeRecursively(reflect.ValueOf(x), reflect.ValueOf(y)).Interface()
}

func mergeRecursively(xV reflect.Value, yV reflect.Value) reflect.Value {
	if !yV.IsValid() {
		logrus.Debugf("invalid value given to merge recursively value: %+v", yV)
		return xV
	}
	if !xV.IsValid() {
		logrus.Debugf("invalid value given to merge recursively value: %+v", xV)
		return yV
	}
	xT := xV.Type()
	xK := xV.Kind()
	yK := yV.Kind()
	if xK != yK {
		logrus.Debugf("Unable to merge due to varying types : %s & %s", xK, yK)
		return yV
	}
	if xV.CanInterface() {
		if mergable, ok := xV.Interface().(MergeInterface); ok {
			nV := reflect.ValueOf(mergable.Merge(yV.Interface()))
			if !nV.IsValid() {
				return yV
			}
			return nV
		}
	}
	switch xK {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return yV
	case reflect.Array, reflect.Slice:
		nV := reflect.MakeSlice(xT, xV.Len()+yV.Len(), xV.Cap()+yV.Cap())
		itemsYetToBeMerged := map[int]bool{}
		for i := 0; i < yV.Len(); i++ {
			itemsYetToBeMerged[i] = true
		}
		for i := 0; i < xV.Len(); i++ {
			merged := false
			for j := 0; j < yV.Len(); j++ {
				if compare(xV.Index(i), yV.Index(j)) {
					nV.Index(i).Set(mergeRecursively(xV.Index(i), yV.Index(j)))
					merged = true
					itemsYetToBeMerged[j] = false
					break
				}
			}
			if !merged {
				nV.Index(i).Set(copyRecursively(xV.Index(i)))
			}
		}
		for i := 0; i < yV.Len(); i++ {
			if itemsYetToBeMerged[i] {
				nV.Index(xV.Len() + i).Set(copyRecursively(yV.Index(i)))
			}
		}
		return nV
	case reflect.Interface:
		return mergeRecursively(xV.Elem(), yV.Elem())
	case reflect.Map:
		nV := reflect.MakeMapWithSize(xT, xV.Len()+yV.Len())
		for _, key := range xV.MapKeys() {
			if yV.MapIndex(key) == reflect.Zero(reflect.TypeOf(xV)).Interface() {
				nV.SetMapIndex(copyRecursively(key), copyRecursively(xV.MapIndex(key)))
				continue
			}
			nV.SetMapIndex(copyRecursively(key), mergeRecursively(xV.MapIndex(key), yV.MapIndex(key)))
		}
		for _, key := range yV.MapKeys() {
			if xV.MapIndex(key) == reflect.Zero(reflect.TypeOf(yV)).Interface() {
				nV.SetMapIndex(copyRecursively(key), copyRecursively(yV.MapIndex(key)))
				continue
			}
			nV.SetMapIndex(copyRecursively(key), mergeRecursively(xV.MapIndex(key), yV.MapIndex(key)))
		}
		return nV
	case reflect.Ptr:
		if xV.IsNil() {
			return xV
		}
		nV := reflect.New(xV.Elem().Type())
		nV.Elem().Set(mergeRecursively(xV.Elem(), yV.Elem()))
		return yV
	case reflect.Struct:
		nV := reflect.New(xT).Elem()
		for i := 0; i < xV.NumField(); i++ {
			if !nV.Field(i).CanSet() {
				continue
			}
			nV.Field(i).Set(mergeRecursively(xV.Field(i), yV.Field(i)))
		}
		return nV
	default:
		logrus.Debugf("unsupported for deep copy kind: %+v type: %+v value: %+v", xK, xT, xV)
		return xV
	}
}

// CompareInterface can be implemented by types to have custom deep copy logic.
type CompareInterface interface {
	Compare(interface{}, interface{}) bool
}

func compare(xV reflect.Value, yV reflect.Value) bool {
	if !yV.IsValid() {
		logrus.Debugf("invalid value given to merge recursively value: %+v", yV)
		return false
	}
	if !xV.IsValid() {
		logrus.Debugf("invalid value given to merge recursively value: %+v", xV)
		return false
	}
	xT := xV.Type()
	xK := xV.Kind()
	yK := yV.Kind()
	if xK != yK {
		logrus.Debugf("Unable to merge due to varying types : %s & %s", xK, yK)
		return false
	}
	if xV.CanInterface() {
		if comparable, ok := xV.Interface().(CompareInterface); ok {
			return comparable.Compare(xV.Interface(), yV.Interface())
		}
	}
	switch xK {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return xV == yV
	case reflect.Array:
		// TODO: Check this is the required behaviour
		return true
	case reflect.Slice:
		// TODO: Check this is the required behaviour
		return true
	case reflect.Interface:
		return compare(xV.Elem(), yV.Elem())
	case reflect.Map:
		// TODO: Check this is the required behaviour
		return true
	case reflect.Ptr:
		return compare(xV.Elem(), yV.Elem())
	case reflect.Struct:
		nV := reflect.New(xT).Elem()
		for i := 0; i < xV.NumField(); i++ {
			if nV.Type().Field(i).Name == "name" || nV.Type().Field(i).Name == "id" {
				return compare(xV.Field(i), yV.Field(i))
			}
		}
		return true
	default:
		logrus.Debugf("unsupported for deep compare kind: %+v type: %+v value: %+v", xK, xT, xV)
		return true
	}
}
