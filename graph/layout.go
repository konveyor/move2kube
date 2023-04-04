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
	"sort"

	"github.com/konveyor/move2kube/common"
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

// BfsUpdatePositions updates the positions of the nodes using a layered layout algorithm that utilizes Breadth First Search.
func BfsUpdatePositions(nodes []graphtypes.Node, edges []graphtypes.EdgeT) {
	adjMat := map[string]map[string]struct{}{}
	for _, edge := range edges {
		if adjMat[edge.Source] == nil {
			adjMat[edge.Source] = map[string]struct{}{}
		}
		adjMat[edge.Source][edge.Target] = struct{}{}
	}

	// the bread first search algorithm
	queue := []string{"v-0"}
	visited := map[string]struct{}{"v-0": {}}
	xsPerIteration := map[int]int{}
	for len(queue) > 0 {
		// pop a vertex from the queue
		current := queue[0]
		queue = queue[1:]
		// add its neighbours to the queue for processing later
		neighbours := adjMat[current]
		sortedNotVisitedNeighbours := []string{}
		for neighbour := range neighbours {
			if _, ok := visited[neighbour]; !ok {
				sortedNotVisitedNeighbours = append(sortedNotVisitedNeighbours, neighbour)
				visited[neighbour] = struct{}{}
			}
		}
		sort.StringSlice(sortedNotVisitedNeighbours).Sort()
		queue = append(queue, sortedNotVisitedNeighbours...)
		// calculate the new position for the current node
		idx := common.FindIndex(nodes, func(n graphtypes.Node) bool { return n.Id == current })
		if idx < 0 {
			logrus.Errorf("failed to find a node for the vertex with id '%s'", current)
			continue
		}
		node := nodes[idx]
		iteration := node.Position.Y
		newX := xsPerIteration[iteration]
		xsPerIteration[iteration] = newX + 200
		newY := iteration * 200
		node.Position.X = newX
		node.Position.Y = newY
		nodes[idx] = node
	}

	// handle islands
	for i, node := range nodes {
		if _, ok := visited[node.Id]; ok {
			continue
		}
		logrus.Errorf("found an unvisited node: %+v", node)
		visited[node.Id] = struct{}{}
		// calculate the new position for the current node
		iteration := node.Position.Y
		newX := xsPerIteration[iteration]
		xsPerIteration[iteration] = newX + 200
		newY := iteration * 200
		node.Position.X = newX
		node.Position.Y = newY
		nodes[i] = node
	}
}
