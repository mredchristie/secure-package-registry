package behavior

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ProcessNode represents a single process in the ancestry tree with its
// observed behaviors. Children are keyed by their ordinal occurrence name
// (e.g., "node(1)", "node(2)").
type ProcessNode struct {
	// Name is the final executable name (e.g. "node", "curl", "npm").
	Name string `json:"name"`

	// Identity is the full ancestry path used as a stable key
	// (e.g., "root -> sh -> node -> curl").
	Identity string `json:"identity"`

	// Behaviors contains the categorized events observed for this process.
	Behaviors ProcessBehaviors `json:"behaviors"`

	// Children maps ordinal names to child nodes, ordered by first appearance.
	Children []*ProcessNode `json:"children,omitempty"`
}

// ProcessBehaviors holds deduplicated, categorized events for a single process.
type ProcessBehaviors struct {
	// Execs are the execve calls this process made (spawning children).
	Execs []ExecBehavior `json:"execs,omitempty"`

	// FilesOpened is the set of unique file paths opened by this process.
	FilesOpened []string `json:"files_opened,omitempty"`

	// Connections are unique outbound network connections.
	Connections []ConnectionBehavior `json:"connections,omitempty"`

	// DNSQueries are unique DNS queries made by this process.
	DNSQueries []string `json:"dns_queries,omitempty"`
}

// ExecBehavior records a process execution.
type ExecBehavior struct {
	Pathname string   `json:"pathname"`
	Argv     []string `json:"argv"`
}

// ConnectionBehavior records an outbound connection.
type ConnectionBehavior struct {
	Addr string `json:"addr"`
	Port uint16 `json:"port"`
}

// IsEmpty returns true if the behaviors contain no events.
func (b *ProcessBehaviors) IsEmpty() bool {
	return len(b.Execs) == 0 &&
		len(b.FilesOpened) == 0 &&
		len(b.Connections) == 0 &&
		len(b.DNSQueries) == 0
}

// ProcessTree is the complete behavioral profile of a package analysis run.
type ProcessTree struct {
	// Root is the virtual root node whose children are the container entry
	// points (runc init processes with ppid=0).
	Root *ProcessNode `json:"root"`
}

// IsEmpty returns true if the tree contains no behavioral events.
// After Dedupe + pruneEmpty, a clean tree has a root with empty behaviors
// and no remaining children.
func (t *ProcessTree) IsEmpty() bool {
	if t.Root == nil {
		return true
	}
	return t.Root.Behaviors.IsEmpty() && len(t.Root.Children) == 0
}

// DefaultFileIgnorePrefixes are file path prefixes that produce
// non-deterministic noise across runs (cache hashes, timestamps, etc.)
// and carry no meaningful behavioral signal.
var DefaultFileIgnorePrefixes = []string{
	"/root/.npm/_cacache/",
	"/root/.npm/_logs/",
	"/root/.node_modules/",
	"/test/node_modules/",
}

// BuildTreeOptions controls filtering during tree construction.
type BuildTreeOptions struct {
	// FileIgnorePrefixes is a list of path prefixes to drop from file-open
	// events. If nil, DefaultFileIgnorePrefixes is used. Set to an empty
	// slice to disable filtering.
	FileIgnorePrefixes []string
}

