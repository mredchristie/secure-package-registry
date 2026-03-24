// Package server provides HTTP server implementations for the core service.
package server

import (
	"net/http"

	"git.duti.dev/secure-package-registry/pkg/httpserver"
)

// Server wraps httpserver.Server for the core service.
type Server = httpserver.Server

// New creates a new Server instance with the "core-svc:<name>" component tag.
func New(name, addr string, handler http.Handler) *Server {
	return httpserver.New("core-svc:"+name, addr, handler)
}
