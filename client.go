package tinytcp

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
)

// Client represents a TCP/TLS client.
type Client struct {
	connection net.Conn
	closeSync  sync.Once

	onCloseHandler func()
}

// Dial connects to the TCP socket and creates new Client.
func Dial(address string) (*Client, error) {
	connection, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return &Client{
		connection: connection,
	}, nil
}

// DialTLS connects to the TCP socket and performs TLS handshake, and then creates new Client.
// Connection is TLS secured.
func DialTLS(address string, tlsConfig *tls.Config) (*Client, error) {
	connection, err := tls.Dial("tcp", address, tlsConfig)
	if err != nil {
		return nil, err
	}

	return &Client{
		connection: connection,
	}, nil
}

// Close closes the socket.
func (c *Client) Close() error {
	var err error

	c.closeSync.Do(func() {
		e := c.connection.Close()
		if e != nil {
			err = e
		}

		if c.onCloseHandler != nil {
			c.onCloseHandler()
		}
	})

	return err
}

// Read conforms to the io.Reader interface.
func (c *Client) Read(b []byte) (int, error) {
	n, err := c.connection.Read(b)
	if err != nil {
		if isBrokenPipe(err) {
			_ = c.Close()
			return n, io.EOF
		}

		return n, err
	}

	return n, nil
}

// Write conforms to the io.Writer interface.
func (c *Client) Write(b []byte) (int, error) {
	n, err := c.connection.Write(b)
	if err != nil {
		if isBrokenPipe(err) {
			_ = c.Close()
			return n, io.EOF
		}

		return n, err
	}

	return n, nil
}

// Unwrap returns underlying TCP connection.
func (c *Client) Unwrap() net.Conn {
	return c.connection
}

// UnwrapTLS tries to return underlying tls.Conn instance.
func (c *Client) UnwrapTLS() (*tls.Conn, bool) {
	if conn, ok := c.connection.(*tls.Conn); ok {
		return conn, true
	}

	return nil, false
}

// OnClose sets a handler called on closing connection (either by client or server).
func (c *Client) OnClose(handler func()) {
	c.onCloseHandler = handler
}
