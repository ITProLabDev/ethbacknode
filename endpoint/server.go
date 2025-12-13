package endpoint

import (
	"net"

	"github.com/valyala/fasthttp"
)

type ServerOption func(s *endpointServer) error

func WithHandler(handler fasthttp.RequestHandler) ServerOption {
	return func(s *endpointServer) error {
		s.server.Handler = handler
		return nil
	}
}

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

type endpointServer struct {
	server *fasthttp.Server
	ln     net.Listener
}

func (s *endpointServer) ListenAndServe() error {
	return s.server.Serve(s.ln)
}

func (s *endpointServer) Close() error {
	return s.ln.Close()
}
