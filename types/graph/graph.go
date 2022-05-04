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

// Graph contains transformers and artifacts being passed between them.
type Graph struct {
	vertexId int
	edgeId   int
	Version  string         `json:"version"`
	Vertices map[int]Vertex `json:"vertices,omitempty"`
	Edges    map[int]Edge   `json:"edges,omitempty"`
}

// Vertex is a single transformer.
type Vertex struct {
	Id        int                    `json:"id"`
	Iteration int                    `json:"iteration"`
	Name      string                 `json:"name"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Edge is an artifact.
type Edge struct {
	Id   int                    `json:"id"`
	From int                    `json:"from"`
	To   int                    `json:"to"`
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// Node is the web UI version of Vertex.
type Node struct {
	Id       string   `json:"id"`
	Type     string   `json:"type,omitempty"`
	Position Position `json:"position"`
	Data     Data     `json:"data"`
}

// EdgeT is the web UI version of Edge.
type EdgeT struct {
	Id        string    `json:"id"`
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Label     string    `json:"label"`
	MarkerEnd MarkerEnd `json:"markerEnd"`
}

// MarkerEnd is used to indicate the style of edge endings.
type MarkerEnd struct {
	Type string `json:"type"`
}

// Position is the 2d position of a Node.
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Data contains the label for a vertex.
type Data struct {
	Label        string `json:"label"`
	PathMappings string `json:"pathMappings,omitempty"`
}

const (
	// GraphFileVersion is the version of the graph file that is generated.
	GraphFileVersion = "1.0.0"
	// GraphFileName is the default file name used to save the graph.
	GraphFileName = "m2k-graph.json"
	// GraphSourceVertexKey is used to track an artifact across iterations. It contains the transformer that created this artifact.
	GraphSourceVertexKey = "m2k-logging-source-vertex"
	// GraphProcessVertexKey is used to track an artifact across iterations. It contains the transformer that last processed this artifact.
	GraphProcessVertexKey = "m2k-logging-process-vertex"
	// GraphEdgeArrowClosed is used to indicate a closed arrow edge ending.
	GraphEdgeArrowClosed = "arrowclosed"
	// GraphNodeTypeInput is used to indicate that the Node is an input/starting node.
	GraphNodeTypeInput = "input"
)

// NewGraph creates a new graph.
func NewGraph() *Graph {
	return &Graph{
		vertexId: -1,
		edgeId:   -1,
		Version:  GraphFileVersion,
		Vertices: map[int]Vertex{},
		Edges:    map[int]Edge{},
	}
}

// AddVertex adds a new vertex to the graph.
func (g *Graph) AddVertex(name string, iteration int, data map[string]interface{}) int {
	g.vertexId++
	g.Vertices[g.vertexId] = Vertex{Id: g.vertexId, Iteration: iteration, Name: name, Data: data}
	return g.vertexId
}

// AddEdge adds a new edge to the graph.
func (g *Graph) AddEdge(from, to int, name string, data map[string]interface{}) int {
	g.edgeId++
	g.Edges[g.edgeId] = Edge{Id: g.edgeId, From: from, To: to, Name: name, Data: data}
	return g.edgeId
}
