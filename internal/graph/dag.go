// Package graph provides a directed acyclic graph (DAG) with topological
// sorting, ready-node detection, and wave-based grouping for pipeline
// scheduling.
package graph

import "sort"

// DAG is a directed acyclic graph. Nodes are identified by string IDs.
// Edges encode dependency relationships: AddEdge("A", "B") means B depends
// on A (A must complete before B can start).
type DAG struct {
	nodes   map[string]bool     // set of node IDs
	edges   map[string][]string // node -> dependents (outbound edges)
	inbound map[string][]string // node -> dependencies (inbound edges)
}

// NewDAG returns an empty DAG ready for use.
func NewDAG() *DAG {
	return &DAG{
		nodes:   make(map[string]bool),
		edges:   make(map[string][]string),
		inbound: make(map[string][]string),
	}
}

// AddNode registers a node. Duplicate adds are idempotent.
func (d *DAG) AddNode(id string) {
	if d.nodes[id] {
		return
	}
	d.nodes[id] = true
}

// AddEdge adds a directed edge from -> to, meaning "to" depends on "from".
// Both nodes are auto-registered if not already present.
func (d *DAG) AddEdge(from, to string) {
	d.AddNode(from)
	d.AddNode(to)

	// Avoid duplicate edges.
	for _, existing := range d.edges[from] {
		if existing == to {
			return
		}
	}

	d.edges[from] = append(d.edges[from], to)
	d.inbound[to] = append(d.inbound[to], from)
}

// Nodes returns a deterministically sorted slice of all node IDs.
func (d *DAG) Nodes() []string {
	result := make([]string, 0, len(d.nodes))
	for id := range d.nodes {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

// DependenciesOf returns the sorted list of nodes that the given node depends
// on (its inbound edges). Returns nil for unknown nodes.
func (d *DAG) DependenciesOf(id string) []string {
	deps := d.inbound[id]
	if len(deps) == 0 {
		return nil
	}
	out := make([]string, len(deps))
	copy(out, deps)
	sort.Strings(out)
	return out
}

// DependentsOf returns the sorted list of nodes that depend on the given node
// (its outbound edges). Returns nil for unknown nodes.
func (d *DAG) DependentsOf(id string) []string {
	deps := d.edges[id]
	if len(deps) == 0 {
		return nil
	}
	out := make([]string, len(deps))
	copy(out, deps)
	sort.Strings(out)
	return out
}

// HasCycle returns true if the graph contains a cycle, detected via DFS.
func (d *DAG) HasCycle() bool {
	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully explored
	)

	color := make(map[string]int, len(d.nodes))

	var dfs func(node string) bool
	dfs = func(node string) bool {
		color[node] = gray
		for _, neighbor := range d.edges[node] {
			switch color[neighbor] {
			case gray:
				return true // back edge => cycle
			case white:
				if dfs(neighbor) {
					return true
				}
			}
		}
		color[node] = black
		return false
	}

	// Visit nodes in sorted order for determinism.
	for _, node := range d.Nodes() {
		if color[node] == white {
			if dfs(node) {
				return true
			}
		}
	}
	return false
}

// inDegree returns a map of node -> number of inbound edges.
func (d *DAG) inDegree() map[string]int {
	deg := make(map[string]int, len(d.nodes))
	for id := range d.nodes {
		deg[id] = len(d.inbound[id])
	}
	return deg
}
