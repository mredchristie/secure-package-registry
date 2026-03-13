package npm

import (
	"encoding/json"
	"fmt"
	"time"
)

// SequenceID represents a CouchDB sequence number (can be string or int)
type SequenceID struct {
	value string
}

// NewSequenceID creates a SequenceID from a string
func NewSequenceID(s string) SequenceID {
	return SequenceID{value: s}
}

// String returns the sequence ID as a string
func (s SequenceID) String() string {
	return s.value
}

// MarshalJSON implements json.Marshaler
func (s SequenceID) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.value)
}

// UnmarshalJSON implements json.Unmarshaler (handles both string and int)
func (s *SequenceID) UnmarshalJSON(data []byte) error {
	// Try string first
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		s.value = strVal
		return nil
	}

	// Fall back to int and convert
	var intVal int64
	if err := json.Unmarshal(data, &intVal); err != nil {
		return fmt.Errorf("sequence ID must be string or int: %w", err)
	}
	s.value = fmt.Sprintf("%d", intVal)
	return nil
}

// Change represents a single change from the CouchDB _changes feed
type Change struct {
	Sequence SequenceID  `json:"seq"`
	ID       string      `json:"id"`
	Changes  []ChangeRev `json:"changes"`
	Deleted  bool        `json:"deleted,omitempty"`
	Document *NPMPackage `json:"doc,omitempty"`
}

// ChangeRev represents a revision in the changes array
type ChangeRev struct {
	Revision string `json:"rev"`
}

// ChangesResponse represents the response from _changes endpoint
type ChangesResponse struct {
	Results []Change   `json:"results"`
	LastSeq SequenceID `json:"last_seq"`
}

// NPMPackage represents a package document from the NPM registry
type NPMPackage struct {
	ID          string                `json:"_id"`
	Revision    string                `json:"_rev"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	DistTags    map[string]string     `json:"dist-tags,omitempty"`
	Versions    map[string]NPMVersion `json:"versions,omitempty"`
	Time        map[string]time.Time  `json:"time,omitempty"`
	Maintainers []NPMMaintainer       `json:"maintainers,omitempty"`
	Author      *NPMAuthor            `json:"author,omitempty"`
	Repository  *NPMRepository        `json:"repository,omitempty"`
}

// NPMVersion represents a specific version of a package.
type NPMVersion struct {
	Version              string            `json:"version"`
	Dependencies         map[string]string `json:"dependencies,omitempty"`
	OptionalDependencies map[string]string `json:"optionalDependencies,omitempty"`
	PeerDependencies     map[string]string `json:"peerDependencies,omitempty"`
	Dist                 NPMDist           `json:"dist"`
}

// NPMDist contains distribution metadata for a version.
type NPMDist struct {
	Tarball   string `json:"tarball"`
	Shasum    string `json:"shasum,omitempty"`
	Integrity string `json:"integrity,omitempty"`
}

// NPMMaintainer represents a package maintainer
type NPMMaintainer struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// NPMAuthor represents the package author
type NPMAuthor struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// NPMRepository represents the source code repository
type NPMRepository struct {
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
}

// RegistryInfo represents the response from GET /
type RegistryInfo struct {
	DBName                  string     `json:"db_name"`
	DocCount                int64      `json:"doc_count"`
	DocDeletedCount         int64      `json:"doc_del_count"`
	UpdateSequence          SequenceID `json:"update_seq"`
	PurgeSequence           int64      `json:"purge_seq"`
	CompactRunning          bool       `json:"compact_running"`
	DiskSize                int64      `json:"disk_size"`
	DataSize                int64      `json:"data_size"`
	InstanceStartTime       string     `json:"instance_start_time"`
	DiskFormatVersion       int        `json:"disk_format_version"`
	CommittedUpdateSequence int64      `json:"committed_update_seq"`
}

// IsDesignDoc returns true if this change is a CouchDB design document (not a real package)
func (c *Change) IsDesignDoc() bool {
	return len(c.ID) > 0 && c.ID[0] == '_'
}

// GetLatestVersion returns the latest version from dist-tags, or "" if not available
func (p *NPMPackage) GetLatestVersion() string {
	if p.DistTags == nil {
		return ""
	}
	return p.DistTags["latest"]
}
