package graph

import (
	"fmt"
	"sort"
)

// TopoSort returns nodes in topological order using Kahn's algorithm.
// Returns an error if the graph contains a cycle.
func TopoSort(d *DAG) ([]string, error) {
	if len(d.nodes) == 0 {
		return nil, nil
	}

	inDeg := d.inDegree()

	// Seed the queue with all zero-in-degree nodes, sorted for determinism.
	queue := make([]string, 0)
	for _, node := range d.Nodes() {
		if inDeg[node] == 0 {
			queue = append(queue, node)
		}
	}

	result := make([]string, 0, len(d.nodes))

	for len(queue) > 0 {
		// Sort queue each iteration for deterministic output.
		sort.Strings(queue)

		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// "Remove" this node by decrementing in-degree of its dependents.
		dependents := d.DependentsOf(node)
		for _, dep := range dependents {
			inDeg[dep]--
			if inDeg[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(result) != len(d.nodes) {
		return nil, fmt.Errorf("graph contains a cycle: processed %d of %d nodes", len(result), len(d.nodes))
	}

	return result, nil
}

// ReadyNodes returns nodes whose dependencies are all in the completed set,
// excluding nodes already completed. The result is sorted for determinism.
func ReadyNodes(d *DAG, completed map[string]bool) []string {
	ready := make([]string, 0)

	for _, node := range d.Nodes() {
		if completed[node] {
			continue
		}

		allDepsComplete := true
		for _, dep := range d.DependenciesOf(node) {
			if !completed[dep] {
				allDepsComplete = false
				break
			}
		}

		if allDepsComplete {
			ready = append(ready, node)
		}
	}

	sort.Strings(ready)
	return ready
}

// GroupByWave returns nodes grouped into execution waves using a modified
// Kahn's algorithm (BFS layer-by-layer). Wave 0 contains root nodes (no
// dependencies), wave 1 contains nodes depending only on wave 0, etc.
// Returns an error if the graph contains a cycle.
func GroupByWave(d *DAG) ([][]string, error) {
	if len(d.nodes) == 0 {
		return nil, nil
	}

	inDeg := d.inDegree()

	// Find initial roots (in-degree 0).
	currentWave := make([]string, 0)
	for _, node := range d.Nodes() {
		if inDeg[node] == 0 {
			currentWave = append(currentWave, node)
		}
	}

	waves := make([][]string, 0)
	processed := 0

	for len(currentWave) > 0 {
		sort.Strings(currentWave)
		waves = append(waves, currentWave)
		processed += len(currentWave)

		nextWave := make([]string, 0)
		for _, node := range currentWave {
			for _, dep := range d.DependentsOf(node) {
				inDeg[dep]--
				if inDeg[dep] == 0 {
					nextWave = append(nextWave, dep)
				}
			}
		}
		currentWave = nextWave
	}

	if processed != len(d.nodes) {
		return nil, fmt.Errorf("graph contains a cycle: processed %d of %d nodes", processed, len(d.nodes))
	}

	return waves, nil
}
