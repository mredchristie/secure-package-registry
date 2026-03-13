package behavior_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"git.duti.dev/secure-package-registry/pkg/behavior"
)

const (
	safeFile      = "../../context-references/sample-behaviors/safe.jsonl"
	safe2File     = "../../context-references/sample-behaviors/safe-2.jsonl"
	maliciousFile = "../../context-references/sample-behaviors/malicious.jsonl"
)

func loadTree(t *testing.T, path string) *behavior.ProcessTree {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()

	tree, err := behavior.BuildTree(f, nil)
	if err != nil {
		t.Fatalf("building tree from %s: %v", path, err)
	}
	return tree
}

func TestBuildTree_Safe(t *testing.T) {
	tree := loadTree(t, safeFile)

	t.Log("Safe tree:\n" + behavior.PrintTree(tree))

	if tree.Root == nil {
		t.Fatal("expected non-nil root")
	}
	if len(tree.Root.Children) == 0 {
		t.Fatal("expected children under root")
	}

	// Verify we see expected process names in the tree.
	names := collectProcessNames(tree.Root)
	for _, expected := range []string{"node", "npm", "sh"} {
		if !names[expected] {
			t.Errorf("expected process %q in tree, got names: %v", expected, names)
		}
	}
}

func TestBuildTree_Malicious(t *testing.T) {
	tree := loadTree(t, maliciousFile)

	t.Log("Malicious tree:\n" + behavior.PrintTree(tree))

	// Verify malicious-specific processes appear.
	names := collectProcessNames(tree.Root)
	for _, expected := range []string{"curl", "trufflehog", "tar"} {
		if !names[expected] {
			t.Errorf("expected malicious process %q in tree, got names: %v", expected, names)
		}
	}
}

func TestDedupe_MaliciousVsSafe_KeepsSignals(t *testing.T) {
	baseline := loadTree(t, safeFile)
	target := loadTree(t, maliciousFile)

	diff := behavior.Dedupe(target, baseline)

	t.Log("Dedupe (malicious - safe):\n" + behavior.PrintTree(diff))

	// The diff must retain the malicious signals.
	names := collectProcessNames(diff.Root)

	// curl downloading trufflehog binary must survive.
	if !names["curl"] {
		t.Error("diff should contain 'curl' (downloads trufflehog binary)")
	}

	// trufflehog scanning filesystem must survive.
	if !names["trufflehog"] {
		t.Error("diff should contain 'trufflehog' (scans for secrets)")
	}

	// Check that malicious DNS queries survive.
	dnsQueries := collectAllDNS(diff.Root)
	for _, expected := range []string{"github.com", "oss.trufflehog.org"} {
		if !dnsQueries[expected] {
			t.Errorf("diff should contain DNS query for %q, got: %v", expected, dnsQueries)
		}
	}

	// Check that malicious connections survive (169.254.169.254 = IMDS).
	connections := collectAllConnections(diff.Root)
	if !connections["169.254.169.254:80"] {
		t.Errorf("diff should contain connection to IMDS 169.254.169.254:80, got: %v", connections)
	}

	// Verify the malicious exec chain survives.
	execs := collectAllExecs(diff.Root)
	if !execs["/usr/bin/curl"] {
		t.Errorf("diff should contain exec of /usr/bin/curl, got execs: %v", execs)
	}
}

func TestDedupe_MaliciousVsSafe_RemovesBaseline(t *testing.T) {
	baseline := loadTree(t, safeFile)
	target := loadTree(t, maliciousFile)

	diff := behavior.Dedupe(target, baseline)

	// The baseline npm install connecting to git.duti.dev should be removed.
	dnsQueries := collectAllDNS(diff.Root)
	if dnsQueries["git.duti.dev"] {
		t.Error("diff should NOT contain DNS query for git.duti.dev (baseline noise)")
	}

	// Baseline connections to the Gitea registry (80.0.44.82:443) should be removed.
	connections := collectAllConnections(diff.Root)
	if connections["80.0.44.82:443"] {
		t.Error("diff should NOT contain connection to 80.0.44.82:443 (registry baseline)")
	}
}

