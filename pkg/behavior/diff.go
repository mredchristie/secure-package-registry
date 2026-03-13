package behavior

import (
	"fmt"
)

// Dedupe removes baseline behaviors from a target tree, returning only
// novel/suspicious behaviors. It returns a new tree containing only the
// nodes and behaviors that exist in target but not in baseline. The
// comparison is structural — process identities are matched by their
// ancestry path (name-based, not PID-based).
//
// The algorithm:
//  1. Flatten both trees into maps keyed by process name path (ignoring ordinals)
//  2. For each target process, subtract baseline behaviors
//  3. Keep only processes that have remaining behaviors or novel children
func Dedupe(target, baseline *ProcessTree) *ProcessTree {
	// Build lookup of baseline behaviors keyed by normalized identity
	// (strip ordinals so "node(1)" and "node(2)" both map to the same baseline).
	baselineBehaviors := flattenBehaviors(baseline.Root)

	// Deep copy the target tree and subtract baseline behaviors.
	result := &ProcessTree{
		Root: dedupeNode(target.Root, baselineBehaviors),
	}

	// Prune empty leaves.
	pruneEmpty(result.Root)

	return result
}

// normalizeIdentity strips ordinal numbers from an identity string.
// "root -> sh -> node(1) -> curl" becomes "root -> sh -> node -> curl".
func normalizeIdentity(identity string) string {
	// Fast path: no parentheses means no ordinals.
	result := make([]byte, 0, len(identity))
	i := 0
	for i < len(identity) {
		if identity[i] == '(' {
			// Skip until closing paren.
			j := i + 1
			for j < len(identity) && identity[j] != ')' {
				j++
			}
			if j < len(identity) {
				i = j + 1 // skip past ')'
				continue
			}
		}
		result = append(result, identity[i])
		i++
	}
	return string(result)
}

// behaviorSet holds sets for fast lookup during diffing.
type behaviorSet struct {
	files       map[string]bool
	connections map[string]bool
	dns         map[string]bool
	execs       map[string]bool
}

// flattenBehaviors walks the tree and returns a map of normalized identity
// to the union of all behaviors seen at that identity across all ordinals.
func flattenBehaviors(node *ProcessNode) map[string]*behaviorSet {
	result := make(map[string]*behaviorSet)
	flattenNode(node, result)
	return result
}

func flattenNode(node *ProcessNode, result map[string]*behaviorSet) {
	normID := normalizeIdentity(node.Identity)

	bs, ok := result[normID]
	if !ok {
		bs = &behaviorSet{
			files:       make(map[string]bool),
			connections: make(map[string]bool),
			dns:         make(map[string]bool),
			execs:       make(map[string]bool),
		}
		result[normID] = bs
	}

	for _, f := range node.Behaviors.FilesOpened {
		bs.files[f] = true
	}
	for _, c := range node.Behaviors.Connections {
		bs.connections[fmt.Sprintf("%s:%d", c.Addr, c.Port)] = true
	}
	for _, d := range node.Behaviors.DNSQueries {
		bs.dns[d] = true
	}
	for _, e := range node.Behaviors.Execs {
		bs.execs[e.Pathname] = true
	}

	for _, child := range node.Children {
		flattenNode(child, result)
	}
}

// dedupeNode creates a deep copy of target with baseline behaviors subtracted.
func dedupeNode(target *ProcessNode, baseline map[string]*behaviorSet) *ProcessNode {
	normID := normalizeIdentity(target.Identity)
	bs := baseline[normID] // may be nil if target process doesn't exist in baseline

	result := &ProcessNode{
		Name:     target.Name,
		Identity: target.Identity,
	}

	// Subtract behaviors.
	if bs != nil {
		for _, e := range target.Behaviors.Execs {
			if !bs.execs[e.Pathname] {
				result.Behaviors.Execs = append(result.Behaviors.Execs, e)
			}
		}
		for _, f := range target.Behaviors.FilesOpened {
			if !bs.files[f] {
				result.Behaviors.FilesOpened = append(result.Behaviors.FilesOpened, f)
			}
		}
		for _, c := range target.Behaviors.Connections {
			key := fmt.Sprintf("%s:%d", c.Addr, c.Port)
			if !bs.connections[key] {
				result.Behaviors.Connections = append(result.Behaviors.Connections, c)
			}
		}
		for _, d := range target.Behaviors.DNSQueries {
			if !bs.dns[d] {
				result.Behaviors.DNSQueries = append(result.Behaviors.DNSQueries, d)
			}
		}
	} else {
		// No baseline match — keep everything.
		result.Behaviors = target.Behaviors
	}

	// Recurse into children.
	for _, child := range target.Children {
		diffChild := dedupeNode(child, baseline)
		result.Children = append(result.Children, diffChild)
	}

	return result
}

// pruneEmpty removes nodes that have no behaviors and no children with behaviors.
// Returns true if the node itself should be pruned.
func pruneEmpty(node *ProcessNode) bool {
	// Prune children first (bottom-up).
	kept := node.Children[:0]
	for _, child := range node.Children {
		if !pruneEmpty(child) {
			kept = append(kept, child)
		}
	}
	node.Children = kept

	// A node is empty if it has no behaviors and no children.
	return node.Behaviors.IsEmpty() && len(node.Children) == 0
}
