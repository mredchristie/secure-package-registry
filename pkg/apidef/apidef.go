// Package apidef provides route metadata definitions used by both handler
// packages and the OpenAPI generator (cmd/apigen).
package apidef

import "reflect"

// RouteDef describes a single API route for OpenAPI generation.
type RouteDef struct {
	// Method is the HTTP method (GET, POST, DELETE, etc.).
	Method string
	// Path is the full OpenAPI-style path (e.g. "/api/v1/admin/packages").
	Path string
	// Summary is a short description for the OpenAPI operation.
	Summary string
	// Description is an optional longer description.
	Description string
	// Tag groups the operation in the OpenAPI spec.
	Tag string
	// OperationID is a unique identifier for the operation.
	OperationID string

	// RequestBody is the reflect.Type of the request body struct (nil if none).
	RequestBody reflect.Type
	// ResponseType is the reflect.Type of the success response struct (nil for raw/no-body).
	ResponseType reflect.Type
	// SuccessCode is the HTTP status code for the success response (e.g. 200, 201, 204).
	SuccessCode int

	// PathParams lists the names of path parameters (e.g. ["ecosystem", "identifier"]).
	PathParams []string
	// QueryParams lists the query parameters with optional descriptions.
	QueryParams []QueryParam

	// Auth indicates whether the endpoint requires authentication.
	Auth bool
	// StreamResponse indicates the success response is raw bytes (not JSON).
	StreamResponse bool
	// ContentType overrides the response content type (default "application/json").
	ContentType string
}

// QueryParam describes a query parameter.
type QueryParam struct {
	Name        string
	Description string
	Required    bool
}
