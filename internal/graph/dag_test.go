package graph

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
)

// --- AddNode Tests ---

func TestAddNode(t *testing.T) {
	d := New()
	if err := d.AddNode("A"); err != nil {
		t.Fatalf("AddNode(A) unexpected error: %v", err)
	}
	if d.NodeCount() != 1 {
		t.Fatalf("expected 1 node, got %d", d.NodeCount())
	}
}

func TestAddNodeDuplicate(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	err := d.AddNode("A")
	if !errors.Is(err, ErrDuplicateNode) {
		t.Fatalf("expected ErrDuplicateNode, got %v", err)
	}
}

func TestAddMultipleNodes(t *testing.T) {
	d := New()
	for _, id := range []string{"A", "B", "C"} {
		if err := d.AddNode(id); err != nil {
			t.Fatalf("AddNode(%s) unexpected error: %v", id, err)
		}
	}
	if d.NodeCount() != 3 {
		t.Fatalf("expected 3 nodes, got %d", d.NodeCount())
	}
}

// --- RemoveNode Tests ---

func TestRemoveNode(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddEdge("A", "B")

	if err := d.RemoveNode("A"); err != nil {
		t.Fatalf("RemoveNode(A) unexpected error: %v", err)
	}
	if d.NodeCount() != 1 {
		t.Fatalf("expected 1 node, got %d", d.NodeCount())
	}
	if d.EdgeCount() != 0 {
		t.Fatalf("expected 0 edges after removing node, got %d", d.EdgeCount())
	}
}

func TestRemoveNodeNotFound(t *testing.T) {
	d := New()
	err := d.RemoveNode("X")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestRemoveNodeCleansReverseEdges(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("C")
	_ = d.AddEdge("A", "B") // A depends on B
	_ = d.AddEdge("C", "B") // C depends on B

	_ = d.RemoveNode("B")

	// A and C should have no edges left
	if d.EdgeCount() != 0 {
		t.Fatalf("expected 0 edges, got %d", d.EdgeCount())
	}
}

// --- AddEdge Tests ---

func TestAddEdge(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	if err := d.AddEdge("A", "B"); err != nil {
		t.Fatalf("AddEdge(A,B) unexpected error: %v", err)
	}
	if d.EdgeCount() != 1 {
		t.Fatalf("expected 1 edge, got %d", d.EdgeCount())
	}
}

func TestAddEdgeSelfLoop(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	err := d.AddEdge("A", "A")
	if !errors.Is(err, ErrSelfEdge) {
		t.Fatalf("expected ErrSelfEdge, got %v", err)
	}
}

func TestAddEdgeNodeNotFound(t *testing.T) {
	d := New()
	_ = d.AddNode("A")

	err := d.AddEdge("A", "B")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound for missing 'to', got %v", err)
	}

	err = d.AddEdge("B", "A")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound for missing 'from', got %v", err)
	}
}

func TestAddEdgeCycleDetection(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("C")
	_ = d.AddEdge("A", "B") // A -> B
	_ = d.AddEdge("B", "C") // B -> C

	err := d.AddEdge("C", "A") // C -> A would create cycle
	if !errors.Is(err, ErrCycleDetected) {
		t.Fatalf("expected ErrCycleDetected, got %v", err)
	}
	// Edge should NOT have been added
	if d.EdgeCount() != 2 {
		t.Fatalf("expected 2 edges (cycle edge rejected), got %d", d.EdgeCount())
	}
}

func TestAddEdgeDuplicateIsIdempotent(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddEdge("A", "B")
	err := d.AddEdge("A", "B") // duplicate
	if err != nil {
		t.Fatalf("duplicate edge should be idempotent, got %v", err)
	}
	if d.EdgeCount() != 1 {
		t.Fatalf("expected 1 edge, got %d", d.EdgeCount())
	}
}

// --- RemoveEdge Tests ---

