package bundle

// Scaffolding for customization and library pack handling. The simplified
// Create flow (see create.go) only processes query-kind packs; ResolveDeps
// and BuildProcessingOrder are retained here so the customization and library
// stubs in process_stubs.go can adopt them when implemented.

import (
	"fmt"

	"github.com/trganda/codeql-development-toolkit/internal/pack"
)

// versionMatches returns true if candidate satisfies the version spec.
// Handles "*", exact versions, and range specs (^, >=) by accepting any candidate
// — CodeQL workspaces typically contain one version per pack name.
func versionMatches(spec, candidate string) bool {
	if spec == "*" || spec == "" {
		return true
	}
	if spec == candidate {
		return true
	}
	// Accept range specs (^X.Y.Z, >=X.Y.Z, etc.) without strict range checking.
	if len(spec) > 0 && (spec[0] == '^' || spec[0] == '>' || spec[0] == '~') {
		return true
	}
	return false
}

// ResolveDeps populates the Deps field on each workspace pack by looking up
// dependencies in the combined allPacks slice (workspace + bundle packs).
func ResolveDeps(workspacePacks, allPacks []*pack.Pack) error {
	byName := make(map[string]*pack.Pack, len(allPacks))
	for _, p := range allPacks {
		byName[p.Config.Name] = p
	}
	for _, wp := range workspacePacks {
		for depName, depSpec := range wp.Config.Dependencies {
			candidate, ok := byName[depName]
			if !ok {
				return fmt.Errorf("pack %s: dependency %q not found in bundle or workspace", wp.Config.Name, depName)
			}
			if !versionMatches(depSpec, candidate.Config.Version) {
				return fmt.Errorf("pack %s: dependency %s@%s not satisfied by available version %s",
					wp.Config.Name, depName, depSpec, candidate.Config.Version)
			}
			wp.Deps = append(wp.Deps, candidate)
		}
	}
	return nil
}

// BuildProcessingOrder determines which packs must be processed and in what order.
//
// It returns:
//   - The topologically-sorted list of packs to process (workspace packs +
//     any stdlib lib/query packs that must be re-bundled due to customizations).
//   - A map from each stdlib lib pack to the customization packs that extend it.
// func BuildProcessingOrder(workspacePacks, bundlePacks []*Pack) ([]*Pack, map[*Pack][]*Pack, error) {
// 	// Track customization pack → stdlib lib pack mappings.
// 	stdlibCustomizations := make(map[*Pack][]*Pack) // stdlib lib → []customization packs

// 	// Collect all packs that need topological sorting.
// 	// Key: pack; Value: set of packs that must come before it.
// 	graph := make(map[*Pack]map[*Pack]struct{})

// 	var addToGraph func(p *Pack, visited map[*Pack]bool) error
// 	addToGraph = func(p *Pack, visited map[*Pack]bool) error {
// 		if visited[p] {
// 			return nil
// 		}
// 		if _, exists := graph[p]; !exists {
// 			graph[p] = make(map[*Pack]struct{})
// 		}

// 		if p.Kind == CustomizationPack {
// 			// Customization packs have no ordering deps beyond themselves.
// 			// Record which stdlib lib pack they customize (their first/only dep).
// 			if len(p.Deps) > 0 {
// 				stdlibLib := p.Deps[0]
// 				stdlibCustomizations[stdlibLib] = append(stdlibCustomizations[stdlibLib], p)
// 			}
// 		} else {
// 			// Library and query workspace packs depend on their deps.
// 			for _, dep := range p.Deps {
// 				graph[p][dep] = struct{}{}
// 				if err := addToGraph(dep, visited); err != nil {
// 					return err
// 				}
// 			}
// 		}
// 		visited[p] = true
// 		return nil
// 	}

// 	visited := make(map[*Pack]bool)
// 	for _, wp := range workspacePacks {
// 		if err := addToGraph(wp, visited); err != nil {
// 			return nil, nil, err
// 		}
// 	}

