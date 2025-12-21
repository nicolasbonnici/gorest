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

func NewDependencyResolver() *DependencyResolver {
	return &DependencyResolver{
		sources: make(map[string]SourceDependency),
	}
}

func (r *DependencyResolver) AddSource(name string, dependencies []string) {
	r.sources[name] = SourceDependency{
		Name:         name,
		Dependencies: dependencies,
	}
}

// Resolve performs topological sort using Kahn's algorithm to determine execution order.
func (r *DependencyResolver) Resolve() ([]string, error) {
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	for name := range r.sources {
		inDegree[name] = 0
		adjList[name] = []string{}
	}

	for name, dep := range r.sources {
		for _, dependency := range dep.Dependencies {
			if _, exists := r.sources[dependency]; !exists {
				continue
			}

			adjList[dependency] = append(adjList[dependency], name)
			inDegree[name]++
		}
	}

	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	sort.Strings(queue)

	var result []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		result = append(result, current)

		neighbors := adjList[current]
		sort.Strings(neighbors)

		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue)
			}
		}
	}

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

// OrderMigrations orders migrations by source dependencies first, then by version timestamp.
func (r *DependencyResolver) OrderMigrations(migrations []Migration) ([]Migration, error) {
	sourceOrder, err := r.Resolve()
	if err != nil {
		return nil, err
	}

	sourcePriority := make(map[string]int)
	for i, source := range sourceOrder {
		sourcePriority[source] = i
	}

	sorted := make([]Migration, len(migrations))
	copy(sorted, migrations)

	sort.Slice(sorted, func(i, j int) bool {
		priI := sourcePriority[sorted[i].Source]
		priJ := sourcePriority[sorted[j].Source]

		if priI != priJ {
			return priI < priJ
		}

		return sorted[i].Version < sorted[j].Version
	})

	return sorted, nil
}
