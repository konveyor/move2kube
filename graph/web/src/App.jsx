import React, { useEffect } from "react";
import ReactFlow, { useNodesState, useEdgesState, MiniMap, Controls, Background } from "react-flow-renderer";

function process(x) {
  return x.split('\n').map((line, i) => <div key={i}>{line}</div>)
}

const App = () => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  useEffect(() => {
    fetch('graph.json')
      .then(res => res.json())
      .then(d => {
        console.log(d);
        d.nodes = d.nodes.map(node => {
          const newLabel = (<div className="on-hover">
            {process(node.data.label)}
            {node.data.pathMappings && <div className="on-hover-child"><div>pathMappings:</div>{process(node.data.pathMappings)}</div>}
          </div>);
          return { ...node, data: { ...node.data, label: newLabel } };
        });
        setNodes(d.nodes);
        setEdges(d.edges);
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      fitView
    >
      <MiniMap />
      <Controls />
      <Background />
    </ReactFlow>
  );
};

export default App;
