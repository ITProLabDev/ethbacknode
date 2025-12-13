package endpoint

import (
	"net"
	"strconv"

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
