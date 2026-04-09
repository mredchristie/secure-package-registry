package messages

type PackageRequest struct {
	Ecosystem  string
	Identifier string
}

type PackageUpdated struct {
	Ecosystem  string
	Identifier string
	Version    string
}

type CollectionRequested struct {
	TaskID     int32
	Ecosystem  string
	Identifier string
	Version    string
}

type CollectionCompleted struct {
	TaskID         int32
	Ecosystem      string
	Identifier     string
	Version        string
	Success        bool
	ArtifactBucket string // empty on failure
	ArtifactKey    string // empty on failure
	FailureReason  string // empty on success
}

// ProjectProcessingRequested is published when a user uploads a project file.
// The consumer picks it up and runs async dependency resolution + storage.
type ProjectProcessingRequested struct {
	ProjectID  int32
	Generation int32 // matches user_projects.generation; used to discard stale messages
}
