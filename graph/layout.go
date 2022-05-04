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
	"fmt"

	graphtypes "github.com/konveyor/move2kube/types/graph"
	"github.com/sirupsen/logrus"
)

// GetNodesAndEdges returns nodes and edges that can be displayed using the web UI.
func GetNodesAndEdges(graph graphtypes.Graph) ([]graphtypes.Node, []graphtypes.EdgeT) {
	nodes := []graphtypes.Node{}
	for _, vertex := range graph.Vertices {
		pathMappings := ""
		if vertex.Data != nil {
			if value, ok := vertex.Data["pathMappings"].(string); ok {
				pathMappings = value
			}
		}
		node := graphtypes.Node{
			Id:       fmt.Sprintf("v-%d", vertex.Id),
			Position: graphtypes.Position{X: 0, Y: vertex.Iteration}, // store the iteration in the Y coordinate
			Data:     graphtypes.Data{Label: vertex.Name, PathMappings: pathMappings},
		}
		if vertex.Id == 0 {
			node.Type = graphtypes.GraphNodeTypeInput
		}
		nodes = append(nodes, node)
	}
	edges := []graphtypes.EdgeT{}
	for _, edge := range graph.Edges {
		label := edge.Name
		if xs, ok := edge.Data["newArtifact"].([]interface{}); ok && len(xs) != 0 {
			label += " (" + xs[0].(string) + ")"
		}
		edges = append(edges, graphtypes.EdgeT{
			Id:        fmt.Sprintf("e-%d", edge.Id),
			Source:    fmt.Sprintf("v-%d", edge.From),
			Target:    fmt.Sprintf("v-%d", edge.To),
			Label:     label,
			MarkerEnd: graphtypes.MarkerEnd{Type: graphtypes.GraphEdgeArrowClosed},
		})
	}
	return nodes, edges
}

// DfsUpdatePositions updates the positions of the nodes using a layered layout algorithm that utilizes Depth First Search.
func DfsUpdatePositions(nodes []graphtypes.Node, edges []graphtypes.EdgeT) {
	visited := map[string]bool{}
	adjMat := map[string]map[string]bool{}
	xsPerIteration := map[int]int{}
	for _, edge := range edges {
		if adjMat[edge.Source] == nil {
			adjMat[edge.Source] = map[string]bool{}
		}
		adjMat[edge.Source][edge.Target] = true
	}
	dfsHelper(nodes, edges, visited, adjMat, "v-0", xsPerIteration, 0, 0, 0)

	// handle islands

	for i, node := range nodes {
		if visited[node.Id] {
			continue
		}
		logrus.Errorf("found an unvisited node: %+v", node)

		iteration := node.Position.Y

		newX := 0
		if oldX, ok := xsPerIteration[iteration]; ok {
			newX = oldX + 200
		}
		xsPerIteration[iteration] = newX
		node.Position.X = newX

		newY := iteration * 200
		node.Position.Y = newY

		nodes[i] = node
	}
}

func dfsHelper(nodes []graphtypes.Node, edges []graphtypes.EdgeT, visited map[string]bool, adjMat map[string]map[string]bool, currVertId string, xsPerIteration map[int]int, parentIteration, parentX, parentY int) {
	visited[currVertId] = true
	newX := 0
	newY := 0
	iteration := 0
	for i, node := range nodes {
		if node.Id != currVertId {
			continue
		}

		iteration = node.Position.Y

		if iteration == parentIteration {
			// mandatory/on-demand passthrough
			newX = parentX
			newY = parentY + 100
		} else {
			newX = 0
			if oldX, ok := xsPerIteration[iteration]; ok {
				newX = oldX + 200
			}
			xsPerIteration[iteration] = newX
			newY = iteration * 200
		}

		node.Position.X = newX
		node.Position.Y = newY
		nodes[i] = node
		break
	}
	for neighbourVertId, ok := range adjMat[currVertId] {
		if !ok || visited[neighbourVertId] {
			// no edge from the current vertex to this neighbour vertex OR this neighbour has already been visited
			continue
		}
		dfsHelper(nodes, edges, visited, adjMat, neighbourVertId, xsPerIteration, iteration, newX, newY)
	}
}
