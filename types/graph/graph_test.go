/*
 *  Copyright IBM Corporation 2022
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

package graph

import (
	"reflect"
	"testing"
)

func TestGraph(t *testing.T) {

	t.Run("test for new graph", func(t *testing.T) {
		expected := &Graph{
			SourceVertexId: -1,
			vertexId:       -1,
			edgeId:         -1,
			Version:        GraphFileVersion,
			Vertices:       map[int]Vertex{},
			Edges:          map[int]Edge{},
		}
		result := NewGraph()
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("failed to create new graph need %+v, got %+v", expected, result)
		}
	})

	t.Run("test for adding vertex", func(t *testing.T) {
		g := NewGraph()
		expectedId := 0
		expectedVertex := Vertex{Id: expectedId, Iteration: 1, Name: "vertex1", Data: map[string]interface{}{}}

		resultId := g.AddVertex("vertex1", 1, map[string]interface{}{})
		if resultId != expectedId {
			t.Errorf("failed to add vertex. need %d, got %d", expectedId, resultId)
		}
		if !reflect.DeepEqual(g.Vertices[resultId], expectedVertex) {
			t.Errorf("failed to add vertex. need %+v, got %+v", expectedVertex, g.Vertices[resultId])
		}

		expectedId = 1
		expectedVertex = Vertex{Id: expectedId, Iteration: 2, Name: "vertex2", Data: map[string]interface{}{"foo": "bar"}}
		resultId = g.AddVertex("vertex2", 2, map[string]interface{}{"foo": "bar"})
		if resultId != expectedId {
			t.Errorf("failed to add vertex. need %d, got %d", expectedId, resultId)
		}
		if !reflect.DeepEqual(g.Vertices[resultId], expectedVertex) {
			t.Errorf("failed to add vertex. need %+v, got %+v", expectedVertex, g.Vertices[resultId])
		}
	})

	t.Run("test for adding edge", func(t *testing.T) {
		g := NewGraph()
		v1 := g.AddVertex("test1", 1, map[string]interface{}{})
		v2 := g.AddVertex("test2", 2, map[string]interface{}{})
		expectedId := 0
		expectedEdge := Edge{Id: expectedId, From: v1, To: v2, Name: "edge1", Data: map[string]interface{}{}}

		// Test adding an edge
		resultId := g.AddEdge(v1, v2, "edge1", map[string]interface{}{})
		if resultId != expectedId {
			t.Errorf("AddEdge() returned %d, expected %d", resultId, expectedId)
		}
		if !reflect.DeepEqual(g.Edges[resultId], expectedEdge) {
			t.Errorf("Added edge %+v does not match expected edge %+v", g.Edges[resultId], expectedEdge)
		}

		// Test adding a second edge
		v3 := g.AddVertex("test3", 3, map[string]interface{}{})
		expectedId = 1
		expectedEdge = Edge{Id: expectedId, From: v2, To: v3, Name: "edge2", Data: map[string]interface{}{"foo": "bar"}}
		resultId = g.AddEdge(v2, v3, "edge2", map[string]interface{}{"foo": "bar"})
		if resultId != expectedId {
			t.Errorf("failed to add edge. need %+v, got %+v", expectedId, resultId)
		}
		if !reflect.DeepEqual(g.Edges[resultId], expectedEdge) {
			t.Errorf("failed to add edge. need %+v, got %+v", expectedEdge, g.Edges[resultId])
		}
	})
}