// 	// For each customized stdlib lib pack: it depends on its customization packs
// 	// (must process them first), and any stdlib query packs that transitively
// 	// depend on it must also be processed after.
// 	isDependent := func(pack, target *Pack) bool {
// 		seen := make(map[*Pack]bool)
// 		var check func(p *Pack) bool
// 		check = func(p *Pack) bool {
// 			if seen[p] {
// 				return false
// 			}
// 			seen[p] = true
// 			for _, d := range p.Deps {
// 				if d == target || check(d) {
// 					return true
// 				}
// 			}
// 			return false
// 		}
// 		return check(pack)
// 	}

// 	for stdlibLib, customizationPacks := range stdlibCustomizations {
// 		if _, exists := graph[stdlibLib]; !exists {
// 			graph[stdlibLib] = make(map[*Pack]struct{})
// 		}
// 		for _, cp := range customizationPacks {
// 			// stdlib lib must come after its customization packs.
// 			graph[stdlibLib][cp] = struct{}{}
// 		}
// 		// Find stdlib query packs in the bundle that depend on this stdlib lib pack.
// 		for _, bp := range bundlePacks {
// 			if bp.Kind == QueryPack && bp.Config.Scope() == "codeql" && isDependent(bp, stdlibLib) {
// 				if _, exists := graph[bp]; !exists {
// 					graph[bp] = make(map[*Pack]struct{})
// 				}
// 				// stdlib query pack must come after the stdlib lib pack.
// 				graph[bp][stdlibLib] = struct{}{}
// 			}
// 		}
// 	}

// 	// Kahn's algorithm for topological sort.
// 	inDegree := make(map[*Pack]int, len(graph))
// 	for p := range graph {
// 		if _, ok := inDegree[p]; !ok {
// 			inDegree[p] = 0
// 		}
// 		for dep := range graph[p] {
// 			inDegree[dep] = inDegree[dep] // ensure dep is in inDegree
// 			_ = dep
// 		}
// 	}
// 	// Initialize inDegree for all nodes.
// 	for p := range graph {
// 		inDegree[p] = inDegree[p]
// 	}
// 	for p := range graph {
// 		for dep := range graph[p] {
// 			_ = dep
// 		}
// 	}
// 	// Recompute properly: inDegree[p] = number of packs that must come AFTER p, i.e.
// 	// the reverse: count how many predecessors each pack has.
// 	// graph[p][dep] means p depends on dep → dep must come before p → dep is a predecessor of p.
// 	predCount := make(map[*Pack]int, len(graph))
// 	for p := range graph {
// 		if _, ok := predCount[p]; !ok {
// 			predCount[p] = 0
// 		}
// 		for dep := range graph[p] {
// 			if _, ok := predCount[dep]; !ok {
// 				predCount[dep] = 0
// 			}
// 		}
// 	}
// 	// successors: for each dep, which packs depend on it
// 	successors := make(map[*Pack][]*Pack)
// 	for p := range graph {
// 		for dep := range graph[p] {
// 			predCount[p]++
// 			successors[dep] = append(successors[dep], p)
// 		}
// 	}

// 	// Collect packs with no predecessors.
// 	var queue []*Pack
// 	for p := range predCount {
// 		if predCount[p] == 0 {
// 			queue = append(queue, p)
// 		}
// 	}

// 	var order []*Pack
// 	for len(queue) > 0 {
// 		p := queue[0]
// 		queue = queue[1:]
// 		order = append(order, p)
// 		for _, succ := range successors[p] {
// 			predCount[succ]--
// 			if predCount[succ] == 0 {
// 				queue = append(queue, succ)
// 			}
// 		}
// 	}

// 	if len(order) != len(predCount) {
// 		return nil, nil, fmt.Errorf("circular dependency detected in pack graph")
// 	}

// 	return order, stdlibCustomizations, nil
// }
