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
			Position: graphtypes.Position{X: 0, Y: vertex.Iteration}, // store the iteration in the Y coordinate (sub iterations in X coordinate)
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

type IterAndSubIter struct {
	Iter int
	Sub  int
}

type SortIters struct {
	Items []IterAndSubIter
}

// Len is the number of elements in the collection.
func (x *SortIters) Len() int {
	return len(x.Items)
}

// Less reports whether the element with index i
// must sort before the element with index j.
func (x *SortIters) Less(i, j int) bool {
	return x.Items[i].Iter < x.Items[j].Iter || (x.Items[i].Iter == x.Items[j].Iter && x.Items[i].Sub < x.Items[j].Sub)
}

// Swap swaps the elements with indexes i and j.
func (x *SortIters) Swap(i, j int) {
	x.Items[i], x.Items[j] = x.Items[j], x.Items[i]
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

	iterSubIterToY := map[IterAndSubIter]int{}
	{
		// depth first search to calculate sub-iterations
		subIters := map[IterAndSubIter]struct{}{}
		visited := map[string]struct{}{"v-0": {}}
		dfsRecursive(nodes, subIters, visited, adjMat, "v-0", -1)
		sortIters := SortIters{}
		for subIter := range subIters {
			sortIters.Items = append(sortIters.Items, subIter)
		}
		sort.Stable(&sortIters)
		for i, x := range sortIters.Items {
			iterSubIterToY[x] = i * 200
		}
	}

	visited := map[string]struct{}{"v-0": {}}
	iterSubIterToX := map[IterAndSubIter]int{}
	{
		// the bread first search algorithm
		queue := []string{"v-0"}
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
			ii := IterAndSubIter{Iter: node.Position.Y, Sub: node.Position.X}
			newX := iterSubIterToX[ii]
			iterSubIterToX[ii] = newX + 200
			node.Position.X = newX
			node.Position.Y = iterSubIterToY[ii]
			nodes[idx] = node
		}
	}

	// handle islands
	for i, node := range nodes {
		if _, ok := visited[node.Id]; ok {
			continue
		}
		logrus.Errorf("found an unvisited node: %+v", node)
		visited[node.Id] = struct{}{}
		// calculate the new position for the current node
		ii := IterAndSubIter{Iter: node.Position.Y, Sub: node.Position.X}
		newX := iterSubIterToX[ii]
		iterSubIterToX[ii] = newX + 200
		node.Position.X = newX
		node.Position.Y = iterSubIterToY[ii]
		nodes[i] = node
	}
}

func dfsRecursive(
	nodes []graphtypes.Node,
	subIters map[IterAndSubIter]struct{},
	visited map[string]struct{},
	adjMat map[string]map[string]struct{},
	current string,
	parentIdx int,
) {
	visited[current] = struct{}{}
	currentIdx := common.FindIndex(nodes, func(n graphtypes.Node) bool { return n.Id == current })
	if currentIdx < 0 {
		logrus.Errorf("failed to find the vertex with id '%s' in the list of nodes", current)
		return
	}
	// calculate sub-iteration
	currentNode := nodes[currentIdx]
	if parentIdx >= 0 {
		parentNode := nodes[parentIdx]
		parentIteration := parentNode.Position.Y
		currentIteration := currentNode.Position.Y
		if parentIteration == currentIteration {
			// store the sub iteration in the X coordinate
			currentNode.Position.X = parentNode.Position.X + 1
			nodes[currentIdx] = currentNode
		}
	}
	subIters[IterAndSubIter{Iter: currentNode.Position.Y, Sub: currentNode.Position.X}] = struct{}{}
	neighbours := adjMat[current]
	for neighbour := range neighbours {
		if _, ok := visited[neighbour]; !ok {
			dfsRecursive(nodes, subIters, visited, adjMat, neighbour, currentIdx)
		}
	}
}
