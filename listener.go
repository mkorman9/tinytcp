package tinytcp

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
)

// Listener represents a low-level interface used by server to manage its interface.
type Listener interface {
	net.Listener

	// Listen starts listening. It's expected not to block a calling thread.
	Listen() error
}

type netListener struct {
	address  string
	config   *ServerConfig
	listener net.Listener
	m        sync.RWMutex
}

func (l *netListener) Listen() error {
	l.m.Lock()
	defer l.m.Unlock()

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
	var ln net.Listener

	l.m.RLock()
	{
		if l.listener == nil {
			l.m.RUnlock()
			return nil, io.EOF
		}

		ln = l.listener
	}
	l.m.RUnlock()

	return ln.Accept()
}

func (l *netListener) Addr() net.Addr {
	l.m.RLock()
	defer l.m.RUnlock()

	if l.listener == nil {
		return &net.TCPAddr{}
	}

	return l.listener.Addr()
}

func (l *netListener) Close() error {
	l.m.Lock()
	defer l.m.Unlock()

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