// BuildTree parses a behavior JSONL stream and constructs the process tree.
// Events must be in chronological order (which JSONL naturally provides).
// If opts is nil, default filtering is applied.
func BuildTree(r io.Reader, opts *BuildTreeOptions) (*ProcessTree, error) {
	ignorePrefixes := DefaultFileIgnorePrefixes
	if opts != nil {
		ignorePrefixes = opts.FileIgnorePrefixes
	}
	// Phase 1: Parse all events and track PID lifecycle.
	// A PID can change names via execve, so we track the final name.
	type pidInfo struct {
		ppid      int
		names     []string // ordered list of names this PID had
		finalName string
	}

	pids := make(map[int]*pidInfo)
	var events []*ParsedEvent

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for long lines

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw RawEvent
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		parsed := ParseEvent(&raw)
		if parsed == nil {
			continue
		}

		// Track PID info. The processName field reflects the current name
		// at the time of the event, which changes after each execve.
		info, ok := pids[parsed.PID]
		if !ok {
			info = &pidInfo{ppid: parsed.PPID}
			pids[parsed.PID] = info
		}
		info.finalName = parsed.Name
		if len(info.names) == 0 || info.names[len(info.names)-1] != parsed.Name {
			info.names = append(info.names, parsed.Name)
		}

		events = append(events, parsed)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning input: %w", err)
	}

	// Phase 2: Determine the final name for each PID.
	// The "useful" name is the last non-runc name.
	pidFinalName := make(map[int]string)
	for pid, info := range pids {
		name := info.finalName
		// Skip runc init names — use the first real name
		if strings.HasPrefix(name, "runc:") {
			// This PID never exec'd into something else
			name = "init"
		}
		pidFinalName[pid] = name
	}

	// Phase 3: Build the identity paths.
	// Identity = parent identity + " -> " + process name.
	// For multiple children with the same name under one parent, we number them.

	// First, determine the tree structure by parent-child relationships.
	type childKey struct {
		parentPID int
		name      string
	}
	childCounts := make(map[childKey]int) // how many children with this name under this parent
	childIndex := make(map[int]int)       // PID -> which occurrence number it is

	// Group PIDs by parent, ordered by first appearance (lowest PID = first seen).
	parentChildren := make(map[int][]int)
	for pid, info := range pids {
		parentChildren[info.ppid] = append(parentChildren[info.ppid], pid)
	}
	// Sort children by PID to get stable ordering.
	for ppid := range parentChildren {
		sort.Ints(parentChildren[ppid])
	}

	// Assign ordinal indices to each child.
	for _, children := range parentChildren {
		for _, pid := range children {
			name := pidFinalName[pid]
			key := childKey{parentPID: pids[pid].ppid, name: name}
			childCounts[key]++
			childIndex[pid] = childCounts[key]
		}
	}

	// Build identity strings recursively.
	identities := make(map[int]string)
	var buildIdentity func(pid int) string
	buildIdentity = func(pid int) string {
		if id, ok := identities[pid]; ok {
			return id
		}
		info, ok := pids[pid]
		if !ok {
			return "root"
		}
		name := pidFinalName[pid]
		parentID := "root"
		if info.ppid != 0 {
			parentID = buildIdentity(info.ppid)
		}

		// Only add ordinal if there are multiple siblings with the same name.
		key := childKey{parentPID: info.ppid, name: name}
		nodeName := name
		if childCounts[key] > 1 {
			nodeName = fmt.Sprintf("%s(%d)", name, childIndex[pid])
		}

		identity := parentID + " -> " + nodeName
		identities[pid] = identity
		return identity
	}

	for pid := range pids {
		buildIdentity(pid)
	}

	// Phase 4: Create process nodes and collect behaviors.
	nodes := make(map[string]*ProcessNode) // identity -> node
	getNode := func(pid int) *ProcessNode {
		id := identities[pid]
		if n, ok := nodes[id]; ok {
			return n
		}
		n := &ProcessNode{
			Name:     pidFinalName[pid],
			Identity: id,
		}
		nodes[id] = n
		return n
	}

	// Dedup sets per node for within-run deduplication.
	type dedupSets struct {
		files       map[string]bool
		connections map[string]bool
		dns         map[string]bool
		execs       map[string]bool
	}
	nodeDedup := make(map[string]*dedupSets)
	getDedup := func(id string) *dedupSets {
		if d, ok := nodeDedup[id]; ok {
			return d
		}
		d := &dedupSets{
			files:       make(map[string]bool),
			connections: make(map[string]bool),
			dns:         make(map[string]bool),
			execs:       make(map[string]bool),
		}
		nodeDedup[id] = d
		return d
	}

	// Build a global DNS resolver table: IP -> hostname.
	// Populated from net_packet_dns_response events so that connections to
	// DNS-resolved addresses can be stored by hostname instead of raw IP.
	// This makes deduplication resilient to DNS load balancing / CDN rotation.
	dnsResolver := make(map[string]string)
	for _, evt := range events {
		if evt.DNSResponse != nil {
			for _, addr := range evt.DNSResponse.Addrs {
				dnsResolver[addr] = evt.DNSResponse.Query
			}
		}
	}

	for _, evt := range events {
		node := getNode(evt.PID)
		dedup := getDedup(node.Identity)

		switch {
		case evt.Exec != nil:
			key := evt.Exec.Pathname
			if !dedup.execs[key] {
				dedup.execs[key] = true
				node.Behaviors.Execs = append(node.Behaviors.Execs, ExecBehavior{
					Pathname: evt.Exec.Pathname,
					Argv:     evt.Exec.Argv,
				})
			}
		case evt.FileOp != nil:
			if !shouldIgnoreFile(evt.FileOp.Path, ignorePrefixes) && !dedup.files[evt.FileOp.Path] {
				dedup.files[evt.FileOp.Path] = true
				node.Behaviors.FilesOpened = append(node.Behaviors.FilesOpened, evt.FileOp.Path)
			}
		case evt.Connect != nil:
			// Replace IP with hostname if a DNS response mapped this address.
			addr := evt.Connect.Addr.String()
			if hostname, ok := dnsResolver[addr]; ok {
				addr = hostname
			}
			key := fmt.Sprintf("%s:%d", addr, evt.Connect.Port)
			if !dedup.connections[key] {
				dedup.connections[key] = true
				node.Behaviors.Connections = append(node.Behaviors.Connections, ConnectionBehavior{
					Addr: addr,
					Port: evt.Connect.Port,
				})
			}
		case evt.DNS != nil:
			if !dedup.dns[evt.DNS.Query] {
				dedup.dns[evt.DNS.Query] = true
				node.Behaviors.DNSQueries = append(node.Behaviors.DNSQueries, evt.DNS.Query)
			}
		}
	}

	// Phase 5: Assemble the tree hierarchy.
	root := &ProcessNode{Name: "root", Identity: "root"}

	// Build parent -> children relationships using identities.
	for pid, info := range pids {
		node := getNode(pid)
		var parent *ProcessNode
		if info.ppid == 0 {
			parent = root
		} else {
			parentID := identities[info.ppid]
			if p, ok := nodes[parentID]; ok {
				parent = p
			} else {
				parent = root
			}
		}
		parent.Children = append(parent.Children, node)
	}

	// Sort children by identity for deterministic output.
	sortChildren(root)

	return &ProcessTree{Root: root}, nil
}

