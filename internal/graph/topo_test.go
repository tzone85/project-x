package graph

import "testing"

func TestTopoSort_LinearChain(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddNode("C")
	d.AddEdge("A", "B") // B depends on A
	d.AddEdge("B", "C") // C depends on B

	sorted, err := TopoSort(d)
	if err != nil {
		t.Fatalf("topo sort: %v", err)
	}
	// A must come before B, B before C
	indexOf := func(s string) int {
		for i, n := range sorted {
			if n == s {
				return i
			}
		}
		return -1
	}
	if indexOf("A") >= indexOf("B") {
		t.Error("A should come before B")
	}
	if indexOf("B") >= indexOf("C") {
		t.Error("B should come before C")
	}
}

func TestTopoSort_CycleError(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddEdge("A", "B")
	d.AddEdge("B", "A")

	_, err := TopoSort(d)
	if err == nil {
		t.Error("expected error for cyclic graph")
	}
}

func TestTopoSort_Empty(t *testing.T) {
	d := NewDAG()
	sorted, err := TopoSort(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 0 {
		t.Errorf("expected empty result, got %v", sorted)
	}
}

func TestTopoSort_SingleNode(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	sorted, err := TopoSort(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 1 || sorted[0] != "A" {
		t.Errorf("expected [A], got %v", sorted)
	}
}

func TestReadyNodes_RootNodes(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddNode("C")
	d.AddEdge("A", "C") // C depends on A
	d.AddEdge("B", "C") // C depends on B

	completed := map[string]bool{}
	ready := ReadyNodes(d, completed)
	// A and B have no deps, so both are ready
	if len(ready) != 2 {
		t.Fatalf("expected 2 ready nodes, got %d: %v", len(ready), ready)
	}
}

func TestReadyNodes_AfterCompletion(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddNode("C")
	d.AddEdge("A", "B") // B depends on A
	d.AddEdge("A", "C") // C depends on A

	completed := map[string]bool{"A": true}
	ready := ReadyNodes(d, completed)
	// B and C should now be ready
	if len(ready) != 2 {
		t.Fatalf("expected 2 ready nodes, got %d", len(ready))
	}
}

func TestReadyNodes_ExcludesCompleted(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddEdge("A", "B")

	completed := map[string]bool{"A": true, "B": true}
	ready := ReadyNodes(d, completed)
	if len(ready) != 0 {
		t.Errorf("expected 0 ready (all done), got %d", len(ready))
	}
}

func TestReadyNodes_EmptyGraph(t *testing.T) {
	d := NewDAG()
	completed := map[string]bool{}
	ready := ReadyNodes(d, completed)
	if len(ready) != 0 {
		t.Errorf("expected 0 ready for empty graph, got %d", len(ready))
	}
}

func TestReadyNodes_Deterministic(t *testing.T) {
	d := NewDAG()
	d.AddNode("C")
	d.AddNode("A")
	d.AddNode("B")

	completed := map[string]bool{}
	ready := ReadyNodes(d, completed)
	if len(ready) != 3 {
		t.Fatalf("expected 3 ready, got %d", len(ready))
	}
	// Should be sorted
	if ready[0] != "A" || ready[1] != "B" || ready[2] != "C" {
		t.Errorf("expected sorted [A B C], got %v", ready)
	}
}

func TestWaveGrouping(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddNode("C")
	d.AddNode("D")
	d.AddEdge("A", "C") // C depends on A
	d.AddEdge("B", "C") // C depends on B
	d.AddEdge("C", "D") // D depends on C

	waves, err := GroupByWave(d)
	if err != nil {
		t.Fatalf("wave grouping: %v", err)
	}
	// Wave 0: A, B (roots)
	// Wave 1: C (depends on A, B)
	// Wave 2: D (depends on C)
	if len(waves) != 3 {
		t.Fatalf("expected 3 waves, got %d", len(waves))
	}
	if len(waves[0]) != 2 {
		t.Errorf("wave 0: expected 2 nodes, got %d", len(waves[0]))
	}
	if len(waves[1]) != 1 || waves[1][0] != "C" {
		t.Errorf("wave 1: expected [C], got %v", waves[1])
	}
	if len(waves[2]) != 1 || waves[2][0] != "D" {
		t.Errorf("wave 2: expected [D], got %v", waves[2])
	}
}

func TestWaveGrouping_Independent(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddNode("C")
	// No edges -- all independent

	waves, _ := GroupByWave(d)
	if len(waves) != 1 {
		t.Fatalf("expected 1 wave for independent nodes, got %d", len(waves))
	}
	if len(waves[0]) != 3 {
		t.Errorf("wave 0: expected 3 nodes, got %d", len(waves[0]))
	}
}

func TestWaveGrouping_Empty(t *testing.T) {
	d := NewDAG()
	waves, _ := GroupByWave(d)
	if len(waves) != 0 {
		t.Errorf("expected 0 waves for empty DAG, got %d", len(waves))
	}
}

func TestWaveGrouping_CycleError(t *testing.T) {
	d := NewDAG()
	d.AddNode("A")
	d.AddNode("B")
	d.AddEdge("A", "B")
	d.AddEdge("B", "A")

	_, err := GroupByWave(d)
	if err == nil {
		t.Error("expected error for cyclic graph")
	}
}
