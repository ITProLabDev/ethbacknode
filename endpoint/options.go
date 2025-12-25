package endpoint

import (
	"net"
	"strconv"

	"github.com/ITProLabDev/ethbacknode/security"
	"golang.org/x/crypto/ssh"
)

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

func WithDebugMode(debugMode bool) BackRpcOption {
	return func(r *BackRpc) {
		r.debugMode = debugMode
	}
}

func WithFallbackResponse(response HttpResponse) BackRpcOption {
	return func(r *BackRpc) {
		r.fallbackResponse = response
	}
}

func WithRpcProcessor(method RpcMethod, processor RpcProcessor) BackRpcOption {
	return func(r *BackRpc) {
		r.AddRpcProcessor(method, processor)
	}
}

func WithSecurityManager(securityManager *security.Manager) BackRpcOption {
	return func(r *BackRpc) {
		r.security = securityManager
	}
}
