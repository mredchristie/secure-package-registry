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
