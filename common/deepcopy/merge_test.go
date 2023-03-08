/*
 *  Copyright IBM Corporation 2023
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

package deepcopy_test

import (
	"testing"

	"github.com/konveyor/move2kube/common/deepcopy"
)

func TestMerge(t *testing.T) {
	t.Run("merge interface of slices with different integers", func(t *testing.T) {
		xs := []interface{}{1}
		ys := []interface{}{2}
		wanted := []interface{}{1, 2}
		zsI := deepcopy.Merge(xs, ys)
		zs, ok := zsI.([]interface{})
		if !ok {
			t.Fatal("expected the merged result to be a slice of interfaces")
		}
		if len(zs) != len(wanted) {
			t.Fatalf("length of the merged result is incorrect. expected: %d actual: %d", len(wanted), len(zs))
		}
		for i, z := range zs {
			zInt, ok := z.(int)
			if !ok {
				t.Fatal("expected the elements of the merged result to be integers")
			}
			if zInt != wanted[i] {
				t.Fatalf("the element at index is incorrect. expected: %d actual: %d", wanted[i], zInt)
			}
		}
	})
	t.Run("merge interface of slices with same integers", func(t *testing.T) {
		xs := []interface{}{1}
		ys := []interface{}{1}
		wanted := []interface{}{1}
		zsI := deepcopy.Merge(xs, ys)
		zs, ok := zsI.([]interface{})
		if !ok {
			t.Fatal("expected the merged result to be a slice of interfaces")
		}
		if len(zs) != len(wanted) {
			t.Fatalf("length of the merged result is incorrect. expected: %d actual: %d", len(wanted), len(zs))
		}
		for i, z := range zs {
			zInt, ok := z.(int)
			if !ok {
				t.Fatal("expected the elements of the merged result to be integers")
			}
			if zInt != wanted[i] {
				t.Fatalf("the element at index is incorrect. expected: %d actual: %d", wanted[i], zInt)
			}
		}
	})

	t.Run("merge interface of slices with duplicate integers in the same slice", func(t *testing.T) {
		xs := []interface{}{1, 2, 2}
		ys := []interface{}{0, 2, 3, 2, 1}
		wanted := []interface{}{1, 2, 2, 0, 3}
		zsI := deepcopy.Merge(xs, ys)
		zs, ok := zsI.([]interface{})
		if !ok {
			t.Fatal("expected the merged result to be a slice of interfaces")
		}
		if len(zs) != len(wanted) {
			t.Fatalf("length of the merged result is incorrect. expected: %d actual: %d", len(wanted), len(zs))
		}
		for i, z := range zs {
			zInt, ok := z.(int)
			if !ok {
				t.Fatal("expected the elements of the merged result to be integers")
			}
			if zInt != wanted[i] {
				t.Fatalf("the element at index is incorrect. expected: %d actual: %d", wanted[i], zInt)
			}
		}
	})
	t.Run("merge interface of slices with duplicate integers and strings in the same slice", func(t *testing.T) {
		xs := []interface{}{1, 2, "foo", 2}
		ys := []interface{}{"foo", 0, "bar", 2, 3, 2, 1}
		wanted := []interface{}{1, 2, "foo", 2, 0, "bar", 3}
		zsI := deepcopy.Merge(xs, ys)
		zs, ok := zsI.([]interface{})
		if !ok {
			t.Fatal("expected the merged result to be a slice of interfaces")
		}
		if len(zs) != len(wanted) {
			t.Fatalf("length of the merged result is incorrect. expected: %d actual: %d", len(wanted), len(zs))
		}
		for i, z := range zs {
			zInt, ok := z.(int)
			if !ok {
				zStr, ok := z.(string)
				if !ok {
					t.Fatal("expected the elements of the merged result to be integers or strings")
				}
				if zStr != wanted[i] {
					t.Fatalf("the element at index is incorrect. expected: %d actual: %d", wanted[i], zInt)
				}
				continue
			}
			if zInt != wanted[i] {
				t.Fatalf("the element at index is incorrect. expected: %d actual: %d", wanted[i], zInt)
			}
		}
	})
}
