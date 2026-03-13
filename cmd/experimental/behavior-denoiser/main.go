// Command behavior-denoiser is a CLI tool for inspecting and deduplicating
// Tracee behavioral analysis JSONL files. It supports four subcommands:
//
//	behavior-denoiser tree <file>                 — print the process tree
//	behavior-denoiser dedupe <target> <baseline>  — print the deduplicated behavioral delta
//	behavior-denoiser json <file>                 — output the tree as JSON
//	behavior-denoiser dedupe-json <target> <baseline> — output the deduplicated tree as JSON
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"git.duti.dev/secure-package-registry/pkg/behavior"
)

const usage = `Usage: behavior-denoiser <command> [args]

Commands:
  tree       <file>                Print the process ancestry tree
  dedupe     <target> <baseline>   Print behaviors in target not in baseline
  json       <file>                Output the tree as JSON
  dedupe-json <target> <baseline>  Output the deduplicated tree as JSON

Examples:
  go run ./cmd/experimental/behavior-denoiser tree context-references/sample-behaviors/safe.jsonl
  go run ./cmd/experimental/behavior-denoiser dedupe context-references/sample-behaviors/malicious.jsonl context-references/sample-behaviors/safe.jsonl
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "tree":
		if len(args) != 1 {
			fatal("tree requires exactly 1 argument: <file>")
		}
		tree := mustLoadTree(args[0])
		fmt.Print(behavior.PrintTree(tree))
		printStats(tree)

	case "json":
		if len(args) != 1 {
			fatal("json requires exactly 1 argument: <file>")
		}
		tree := mustLoadTree(args[0])
		mustWriteJSON(tree)

	case "dedupe":
		if len(args) != 2 {
			fatal("dedupe requires exactly 2 arguments: <target> <baseline>")
		}
		target := mustLoadTree(args[0])
		baseline := mustLoadTree(args[1])
		deduped := behavior.Dedupe(target, baseline)
		fmt.Print(behavior.PrintTree(deduped))
		printDedupeStats(target, baseline, deduped)

	case "dedupe-json":
		if len(args) != 2 {
			fatal("dedupe-json requires exactly 2 arguments: <target> <baseline>")
		}
		target := mustLoadTree(args[0])
		baseline := mustLoadTree(args[1])
		deduped := behavior.Dedupe(target, baseline)
		mustWriteJSON(deduped)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

func mustLoadTree(path string) *behavior.ProcessTree {
	f, err := os.Open(path)
	if err != nil {
		fatal("opening %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()

	tree, err := behavior.BuildTree(f, nil)
	if err != nil {
		fatal("parsing %s: %v", path, err)
	}
	return tree
}

func mustWriteJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal("writing JSON: %v", err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// --- stats ---

type stats struct {
	processes   int
	execs       int
	files       int
	connections int
	dns         int
}

func gatherStats(node *behavior.ProcessNode) stats {
	var s stats
	walkStats(node, &s)
	return s
}

func walkStats(node *behavior.ProcessNode, s *stats) {
	if node.Name != "root" {
		s.processes++
	}
	s.execs += len(node.Behaviors.Execs)
	s.files += len(node.Behaviors.FilesOpened)
	s.connections += len(node.Behaviors.Connections)
	s.dns += len(node.Behaviors.DNSQueries)
	for _, c := range node.Children {
		walkStats(c, s)
	}
}

func (s stats) total() int {
	return s.execs + s.files + s.connections + s.dns
}

func (s stats) String() string {
	return fmt.Sprintf("%d processes, %d execs, %d files, %d connections, %d dns (%d total)",
		s.processes, s.execs, s.files, s.connections, s.dns, s.total())
}

func printStats(tree *behavior.ProcessTree) {
	s := gatherStats(tree.Root)
	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("─", 60))
	fmt.Fprintf(os.Stderr, "Stats: %s\n", s)
}

func printDedupeStats(target, baseline, deduped *behavior.ProcessTree) {
	ts := gatherStats(target.Root)
	bs := gatherStats(baseline.Root)
	ds := gatherStats(deduped.Root)

	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("─", 60))
	fmt.Fprintf(os.Stderr, "Target:     %s\n", ts)
	fmt.Fprintf(os.Stderr, "Baseline:   %s\n", bs)
	fmt.Fprintf(os.Stderr, "Remaining:  %s\n", ds)
	if ts.total() > 0 {
		reduction := float64(ts.total()-ds.total()) / float64(ts.total()) * 100
		fmt.Fprintf(os.Stderr, "Reduction: %.1f%%\n", reduction)
	}
}