func TestRemoveEdge(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddEdge("A", "B")

	if err := d.RemoveEdge("A", "B"); err != nil {
		t.Fatalf("RemoveEdge unexpected error: %v", err)
	}
	if d.EdgeCount() != 0 {
		t.Fatalf("expected 0 edges, got %d", d.EdgeCount())
	}
}

func TestRemoveEdgeNodeNotFound(t *testing.T) {
	d := New()
	err := d.RemoveEdge("X", "Y")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}
}

// --- HasCycle Tests ---

func TestHasCycleEmpty(t *testing.T) {
	d := New()
	if d.HasCycle() {
		t.Fatal("empty graph should not have a cycle")
	}
}

func TestHasCycleNoCycle(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("C")
	_ = d.AddEdge("A", "B")
	_ = d.AddEdge("B", "C")

	if d.HasCycle() {
		t.Fatal("linear graph should not have a cycle")
	}
}

// --- Nodes / Edges Tests ---

func TestNodes(t *testing.T) {
	d := New()
	_ = d.AddNode("C")
	_ = d.AddNode("A")
	_ = d.AddNode("B")

	nodes := d.Nodes()
	sort.Strings(nodes)
	expected := []string{"A", "B", "C"}
	if len(nodes) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, nodes)
	}
	for i := range expected {
		if nodes[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, nodes)
		}
	}
}

func TestEdgesReturnsDirectDeps(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("C")
	_ = d.AddEdge("A", "B")
	_ = d.AddEdge("A", "C")

	deps, err := d.Edges("A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(deps)
	if len(deps) != 2 || deps[0] != "B" || deps[1] != "C" {
		t.Fatalf("expected [B C], got %v", deps)
	}
}

func TestEdgesNodeNotFound(t *testing.T) {
	d := New()
	_, err := d.Edges("X")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestDependents(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("C")
	_ = d.AddEdge("B", "A") // B depends on A
	_ = d.AddEdge("C", "A") // C depends on A

	deps, err := d.Dependents("A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(deps)
	if len(deps) != 2 || deps[0] != "B" || deps[1] != "C" {
		t.Fatalf("expected [B C], got %v", deps)
	}
}

func TestDependentsNodeNotFound(t *testing.T) {
	d := New()
	_, err := d.Dependents("X")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}
}

// --- TopologicalSort Tests ---

func TestTopologicalSortEmpty(t *testing.T) {
	d := New()
	result, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %v", result)
	}
}

func TestTopologicalSortSingleNode(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	result, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0] != "A" {
		t.Fatalf("expected [A], got %v", result)
	}
}

func TestTopologicalSortLinearChain(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("C")
	_ = d.AddEdge("A", "B") // A depends on B
	_ = d.AddEdge("B", "C") // B depends on C

	result, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 nodes, got %v", result)
	}
	// C must come before B, B before A
	pos := positionMap(result)
	if pos["C"] >= pos["B"] {
		t.Fatalf("C must appear before B, got %v", result)
	}
	if pos["B"] >= pos["A"] {
		t.Fatalf("B must appear before A, got %v", result)
	}
}

func TestTopologicalSortDiamondDependency(t *testing.T) {
	// D depends on B and C; B and C depend on A
	d := New()
	for _, id := range []string{"A", "B", "C", "D"} {
		_ = d.AddNode(id)
	}
	_ = d.AddEdge("D", "B")
	_ = d.AddEdge("D", "C")
	_ = d.AddEdge("B", "A")
	_ = d.AddEdge("C", "A")

	result, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pos := positionMap(result)
	if pos["A"] >= pos["B"] || pos["A"] >= pos["C"] {
		t.Fatalf("A must appear before B and C, got %v", result)
	}
	if pos["B"] >= pos["D"] || pos["C"] >= pos["D"] {
		t.Fatalf("B and C must appear before D, got %v", result)
	}
}

