// Package graph provides a directed acyclic graph (DAG) with topological
// sorting and wave computation for story dependency management.
package graph

import (
	"fmt"
	"sort"
	"sync"
)

// ErrCycleDetected is returned when adding an edge would create a cycle.
var ErrCycleDetected = fmt.Errorf("cycle detected in graph")

// ErrNodeNotFound is returned when referencing a node that doesn't exist.
var ErrNodeNotFound = fmt.Errorf("node not found")

// ErrDuplicateNode is returned when adding a node that already exists.
var ErrDuplicateNode = fmt.Errorf("duplicate node")

// ErrSelfEdge is returned when adding an edge from a node to itself.
var ErrSelfEdge = fmt.Errorf("self-referencing edge not allowed")

// DAG is a thread-safe directed acyclic graph keyed by string node IDs.
// Edge semantics: edges[from][to] means "from depends on to".
type DAG struct {
	mu      sync.RWMutex
	nodes   map[string]bool
	edges   map[string]map[string]bool // forward: from → {to, ...}
	reverse map[string]map[string]bool // backward: to → {from, ...}
}

// Wave represents a group of nodes that can be processed in parallel.
type Wave struct {
	Number int
	Nodes  []string
}

// New creates an empty DAG.
func New() *DAG {
	return &DAG{
		nodes:   make(map[string]bool),
		edges:   make(map[string]map[string]bool),
		reverse: make(map[string]map[string]bool),
	}
}

// AddNode adds a node to the graph. Returns ErrDuplicateNode if it exists.
func (d *DAG) AddNode(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.nodes[id] {
		return ErrDuplicateNode
	}
	d.nodes[id] = true
	return nil
}

// RemoveNode removes a node and all its edges. Returns ErrNodeNotFound if missing.
func (d *DAG) RemoveNode(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.nodes[id] {
		return ErrNodeNotFound
	}

	// Remove forward edges from this node
	for to := range d.edges[id] {
		delete(d.reverse[to], id)
		if len(d.reverse[to]) == 0 {
			delete(d.reverse, to)
		}
	}
	delete(d.edges, id)

	// Remove reverse edges pointing to this node
	for from := range d.reverse[id] {
		delete(d.edges[from], id)
		if len(d.edges[from]) == 0 {
			delete(d.edges, from)
		}
	}
	delete(d.reverse, id)

	delete(d.nodes, id)
	return nil
}

// AddEdge adds a directed edge from → to (meaning "from" depends on "to").
// Returns ErrNodeNotFound if either node is missing, ErrSelfEdge for self-loops,
// or ErrCycleDetected if the edge would create a cycle.
// Adding a duplicate edge is a no-op.
func (d *DAG) AddEdge(from, to string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if from == to {
		return ErrSelfEdge
	}
	if !d.nodes[from] {
		return ErrNodeNotFound
	}
	if !d.nodes[to] {
		return ErrNodeNotFound
	}

	// Duplicate check — idempotent
	if d.edges[from] != nil && d.edges[from][to] {
		return nil
	}

	// Cycle check: adding from→to creates a cycle if there's a path from to→from
	if d.hasPath(to, from) {
		return ErrCycleDetected
	}

	if d.edges[from] == nil {
		d.edges[from] = make(map[string]bool)
	}
	d.edges[from][to] = true

	if d.reverse[to] == nil {
		d.reverse[to] = make(map[string]bool)
	}
	d.reverse[to][from] = true

	return nil
}

// hasPath checks if there's a directed path from src to dst using BFS.
// Must be called with d.mu held.
func (d *DAG) hasPath(src, dst string) bool {
	visited := make(map[string]bool)
	queue := []string{src}
	visited[src] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == dst {
			return true
		}

		for next := range d.edges[current] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

// RemoveEdge removes an edge. Returns ErrNodeNotFound if either node is missing.
func (d *DAG) RemoveEdge(from, to string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.nodes[from] || !d.nodes[to] {
		return ErrNodeNotFound
	}

	if d.edges[from] != nil {
		delete(d.edges[from], to)
		if len(d.edges[from]) == 0 {
			delete(d.edges, from)
		}
	}
	if d.reverse[to] != nil {
		delete(d.reverse[to], from)
		if len(d.reverse[to]) == 0 {
			delete(d.reverse, to)
		}
	}

	return nil
}

// HasCycle returns true if the graph contains a cycle.
// Uses Kahn's algorithm: if topological sort doesn't include all nodes, a cycle exists.
func (d *DAG) HasCycle() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	sorted := d.kahnSort()
	return len(sorted) != len(d.nodes)
}

