package migrations

import (
	"fmt"
	"sort"
)

// SourceDependency represents a source and its dependencies
type SourceDependency struct {
	Name         string
	Dependencies []string
}

// DependencyResolver resolves migration execution order based on dependencies
type DependencyResolver struct {
	sources map[string]SourceDependency
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver() *DependencyResolver {
	return &DependencyResolver{
		sources: make(map[string]SourceDependency),
	}
}

// AddSource registers a source with its dependencies
func (r *DependencyResolver) AddSource(name string, dependencies []string) {
	r.sources[name] = SourceDependency{
		Name:         name,
		Dependencies: dependencies,
	}
}

// Resolve performs topological sort to determine execution order
func (r *DependencyResolver) Resolve() ([]string, error) {
	// Build adjacency list and in-degree map
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize all sources with in-degree 0
	for name := range r.sources {
		inDegree[name] = 0
		adjList[name] = []string{}
	}

	// Build graph
	for name, dep := range r.sources {
		for _, dependency := range dep.Dependencies {
			// Check if dependency exists
			if _, exists := r.sources[dependency]; !exists {
				// Dependency not found - this might be OK if it's an external dependency
				// For now, we'll skip it
				continue
			}

			// Add edge from dependency to dependent
			adjList[dependency] = append(adjList[dependency], name)
			inDegree[name]++
		}
	}

	// Kahn's algorithm for topological sort
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort queue for deterministic ordering
	sort.Strings(queue)

	var result []string

	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]

		result = append(result, current)

		// Reduce in-degree for neighbors
		neighbors := adjList[current]
		sort.Strings(neighbors) // For deterministic ordering

		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue) // Keep queue sorted
			}
		}
	}

	// Check for circular dependencies
	if len(result) != len(r.sources) {
		return nil, r.findCircularDependency()
	}

	return result, nil
}

// findCircularDependency detects and reports circular dependencies
func (r *DependencyResolver) findCircularDependency() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(string, []string) []string
	dfs = func(node string, path []string) []string {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		if deps, exists := r.sources[node]; exists {
			for _, dep := range deps.Dependencies {
				if !visited[dep] {
					if cycle := dfs(dep, path); cycle != nil {
						return cycle
					}
				} else if recStack[dep] {
					// Found cycle
					return append(path, dep)
				}
			}
		}

		recStack[node] = false
		return nil
	}

	for name := range r.sources {
		if !visited[name] {
			if cycle := dfs(name, []string{}); cycle != nil {
				return fmt.Errorf("%w: %v", ErrCircularDependency, cycle)
			}
		}
	}

	return ErrCircularDependency
}

// OrderMigrations orders migrations based on source dependencies and version
func (r *DependencyResolver) OrderMigrations(migrations []Migration) ([]Migration, error) {
	// Get source execution order
	sourceOrder, err := r.Resolve()
	if err != nil {
		return nil, err
	}

	// Create map of source to priority
	sourcePriority := make(map[string]int)
	for i, source := range sourceOrder {
		sourcePriority[source] = i
	}

	// Sort migrations by:
	// 1. Source priority (based on dependencies)
	// 2. Version (timestamp) within same priority
	sorted := make([]Migration, len(migrations))
	copy(sorted, migrations)

	sort.Slice(sorted, func(i, j int) bool {
		priI := sourcePriority[sorted[i].Source]
		priJ := sourcePriority[sorted[j].Source]

		// If different priorities, sort by priority
		if priI != priJ {
			return priI < priJ
		}

		// Same priority, sort by version (timestamp)
		return sorted[i].Version < sorted[j].Version
	})

	return sorted, nil
}
