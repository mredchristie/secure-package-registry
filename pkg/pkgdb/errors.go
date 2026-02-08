package pkgdb

import (
	"fmt"

	coredb "secure-package-registry/internal/gen/coredb"
)

// Ecosystem is re-exported from the internal package for use in the public API
type Ecosystem = coredb.Ecosystem

// Public Ecosystem constants
const (
	EcosystemNpm   = coredb.EcosystemNpm
	EcosystemGo    = coredb.EcosystemGo
	EcosystemCargo = coredb.EcosystemCargo
	EcosystemPypi  = coredb.EcosystemPypi
)

// TagErrorKind represents the type of tag access error
type TagErrorKind int

const (
	// KindNotFound indicates the tag key does not exist
	KindNotFound TagErrorKind = iota
	// KindParseFailed indicates the tag value could not be parsed
	KindParseFailed
	// KindNoValue indicates the tag exists but has no value
	KindNoValue
)

// TagError represents an error accessing package version tags
type TagError struct {
	Kind TagErrorKind
	Tag  string
	Err  error
}

func (e *TagError) Error() string {
	switch e.Kind {
	case KindNotFound:
		return fmt.Sprintf("tag %q not found", e.Tag)
	case KindParseFailed:
		if e.Err != nil {
			return fmt.Sprintf("tag %q parse failed: %v", e.Tag, e.Err)
		}
		return fmt.Sprintf("tag %q parse failed", e.Tag)
	case KindNoValue:
		return fmt.Sprintf("tag %q exists but has no value", e.Tag)
	default:
		return fmt.Sprintf("tag %q error", e.Tag)
	}
}

// Unwrap returns the underlying error for KindParseFailed
func (e *TagError) Unwrap() error {
	if e.Kind == KindParseFailed {
		return e.Err
	}
	return nil
}
