package endpoint

import (
	"net"

	"github.com/ITProLabDev/ethbacknode/tools/log"
	"golang.org/x/crypto/ssh"
)

//todo access via ssh tunnels

type sshListener struct {
	conn      net.Listener
	knownKeys []string
	sshConfig *ssh.ServerConfig
}

func (s sshListener) Accept() (net.Conn, error) {
	////step 1: accept RAW net Connection...
	//conn, err := s.conn.Accept()
	//if err != nil {
	//	return nil, err
	//}
	////step 2: ssh handshake...
	//// From a standard TCP connection to an encrypted SSH connection
	//sshConn, _, _, err := ssh.NewServerConn(conn, s.sshConfig)
	//warpedConn := &sshConn{
	//	Conn: sshConn,
	//}
	//return sshConn.Conn, err
	panic("implement me")
}

func (s sshListener) Close() error {
	return s.conn.Close()
}

func (s sshListener) Addr() net.Addr {
	return s.conn.Addr()
}

func keyAuth(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	log.Debug(conn.RemoteAddr(), "authenticate with", key.Type())
	log.Critical("TODO: check key and extract ServiceId")
	return nil, nil
}

func apiKey(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	log.Debug(conn.RemoteAddr(), "authenticate with password")
	log.Critical("TODO: check password and extract ServiceId")
	return nil, nil
}