// sortChildren recursively sorts children of each node by identity.
func sortChildren(n *ProcessNode) {
	if len(n.Children) > 0 {
		sort.Slice(n.Children, func(i, j int) bool {
			return n.Children[i].Identity < n.Children[j].Identity
		})
		for _, c := range n.Children {
			sortChildren(c)
		}
	}
}

// PrintTree returns a human-readable representation of the process tree.
func PrintTree(tree *ProcessTree) string {
	var sb strings.Builder
	printNode(&sb, tree.Root, 0)
	return sb.String()
}

func printNode(sb *strings.Builder, n *ProcessNode, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(sb, "%s%s\n", indent, n.Name)

	b := &n.Behaviors
	if len(b.Execs) > 0 {
		for _, e := range b.Execs {
			fmt.Fprintf(sb, "%s  [exec] %s\n", indent, strings.Join(e.Argv, " "))
		}
	}
	if len(b.FilesOpened) > 0 {
		fmt.Fprintf(sb, "%s  [files] %d unique paths\n", indent, len(b.FilesOpened))
	}
	if len(b.Connections) > 0 {
		for _, c := range b.Connections {
			fmt.Fprintf(sb, "%s  [connect] %s:%d\n", indent, c.Addr, c.Port)
		}
	}
	if len(b.DNSQueries) > 0 {
		for _, q := range b.DNSQueries {
			fmt.Fprintf(sb, "%s  [dns] %s\n", indent, q)
		}
	}

	for _, c := range n.Children {
		printNode(sb, c, depth+1)
	}
}

// shouldIgnoreFile returns true if the path matches any of the ignore prefixes.
func shouldIgnoreFile(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
