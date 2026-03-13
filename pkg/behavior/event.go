// Package behavior provides aggregation and deduplication of Tracee behavioral
// analysis output. It parses raw JSONL events into a process ancestry graph
// with categorized behaviors per process node, and supports diffing against a
// baseline to extract only novel behaviors.
package behavior

import (
	"encoding/json"
	"fmt"
	"net/netip"
)

// RawEvent represents a single Tracee event as captured in behavior.jsonl.
// Only fields relevant to behavioral analysis are parsed; the rest is ignored.
type RawEvent struct {
	Timestamp       int64  `json:"timestamp"`
	ProcessID       int    `json:"processId"`
	ParentProcessID int    `json:"parentProcessId"`
	ProcessName     string `json:"processName"`
	EventName       string `json:"eventName"`
	ReturnValue     int    `json:"returnValue"`
	Args            []Arg  `json:"args"`
}

// Arg is a single argument from a Tracee event.
type Arg struct {
	Name  string          `json:"name"`
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

// StringValue returns the arg value as a string, or empty if not a string.
func (a *Arg) StringValue() string {
	var s string
	if err := json.Unmarshal(a.Value, &s); err != nil {
		return ""
	}
	return s
}

// StringSliceValue returns the arg value as a string slice, or nil.
func (a *Arg) StringSliceValue() []string {
	var ss []string
	if err := json.Unmarshal(a.Value, &ss); err != nil {
		return nil
	}
	return ss
}

// SockAddr represents a parsed socket address from a connect event.
type SockAddr struct {
	Family  string `json:"sa_family"`
	Addr    string `json:"sin_addr"`
	Addr6   string `json:"sin6_addr"`
	Port    string `json:"sin_port"`
	Port6   string `json:"sin6_port"`
	SunPath string `json:"sun_path"`
}

// SockAddrValue returns the arg value as a SockAddr, or nil.
func (a *Arg) SockAddrValue() *SockAddr {
	var sa SockAddr
	if err := json.Unmarshal(a.Value, &sa); err != nil {
		return nil
	}
	return &sa
}

// DNSQuestion is a single question from a DNS request event.
type DNSQuestion struct {
	Query string `json:"query"`
	Type  string `json:"type"`
	Class string `json:"class"`
}

// DNSQuestionsValue returns the arg value as a slice of DNS questions.
func (a *Arg) DNSQuestionsValue() []DNSQuestion {
	var qs []DNSQuestion
	if err := json.Unmarshal(a.Value, &qs); err != nil {
		return nil
	}
	return qs
}

// DNSAnswer is a single answer record from a DNS response.
type DNSAnswer struct {
	AnswerType string `json:"answer_type"`
	TTL        int    `json:"ttl"`
	Answer     string `json:"answer"`
}

// DNSResponseData is one entry from the dns_response argument, containing
// the original query and its answer records.
type DNSResponseData struct {
	QueryData  DNSQuestion `json:"query_data"`
	DNSAnswers []DNSAnswer `json:"dns_answer"`
}

// DNSResponseValue returns the arg value as a slice of DNS response data.
func (a *Arg) DNSResponseValue() []DNSResponseData {
	var rds []DNSResponseData
	if err := json.Unmarshal(a.Value, &rds); err != nil {
		return nil
	}
	return rds
}

// ParsedEvent holds the relevant extracted information from a RawEvent,
// categorized by event type.
type ParsedEvent struct {
	PID  int
	PPID int
	Name string // Final process name for this event

	// Exactly one of these is set depending on event type.
	Exec        *ExecEvent
	FileOp      *FileEvent
	Connect     *ConnectEvent
	DNS         *DNSEvent
	DNSResponse *DNSResponseEvent
}

// ExecEvent represents an execve call — a new process being spawned.
type ExecEvent struct {
	Pathname string
	Argv     []string
}

// FileEvent represents a file open/openat call.
type FileEvent struct {
	Path string
}

// ConnectEvent represents an outbound network connection.
type ConnectEvent struct {
	Family string
	Addr   netip.Addr // zero value if not parseable (e.g. AF_UNIX)
	Port   uint16
}

// DNSEvent represents a DNS query.
type DNSEvent struct {
	Query string
}

// DNSResponseEvent represents a DNS response with resolved addresses.
type DNSResponseEvent struct {
	Query string   // The queried domain name.
	Addrs []string // Resolved IP addresses (A/AAAA answers only).
}

// ParseEvent extracts the relevant information from a raw Tracee event.
// Returns nil for events we don't care about (e.g. runc init noise with no
// useful behavioral data).
func ParseEvent(raw *RawEvent) *ParsedEvent {
	p := &ParsedEvent{
		PID:  raw.ProcessID,
		PPID: raw.ParentProcessID,
		Name: raw.ProcessName,
	}

	switch raw.EventName {
	case "execve":
		p.Exec = parseExecve(raw)
	case "openat", "open":
		p.FileOp = parseOpen(raw)
	case "connect":
		p.Connect = parseConnect(raw)
		if p.Connect == nil {
			return nil // AF_UNIX or unparseable — skip
		}
	case "net_packet_dns_request":
		p.DNS = parseDNS(raw)
		if p.DNS == nil {
			return nil
		}
	case "net_packet_dns_response":
		p.DNSResponse = parseDNSResponse(raw)
		if p.DNSResponse == nil {
			return nil
		}
	default:
		return nil // Unknown event type
	}

	return p
}

func parseExecve(raw *RawEvent) *ExecEvent {
	e := &ExecEvent{}
	for i := range raw.Args {
		switch raw.Args[i].Name {
		case "pathname":
			e.Pathname = raw.Args[i].StringValue()
		case "argv":
			e.Argv = raw.Args[i].StringSliceValue()
		}
	}
	return e
}

func parseOpen(raw *RawEvent) *FileEvent {
	for i := range raw.Args {
		if raw.Args[i].Name == "pathname" {
			return &FileEvent{Path: raw.Args[i].StringValue()}
		}
	}
	return &FileEvent{}
}

func parseConnect(raw *RawEvent) *ConnectEvent {
	for i := range raw.Args {
		if raw.Args[i].Name == "addr" {
			sa := raw.Args[i].SockAddrValue()
			if sa == nil {
				return nil
			}
			switch sa.Family {
			case "AF_INET":
				addr, err := netip.ParseAddr(sa.Addr)
				if err != nil {
					return nil
				}
				port := parsePort(sa.Port)
				return &ConnectEvent{Family: sa.Family, Addr: addr, Port: port}
			case "AF_INET6":
				addr, err := netip.ParseAddr(sa.Addr6)
				if err != nil {
					return nil
				}
				port := parsePort(sa.Port6)
				return &ConnectEvent{Family: sa.Family, Addr: addr, Port: port}
			default:
				// AF_UNIX, AF_UNSPEC — not interesting for behavioral analysis
				return nil
			}
		}
	}
	return nil
}

func parseDNS(raw *RawEvent) *DNSEvent {
	for i := range raw.Args {
		if raw.Args[i].Name == "dns_questions" {
			qs := raw.Args[i].DNSQuestionsValue()
			if len(qs) > 0 {
				return &DNSEvent{Query: qs[0].Query}
			}
		}
	}
	return nil
}

func parseDNSResponse(raw *RawEvent) *DNSResponseEvent {
	for i := range raw.Args {
		if raw.Args[i].Name == "dns_response" {
			rds := raw.Args[i].DNSResponseValue()
			if len(rds) == 0 {
				return nil
			}
			// Use the query name from the first response entry.
			query := rds[0].QueryData.Query
			// Collect all A and AAAA answer addresses.
			var addrs []string
			for _, rd := range rds {
				for _, ans := range rd.DNSAnswers {
					if ans.AnswerType == "A" || ans.AnswerType == "AAAA" {
						addrs = append(addrs, ans.Answer)
					}
				}
			}
			if len(addrs) == 0 {
				return nil // No IP answers (e.g. CNAME-only) — nothing to map.
			}
			return &DNSResponseEvent{Query: query, Addrs: addrs}
		}
	}
	return nil
}

func parsePort(s string) uint16 {
	var port uint16
	_, _ = fmt.Sscanf(s, "%d", &port)
	return port
}