func TestDedupe_Safe2VsSafe_MinimalRemainder(t *testing.T) {
	baseline := loadTree(t, safeFile)
	target := loadTree(t, safe2File)

	diff := behavior.Dedupe(target, baseline)

	t.Log("Dedupe (safe-2 - safe):\n" + behavior.PrintTree(diff))

	// safe-2 is another safe package. After diffing against safe baseline,
	// there should be very little remaining. Specifically:
	// - No curl, trufflehog, or other malicious processes.
	names := collectProcessNames(diff.Root)
	for _, bad := range []string{"curl", "trufflehog", "tar", "gzip"} {
		if names[bad] {
			t.Errorf("diff of safe-2 vs safe should NOT contain %q", bad)
		}
	}

	// Count total remaining behaviors to verify noise is low.
	totalBehaviors := countBehaviors(diff.Root)
	t.Logf("Total remaining behaviors after safe-2 vs safe diff: %d", totalBehaviors)

	// We expect minimal remaining behaviors — the two safe packages should be
	// very similar. Allow some tolerance for path differences, but flag if
	// there's too much noise.
	if totalBehaviors > 200 {
		t.Errorf("too many remaining behaviors (%d) in safe-2 vs safe diff — expected mostly deduplicated", totalBehaviors)
	}
}

func TestDedupe_TargetVsItself_Empty(t *testing.T) {
	tree := loadTree(t, safeFile)

	diff := behavior.Dedupe(tree, tree)

	totalBehaviors := countBehaviors(diff.Root)
	if totalBehaviors != 0 {
		t.Errorf("diffing a tree against itself should yield 0 behaviors, got %d", totalBehaviors)
		t.Log("Self-diff:\n" + behavior.PrintTree(diff))
	}
}

func TestProcessTreeJSON(t *testing.T) {
	tree := loadTree(t, safeFile)

	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		t.Fatalf("marshaling tree to JSON: %v", err)
	}

	// Verify it round-trips.
	var roundTrip behavior.ProcessTree
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshaling tree from JSON: %v", err)
	}

	if roundTrip.Root == nil {
		t.Fatal("round-tripped root is nil")
	}
	if roundTrip.Root.Name != "root" {
		t.Errorf("expected root name 'root', got %q", roundTrip.Root.Name)
	}
}

func TestBuildTree_DNSResponseNormalizesConnections(t *testing.T) {
	// Synthetic JSONL: a process makes a DNS query for example.com,
	// receives a DNS response mapping example.com -> 93.184.216.34,
	// then connects to 93.184.216.34:443 and also to 169.254.169.254:80 (no DNS).
	events := []string{
		`{"timestamp":1,"processId":100,"parentProcessId":1,"processName":"node","eventName":"execve","returnValue":0,"args":[{"name":"pathname","type":"const char*","value":"/usr/bin/node"},{"name":"argv","type":"const char*const*","value":["node","index.js"]}]}`,
		`{"timestamp":2,"processId":100,"parentProcessId":1,"processName":"node","eventName":"net_packet_dns_request","returnValue":0,"args":[{"name":"dns_questions","type":"[]trace.DnsQueryData","value":[{"query":"example.com","type":"A","class":"IN"}]}]}`,
		`{"timestamp":3,"processId":100,"parentProcessId":1,"processName":"node","eventName":"net_packet_dns_response","returnValue":0,"args":[{"name":"dns_response","type":"[]trace.DnsResponseData","value":[{"query_data":{"query":"example.com","type":"A","class":"IN"},"dns_answer":[{"answer_type":"A","ttl":300,"answer":"93.184.216.34"}]}]}]}`,
		`{"timestamp":4,"processId":100,"parentProcessId":1,"processName":"node","eventName":"connect","returnValue":0,"args":[{"name":"addr","type":"struct sockaddr*","value":{"sa_family":"AF_INET","sin_addr":"93.184.216.34","sin_port":"443"}}]}`,
		`{"timestamp":5,"processId":100,"parentProcessId":1,"processName":"node","eventName":"connect","returnValue":0,"args":[{"name":"addr","type":"struct sockaddr*","value":{"sa_family":"AF_INET","sin_addr":"169.254.169.254","sin_port":"80"}}]}`,
	}

	r := strings.NewReader(strings.Join(events, "\n"))
	tree, err := behavior.BuildTree(r, nil)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}

	connections := collectAllConnections(tree.Root)

	// The DNS-resolved connection should be stored with hostname.
	if !connections["example.com:443"] {
		t.Errorf("expected connection 'example.com:443' (DNS-normalized), got: %v", connections)
	}
	// Raw IP should NOT appear for the DNS-resolved address.
	if connections["93.184.216.34:443"] {
		t.Error("connection should use hostname 'example.com' not raw IP '93.184.216.34'")
	}
	// Direct IP connection (no DNS) should remain as raw IP.
	if !connections["169.254.169.254:80"] {
		t.Errorf("expected direct IP connection '169.254.169.254:80' to remain, got: %v", connections)
	}
}

