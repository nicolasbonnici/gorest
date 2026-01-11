package pluginloader

import (
	"fmt"

	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/plugin"
)

type pluginInfo struct {
	name         string
	factory      PluginFactory
	config       config.PluginConfig
	dependencies []string
}

func collectPluginDependencies(configs []config.PluginConfig) ([]pluginInfo, error) {
	var pluginInfos []pluginInfo

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		pluginMutex.RLock()
		factory, exists := pluginFactories[cfg.Name]
		pluginMutex.RUnlock()
		if !exists {
			return nil, fmt.Errorf("unknown plugin '%s' - did you forget to register it?", cfg.Name)
		}

		tempPlugin := factory()
		var deps []string

		if depPlugin, ok := tempPlugin.(plugin.PluginDependencies); ok {
			deps = depPlugin.Dependencies()
			if deps == nil {
				deps = []string{}
			}
		}

		pluginInfos = append(pluginInfos, pluginInfo{
			name:         cfg.Name,
			factory:      factory,
			config:       cfg,
			dependencies: deps,
		})
	}

	return pluginInfos, nil
}

func validateDependencies(pluginInfos []pluginInfo, allConfigs []config.PluginConfig) error {
	available := make(map[string]bool)
	disabled := make(map[string]bool)

	for _, cfg := range allConfigs {
		if cfg.Enabled {
			available[cfg.Name] = true
		} else {
			disabled[cfg.Name] = true
		}
	}

	for _, info := range pluginInfos {
		for _, dep := range info.dependencies {
			if !available[dep] {
				if disabled[dep] {
					return fmt.Errorf(
						"plugin '%s' requires dependency '%s', but it is disabled in configuration",
						info.name, dep,
					)
				}
				return fmt.Errorf(
					"plugin '%s' requires dependency '%s', but it is not registered",
					info.name, dep,
				)
			}
		}
	}

	return nil
}

func resolveInitializationOrder(pluginInfos []pluginInfo) ([]pluginInfo, error) {
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	infoMap := make(map[string]pluginInfo)

	for _, info := range pluginInfos {
		inDegree[info.name] = 0
		infoMap[info.name] = info
		graph[info.name] = []string{}
	}

	for _, info := range pluginInfos {
		for _, dep := range info.dependencies {
			graph[dep] = append(graph[dep], info.name)
			inDegree[info.name]++
		}
	}

	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var sorted []pluginInfo
	visited := 0

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		sorted = append(sorted, infoMap[current])
		visited++

		for _, dependent := range graph[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if visited != len(pluginInfos) {
		return nil, detectCircularDependency(pluginInfos)
	}

	return sorted, nil
}

func detectCircularDependency(pluginInfos []pluginInfo) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	graph := make(map[string][]string)

	for _, info := range pluginInfos {
		graph[info.name] = info.dependencies
	}

	var detectCycle func(string, []string) []string
	detectCycle = func(node string, path []string) []string {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, dep := range graph[node] {
			if !visited[dep] {
				if cycle := detectCycle(dep, path); cycle != nil {
					return cycle
				}
			} else if recStack[dep] {
				cycleStart := -1
				for i, n := range path {
					if n == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					return append(path[cycleStart:], dep)
				}
			}
		}

		recStack[node] = false
		return nil
	}

	for _, info := range pluginInfos {
		if !visited[info.name] {
			if cycle := detectCycle(info.name, []string{}); cycle != nil {
				cycleStr := ""
				for i, name := range cycle {
					if i > 0 {
						cycleStr += " -> "
					}
					cycleStr += name
				}
				return fmt.Errorf("circular dependency detected: %s", cycleStr)
			}
		}
	}

	return fmt.Errorf("circular dependency detected in plugins")
}
