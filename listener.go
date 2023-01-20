package tinytcp

import (
	"crypto/tls"
	"io"
	"net"
)

// Listener represents a low-level interface used by server to manage its interface.
type Listener interface {
	io.Closer

	// Listen starts listener.
	Listen() error

	// Accept pulls a connection from a queue and returns it or blocks if there is none available.
	Accept() (net.Conn, error)

	// Port returns a port number used by the listener.
	Port() int
}

type netListener struct {
	address  string
	config   *ServerConfig
	listener net.Listener
}

func (l *netListener) Listen() error {
	if l.config.TLSCert != "" && l.config.TLSKey != "" {
		cert, err := tls.LoadX509KeyPair(l.config.TLSCert, l.config.TLSKey)
		if err != nil {
			return err
		}

		l.config.TLSConfig.Certificates = []tls.Certificate{cert}

		socket, err := tls.Listen(l.config.Network, l.address, l.config.TLSConfig)
		if err != nil {
			return err
		}

		l.listener = socket
	} else {
		socket, err := net.Listen(l.config.Network, l.address)
		if err != nil {
			return err
		}

		l.listener = socket
	}

	return nil
}

func (l *netListener) Accept() (net.Conn, error) {
	if l.listener == nil {
		return nil, io.EOF
	}

	return l.listener.Accept()
}

func (l *netListener) Port() int {
	if l.listener == nil {
		return -1
	}

	return resolveListenerPort(l.listener)
}

func (l *netListener) Close() error {
	if l.listener == nil {
		return nil
	}

	if err := l.listener.Close(); err != nil {
		return err
	}

	l.listener = nil
	return nil
}

func newListener(address string, config *ServerConfig) Listener {
	return &netListener{
		address: address,
		config:  config,
	}
}
