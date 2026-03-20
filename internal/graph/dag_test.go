package graph

import "testing"

func TestDAG_AddNodeAndEdge(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddEdge("A", "B") // B depends on A

	nodes := d.Nodes()
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestDAG_DetectCycle(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddNode("C")
	d.AddEdge("A", "B")
	d.AddEdge("B", "C")
	d.AddEdge("C", "A") // creates cycle

	if !d.HasCycle() {
		t.Error("expected cycle detection")
	}
}

func TestDAG_NoCycle(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddNode("C")
	d.AddEdge("A", "B")
	d.AddEdge("A", "C")

	if d.HasCycle() {
		t.Error("no cycle should be detected")
	}
}

func TestDAG_DuplicateNode(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("A") // duplicate, should be idempotent
	if len(d.Nodes()) != 1 {
		t.Error("duplicate node should be ignored")
	}
}

func TestDAG_Dependencies(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddNode("C")
	d.AddEdge("A", "B") // B depends on A
	d.AddEdge("A", "C") // C depends on A

	deps := d.DependenciesOf("B")
	if len(deps) != 1 || deps[0] != "A" {
		t.Errorf("expected B depends on [A], got %v", deps)
	}
}

func TestDAG_DependentsOf(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddNode("C")
	d.AddEdge("A", "B")
	d.AddEdge("A", "C")

	dependents := d.DependentsOf("A")
	if len(dependents) != 2 {
		t.Errorf("expected 2 dependents of A, got %d: %v", len(dependents), dependents)
	}
}

func TestDAG_NodesSorted(t *testing.T) {
	d := NewDAG()
	d.AddNode("C")
	d.AddNode("A")
	d.AddNode("B")

	nodes := d.Nodes()
	if nodes[0] != "A" || nodes[1] != "B" || nodes[2] != "C" {
		t.Errorf("expected sorted [A B C], got %v", nodes)
	}
}

func TestDAG_DependenciesOfUnknownNode(t *testing.T) {
	d := NewDAG()
	deps := d.DependenciesOf("X")
	if len(deps) != 0 {
		t.Errorf("expected empty deps for unknown node, got %v", deps)
	}
}

func TestDAG_DependentsOfUnknownNode(t *testing.T) {
	d := NewDAG()
	deps := d.DependentsOf("X")
	if len(deps) != 0 {
		t.Errorf("expected empty dependents for unknown node, got %v", deps)
	}
}

func TestDAG_EmptyGraph(t *testing.T) {
	d := NewDAG()
	if len(d.Nodes()) != 0 {
		t.Error("expected 0 nodes in empty graph")
	}
	if d.HasCycle() {
		t.Error("empty graph should not have cycle")
	}
}
