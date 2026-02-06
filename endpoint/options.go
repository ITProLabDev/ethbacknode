package endpoint

import (
	"net"
	"strconv"

	"github.com/ITProLabDev/ethbacknode/security"
	"golang.org/x/crypto/ssh"
)

// WithHttpListener configures the server to listen on HTTP at the specified address.
func WithHttpListener(listenAddress string) ServerOption {
	return func(s *endpointServer) error {
		ln, err := net.Listen("tcp4", listenAddress)
		if err != nil {
			return err
		}
		s.ln = ln
		return nil
	}
}

// WithSocketListener configures the server to listen on a Unix socket.
func WithSocketListener(socketFile string) ServerOption {
	return func(s *endpointServer) error {
		ln, err := net.Listen("tcp4", socketFile)
		if err != nil {
			return err
		}
		s.ln = ln
		return nil
	}
}

// WithSshListener configures the server to accept SSH connections with key authentication.
func WithSshListener(listenPort int, knownKeys []string) ServerOption {
	return func(s *endpointServer) error {
		netConn, err := net.Listen("tcp4", ":"+strconv.Itoa(listenPort))
		if err != nil {
			return err
		}
		s.ln = sshListener{
			conn:      netConn,
			knownKeys: knownKeys,
			sshConfig: &ssh.ServerConfig{
				PublicKeyCallback: keyAuth,
				PasswordCallback:  apiKey,
			},
		}
		return nil
	}
}

// WithDebugMode enables debug logging for RPC requests.
func WithDebugMode(debugMode bool) BackRpcOption {
	return func(r *BackRpc) {
		r.debugMode = debugMode
	}
}

// WithFallbackResponse sets the response for unmatched GET requests.
func WithFallbackResponse(response HttpResponse) BackRpcOption {
	return func(r *BackRpc) {
		r.fallbackResponse = response
	}
}

// WithRpcProcessor registers a custom RPC method processor.
func WithRpcProcessor(method RpcMethod, processor RpcProcessor) BackRpcOption {
	return func(r *BackRpc) {
		r.AddRpcProcessor(method, processor)
	}
}

// WithSecurityManager sets the security manager for request authentication.
func WithSecurityManager(securityManager *security.Manager) BackRpcOption {
	return func(r *BackRpc) {
		r.security = securityManager
	}
}
