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
