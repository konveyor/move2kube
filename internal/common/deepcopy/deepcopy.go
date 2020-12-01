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

	log "github.com/sirupsen/logrus"
)

// Interface can be implemented by types to have custom deep copy logic.
type Interface interface {
	DeepCopy() interface{}
}

// DeepCopy returns a deep copy of x.
// Supports everything except Chan, Func and UnsafePointer.
func DeepCopy(x interface{}) interface{} {
	return copyRecursively(reflect.ValueOf(x)).Interface()
}

func copyRecursively(xV reflect.Value) reflect.Value {
	if !xV.IsValid() {
		log.Debugf("invalid value given to copy recursively value: %+v", xV)
		return xV
	}
	if xV.CanInterface() {
		if deepCopyAble, ok := xV.Interface().(Interface); ok {
			yV := reflect.ValueOf(deepCopyAble.DeepCopy())
			if !yV.IsValid() {
				return xV
			}
			return yV
		}
	}
	xT := xV.Type()
	xK := xV.Kind()
	switch xK {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return xV
	case reflect.Array:
		yV := reflect.New(xT).Elem()
		for i := 0; i < xV.Len(); i++ {
			yV.Index(i).Set(copyRecursively(xV.Index(i)))
		}
		return yV
	case reflect.Slice:
		if xK == reflect.Slice && xV.IsNil() {
			return xV
		}
		yV := reflect.MakeSlice(xT, xV.Len(), xV.Cap())
		for i := 0; i < xV.Len(); i++ {
			yV.Index(i).Set(copyRecursively(xV.Index(i)))
		}
		return yV
	case reflect.Interface:
		if xV.IsNil() {
			return xV
		}
		return copyRecursively(xV.Elem())
	case reflect.Map:
		if xV.IsNil() {
			return xV
		}
		yV := reflect.MakeMapWithSize(xT, xV.Len())
		for _, key := range xV.MapKeys() {
			yV.SetMapIndex(copyRecursively(key), copyRecursively(xV.MapIndex(key)))
		}
		return yV
	case reflect.Ptr:
		if xV.IsNil() {
			return xV
		}
		yV := reflect.New(xV.Elem().Type())
		yV.Elem().Set(copyRecursively(xV.Elem()))
		return yV
	case reflect.Struct:
		yV := reflect.New(xT).Elem()
		for i := 0; i < xV.NumField(); i++ {
			if !yV.Field(i).CanSet() {
				continue
			}
			yV.Field(i).Set(copyRecursively(xV.Field(i)))
		}
		return yV
	default:
		log.Debugf("unsupported for deep copy kind: %+v type: %+v value: %+v", xK, xT, xV)
		return xV
	}
}