func TestTopologicalSortDisconnectedComponents(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("X")
	_ = d.AddNode("Y")
	_ = d.AddEdge("A", "B")
	_ = d.AddEdge("X", "Y")

	result, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("expected 4 nodes, got %v", result)
	}
	pos := positionMap(result)
	if pos["B"] >= pos["A"] {
		t.Fatalf("B must appear before A")
	}
	if pos["Y"] >= pos["X"] {
		t.Fatalf("Y must appear before X")
	}
}

// --- ComputeWaves Tests ---

func TestComputeWavesEmpty(t *testing.T) {
	d := New()
	waves, err := d.ComputeWaves()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(waves) != 0 {
		t.Fatalf("expected 0 waves, got %d", len(waves))
	}
}

func TestComputeWavesSingleNode(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	waves, err := d.ComputeWaves()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(waves) != 1 {
		t.Fatalf("expected 1 wave, got %d", len(waves))
	}
	if waves[0].Number != 1 {
		t.Fatalf("expected wave number 1, got %d", waves[0].Number)
	}
	if len(waves[0].Nodes) != 1 || waves[0].Nodes[0] != "A" {
		t.Fatalf("expected wave 1 = [A], got %v", waves[0].Nodes)
	}
}

func TestComputeWavesAllIndependent(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("C")

	waves, err := d.ComputeWaves()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(waves) != 1 {
		t.Fatalf("expected 1 wave (all independent), got %d", len(waves))
	}
	if waves[0].Number != 1 {
		t.Fatalf("expected wave number 1, got %d", waves[0].Number)
	}
	sort.Strings(waves[0].Nodes)
	if len(waves[0].Nodes) != 3 {
		t.Fatalf("expected 3 nodes in wave 1, got %v", waves[0].Nodes)
	}
}

func TestComputeWavesLinearChain(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("C")
	_ = d.AddEdge("B", "A") // B depends on A
	_ = d.AddEdge("C", "B") // C depends on B

	waves, err := d.ComputeWaves()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(waves) != 3 {
		t.Fatalf("expected 3 waves, got %d", len(waves))
	}
	assertWave(t, waves[0], 1, []string{"A"})
	assertWave(t, waves[1], 2, []string{"B"})
	assertWave(t, waves[2], 3, []string{"C"})
}

func TestComputeWavesDiamondDependency(t *testing.T) {
	// A is root. B and C depend on A. D depends on B and C.
	d := New()
	for _, id := range []string{"A", "B", "C", "D"} {
		_ = d.AddNode(id)
	}
	_ = d.AddEdge("B", "A")
	_ = d.AddEdge("C", "A")
	_ = d.AddEdge("D", "B")
	_ = d.AddEdge("D", "C")

	waves, err := d.ComputeWaves()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(waves) != 3 {
		t.Fatalf("expected 3 waves, got %d", len(waves))
	}
	assertWave(t, waves[0], 1, []string{"A"})
	assertWaveContains(t, waves[1], 2, []string{"B", "C"})
	assertWave(t, waves[2], 3, []string{"D"})
}

func TestComputeWavesComplexGraph(t *testing.T) {
	// Realistic story dependency graph:
	// STR-001 (root), STR-002 (root)
	// STR-003 depends on STR-001
	// STR-004 depends on STR-001, STR-002
	// STR-005 depends on STR-003, STR-004
	d := New()
	for _, id := range []string{"STR-001", "STR-002", "STR-003", "STR-004", "STR-005"} {
		_ = d.AddNode(id)
	}
	_ = d.AddEdge("STR-003", "STR-001")
	_ = d.AddEdge("STR-004", "STR-001")
	_ = d.AddEdge("STR-004", "STR-002")
	_ = d.AddEdge("STR-005", "STR-003")
	_ = d.AddEdge("STR-005", "STR-004")

	waves, err := d.ComputeWaves()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(waves) != 3 {
		t.Fatalf("expected 3 waves, got %d", len(waves))
	}
	// Wave 1: roots (STR-001, STR-002)
	assertWaveContains(t, waves[0], 1, []string{"STR-001", "STR-002"})
	// Wave 2: STR-003, STR-004
	assertWaveContains(t, waves[1], 2, []string{"STR-003", "STR-004"})
	// Wave 3: STR-005
	assertWave(t, waves[2], 3, []string{"STR-005"})
}

