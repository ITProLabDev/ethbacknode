// Package endpoint provides the HTTP/RPC API server for EthBackNode.
// It exposes JSON-RPC 2.0 endpoints for address management, balance queries,
// transaction operations, and subscriber management.
package endpoint

import (
	"net"

	"github.com/valyala/fasthttp"
)

// ServerOption is a function that configures an endpoint server.
type ServerOption func(s *endpointServer) error

// WithHandler sets the HTTP request handler for the server.
func WithHandler(handler fasthttp.RequestHandler) ServerOption {
	return func(s *endpointServer) error {
		s.server.Handler = handler
		return nil
	}
}

// NewServer creates a new HTTP server with the specified options.
func NewServer(options ...ServerOption) (server *endpointServer, err error) {
	server = &endpointServer{
		server: &fasthttp.Server{
			Name: "RPC Server v1.0",
		},
	}
	for _, opt := range options {
		err = opt(server)
		if err != nil {
			return nil, err
		}
	}
	return server, nil
}

// endpointServer wraps a fasthttp server with listener management.
type endpointServer struct {
	server *fasthttp.Server
	ln     net.Listener
}

// ListenAndServe starts the server and blocks until stopped.
func (s *endpointServer) ListenAndServe() error {
	return s.server.Serve(s.ln)
}

// Close stops the server by closing the listener.
func (s *endpointServer) Close() error {
	return s.ln.Close()
}
