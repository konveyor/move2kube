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

package apiresource

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func createService(name string, ports []v1.ServicePort) runtime.Object {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.ServiceSpec{
			Type:  v1.ServiceTypeNodePort,
			Ports: ports,
		},
	}
}

func TestDeepMerge(t *testing.T) {
	tcs := []struct {
		desc string
		x    runtime.Object
		y    runtime.Object
		want runtime.Object
	}{
		{
			desc: "no ports in common",
			x: createService("svc1", []v1.ServicePort{
				{Name: "port1", Port: 1111},
				{Name: "port2", Port: 2222},
			}),
			y: createService("svc1", []v1.ServicePort{
				{Name: "port3", Port: 3333},
				{Name: "port4", Port: 4444},
			}),
			want: createService("svc1", []v1.ServicePort{
				{Name: "port3", Port: 3333},
				{Name: "port4", Port: 4444},
				{Name: "port1", Port: 1111},
				{Name: "port2", Port: 2222},
			}),
		},
		{
			desc: "some ports in common",
			x: createService("svc1", []v1.ServicePort{
				{Name: "port1", Port: 1111},
				{Name: "port2", Port: 2222},
			}),
			y: createService("svc1", []v1.ServicePort{
				{Name: "port2", Port: 2222},
				{Name: "port3", Port: 3333},
				{Name: "port4", Port: 4444},
			}),
			want: createService("svc1", []v1.ServicePort{
				{Name: "port1", Port: 1111},
				{Name: "port2", Port: 2222},
				{Name: "port3", Port: 3333},
				{Name: "port4", Port: 4444},
			}),
		},
		{
			desc: "some ports in common with same number but different name",
			x: createService("svc1", []v1.ServicePort{
				{Name: "port1", Port: 1111},
				{Name: "port2", Port: 2222},
			}),
			y: createService("svc1", []v1.ServicePort{
				{Name: "abcde", Port: 2222},
				{Name: "port3", Port: 3333},
				{Name: "port4", Port: 4444},
			}),
			want: createService("svc1", []v1.ServicePort{
				{Name: "port1", Port: 1111},
				{Name: "abcde", Port: 2222},
				{Name: "port3", Port: 3333},
				{Name: "port4", Port: 4444},
			}),
		},
		{
			desc: "some ports in common with different number but same name",
			x: createService("svc1", []v1.ServicePort{
				{Name: "port1", Port: 1111},
				{Name: "port2", Port: 2222},
			}),
			y: createService("svc1", []v1.ServicePort{
				{Name: "port2", Port: 1234},
				{Name: "port3", Port: 3333},
				{Name: "port4", Port: 4444},
			}),
			want: createService("svc1", []v1.ServicePort{
				{Name: "port2", Port: 1234},
				{Name: "port3", Port: 3333},
				{Name: "port4", Port: 4444},
				{Name: "port1", Port: 1111},
				{Name: "port2", Port: 2222},
			}),
		},
	}

	apiResource := APIResource{}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			actual, err := apiResource.deepMerge(tc.x, tc.y)
			if err != nil {
				t.Fatalf("Failed to merge the objects %v and %v Error: %q", tc.x, tc.y, actual)
			}
			if !cmp.Equal(actual, tc.want) {
				t.Fatalf("Failed to merge properly. Difference:\n%s", cmp.Diff(tc.want, actual))
			}
		})
	}
}