func TestBuildTree_DNSResponseMultipleAddrs(t *testing.T) {
	// DNS response with multiple A records (CDN load balancing).
	// Both IPs should resolve to the same hostname.
	events := []string{
		`{"timestamp":1,"processId":100,"parentProcessId":1,"processName":"node","eventName":"execve","returnValue":0,"args":[{"name":"pathname","type":"const char*","value":"/usr/bin/node"},{"name":"argv","type":"const char*const*","value":["node","index.js"]}]}`,
		`{"timestamp":2,"processId":100,"parentProcessId":1,"processName":"node","eventName":"net_packet_dns_response","returnValue":0,"args":[{"name":"dns_response","type":"[]trace.DnsResponseData","value":[{"query_data":{"query":"registry.npmjs.org","type":"A","class":"IN"},"dns_answer":[{"answer_type":"A","ttl":60,"answer":"104.16.23.35"},{"answer_type":"A","ttl":60,"answer":"104.16.24.35"}]}]}]}`,
		`{"timestamp":3,"processId":100,"parentProcessId":1,"processName":"node","eventName":"connect","returnValue":0,"args":[{"name":"addr","type":"struct sockaddr*","value":{"sa_family":"AF_INET","sin_addr":"104.16.23.35","sin_port":"443"}}]}`,
		`{"timestamp":4,"processId":100,"parentProcessId":1,"processName":"node","eventName":"connect","returnValue":0,"args":[{"name":"addr","type":"struct sockaddr*","value":{"sa_family":"AF_INET","sin_addr":"104.16.24.35","sin_port":"443"}}]}`,
	}

	r := strings.NewReader(strings.Join(events, "\n"))
	tree, err := behavior.BuildTree(r, nil)
	if err != nil {
		t.Fatalf("BuildTree: %v", err)
	}

	connections := collectAllConnections(tree.Root)

	// Both connections should resolve to the hostname, and since they share
	// hostname:port they should be deduplicated to a single entry.
	if !connections["registry.npmjs.org:443"] {
		t.Errorf("expected connection 'registry.npmjs.org:443', got: %v", connections)
	}
	if connections["104.16.23.35:443"] || connections["104.16.24.35:443"] {
		t.Error("raw IPs should not appear when DNS response mapped them to a hostname")
	}

	// Count: should be exactly 1 connection (deduplicated by hostname:port).
	connCount := 0
	var walk func(*behavior.ProcessNode)
	walk = func(n *behavior.ProcessNode) {
		connCount += len(n.Behaviors.Connections)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(tree.Root)
	if connCount != 1 {
		t.Errorf("expected 1 deduplicated connection, got %d", connCount)
	}
}

// --- helpers ---

func collectProcessNames(node *behavior.ProcessNode) map[string]bool {
	names := make(map[string]bool)
	var walk func(*behavior.ProcessNode)
	walk = func(n *behavior.ProcessNode) {
		names[n.Name] = true
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(node)
	return names
}

func collectAllDNS(node *behavior.ProcessNode) map[string]bool {
	result := make(map[string]bool)
	var walk func(*behavior.ProcessNode)
	walk = func(n *behavior.ProcessNode) {
		for _, q := range n.Behaviors.DNSQueries {
			result[q] = true
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(node)
	return result
}

func collectAllConnections(node *behavior.ProcessNode) map[string]bool {
	result := make(map[string]bool)
	var walk func(*behavior.ProcessNode)
	walk = func(n *behavior.ProcessNode) {
		for _, c := range n.Behaviors.Connections {
			key := c.Addr + ":" + fmt.Sprint(c.Port)
			result[key] = true
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(node)
	return result
}

func collectAllExecs(node *behavior.ProcessNode) map[string]bool {
	result := make(map[string]bool)
	var walk func(*behavior.ProcessNode)
	walk = func(n *behavior.ProcessNode) {
		for _, e := range n.Behaviors.Execs {
			result[e.Pathname] = true
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(node)
	return result
}

func countBehaviors(node *behavior.ProcessNode) int {
	count := 0
	var walk func(*behavior.ProcessNode)
	walk = func(n *behavior.ProcessNode) {
		count += len(n.Behaviors.Execs)
		count += len(n.Behaviors.FilesOpened)
		count += len(n.Behaviors.Connections)
		count += len(n.Behaviors.DNSQueries)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(node)
	return count
}