func TestComputeWavesDisconnectedComponents(t *testing.T) {
	d := New()
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("X")
	_ = d.AddNode("Y")
	_ = d.AddEdge("B", "A")
	_ = d.AddEdge("Y", "X")

	waves, err := d.ComputeWaves()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(waves) != 2 {
		t.Fatalf("expected 2 waves, got %d", len(waves))
	}
	// Wave 1: A, X (both roots)
	assertWaveContains(t, waves[0], 1, []string{"A", "X"})
	// Wave 2: B, Y
	assertWaveContains(t, waves[1], 2, []string{"B", "Y"})
}

// --- Thread Safety Tests ---

func TestConcurrentReads(t *testing.T) {
	d := New()
	for i := 0; i < 100; i++ {
		_ = d.AddNode(fmt.Sprintf("N%d", i))
	}
	for i := 1; i < 100; i++ {
		_ = d.AddEdge(fmt.Sprintf("N%d", i), fmt.Sprintf("N%d", i-1))
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = d.TopologicalSort()
			_, _ = d.ComputeWaves()
			_ = d.Nodes()
			_ = d.NodeCount()
			_ = d.EdgeCount()
			_ = d.HasCycle()
		}()
	}
	wg.Wait()
}

// --- Edge Count Tests ---

func TestEdgeCount(t *testing.T) {
	d := New()
	if d.EdgeCount() != 0 {
		t.Fatalf("expected 0 edges, got %d", d.EdgeCount())
	}
	_ = d.AddNode("A")
	_ = d.AddNode("B")
	_ = d.AddNode("C")
	_ = d.AddEdge("A", "B")
	_ = d.AddEdge("A", "C")
	if d.EdgeCount() != 2 {
		t.Fatalf("expected 2 edges, got %d", d.EdgeCount())
	}
}

func TestNodeCount(t *testing.T) {
	d := New()
	if d.NodeCount() != 0 {
		t.Fatalf("expected 0 nodes, got %d", d.NodeCount())
	}
}

// --- Helpers ---

func positionMap(order []string) map[string]int {
	pos := make(map[string]int, len(order))
	for i, id := range order {
		pos[id] = i
	}
	return pos
}

func assertWave(t *testing.T, w Wave, expectedNum int, expectedNodes []string) {
	t.Helper()
	if w.Number != expectedNum {
		t.Fatalf("expected wave %d, got %d", expectedNum, w.Number)
	}
	sort.Strings(w.Nodes)
	sort.Strings(expectedNodes)
	if len(w.Nodes) != len(expectedNodes) {
		t.Fatalf("wave %d: expected nodes %v, got %v", expectedNum, expectedNodes, w.Nodes)
	}
	for i := range expectedNodes {
		if w.Nodes[i] != expectedNodes[i] {
			t.Fatalf("wave %d: expected nodes %v, got %v", expectedNum, expectedNodes, w.Nodes)
		}
	}
}

func assertWaveContains(t *testing.T, w Wave, expectedNum int, expectedNodes []string) {
	t.Helper()
	if w.Number != expectedNum {
		t.Fatalf("expected wave %d, got %d", expectedNum, w.Number)
	}
	sort.Strings(w.Nodes)
	sort.Strings(expectedNodes)
	if len(w.Nodes) != len(expectedNodes) {
		t.Fatalf("wave %d: expected %d nodes %v, got %d nodes %v",
			expectedNum, len(expectedNodes), expectedNodes, len(w.Nodes), w.Nodes)
	}
	for i := range expectedNodes {
		if w.Nodes[i] != expectedNodes[i] {
			t.Fatalf("wave %d: expected nodes %v, got %v", expectedNum, expectedNodes, w.Nodes)
		}
	}
}