// Nodes returns all node IDs in the graph (sorted for deterministic output).
func (d *DAG) Nodes() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]string, 0, len(d.nodes))
	for id := range d.nodes {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

// Edges returns the direct dependencies of a node (the nodes it points to).
func (d *DAG) Edges(id string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.nodes[id] {
		return nil, ErrNodeNotFound
	}

	result := make([]string, 0, len(d.edges[id]))
	for to := range d.edges[id] {
		result = append(result, to)
	}
	sort.Strings(result)
	return result, nil
}

// Dependents returns nodes that depend on the given node.
func (d *DAG) Dependents(id string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.nodes[id] {
		return nil, ErrNodeNotFound
	}

	result := make([]string, 0, len(d.reverse[id]))
	for from := range d.reverse[id] {
		result = append(result, from)
	}
	sort.Strings(result)
	return result, nil
}

// TopologicalSort returns nodes in topological order using Kahn's algorithm.
// Dependencies appear before dependents. Returns ErrCycleDetected if cyclic.
func (d *DAG) TopologicalSort() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	sorted := d.kahnSort()
	if len(sorted) != len(d.nodes) {
		return nil, ErrCycleDetected
	}
	return sorted, nil
}

// ComputeWaves groups nodes into waves where wave N contains nodes whose
// dependencies are all in waves < N. Wave 1 contains root nodes (no deps).
// Returns ErrCycleDetected if the graph has a cycle.
func (d *DAG) ComputeWaves() ([]Wave, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.nodes) == 0 {
		return nil, nil
	}

	// Build dependency count: how many unresolved deps each node has
	depCount := make(map[string]int, len(d.nodes))
	for id := range d.nodes {
		depCount[id] = len(d.edges[id])
	}

	// Seed with root nodes (zero dependencies)
	var current []string
	for id, count := range depCount {
		if count == 0 {
			current = append(current, id)
		}
	}

	var waves []Wave
	processed := 0

	for len(current) > 0 {
		sort.Strings(current) // deterministic wave ordering
		wave := Wave{
			Number: len(waves) + 1,
			Nodes:  current,
		}
		waves = append(waves, wave)
		processed += len(current)

		// Find next wave: for each processed node, decrement dep count
		// of nodes that depend on it
		var next []string
		for _, id := range current {
			for dep := range d.reverse[id] {
				depCount[dep]--
				if depCount[dep] == 0 {
					next = append(next, dep)
				}
			}
		}
		current = next
	}

	if processed != len(d.nodes) {
		return nil, ErrCycleDetected
	}

	return waves, nil
}

// NodeCount returns the number of nodes in the graph.
func (d *DAG) NodeCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.nodes)
}

// EdgeCount returns the number of edges in the graph.
func (d *DAG) EdgeCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	count := 0
	for _, targets := range d.edges {
		count += len(targets)
	}
	return count
}

// kahnSort performs Kahn's algorithm and returns the sorted nodes.
// Must be called with d.mu held (at least RLock).
func (d *DAG) kahnSort() []string {
	// Build in-degree map (count of forward edges pointing into each node)
	inDegree := make(map[string]int, len(d.nodes))
	for id := range d.nodes {
		inDegree[id] = len(d.edges[id])
	}

	// Seed queue with nodes that have no outgoing dependencies
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var sorted []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)

		// For each node that depends on this one, reduce its in-degree
		for dep := range d.reverse[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
				sort.Strings(queue) // maintain deterministic order
			}
		}
	}

	return sorted
}
