package tinytcp

import (
	"io"
	"net"
	"time"
)

// net.Addr

type AddrMock struct {
}

func (am *AddrMock) Network() string {
	return "tcp"
}

func (am *AddrMock) String() string {
	return "127.0.0.1:1234"
}

// net.Conn

type ConnMock struct {
}

func (cm *ConnMock) Read(_ []byte) (int, error) {
	return 0, nil
}

func (cm *ConnMock) Write(_ []byte) (int, error) {
	return 0, nil
}

func (cm *ConnMock) Close() error {
	return nil
}

func (cm *ConnMock) LocalAddr() net.Addr {
	return &AddrMock{}
}

func (cm *ConnMock) RemoteAddr() net.Addr {
	return &AddrMock{}
}

func (cm *ConnMock) SetDeadline(_ time.Time) error {
	return nil
}

func (cm *ConnMock) SetReadDeadline(_ time.Time) error {
	return nil
}

func (cm *ConnMock) SetWriteDeadline(_ time.Time) error {
	return nil
}

// Socket

func MockSocket(in io.Reader, out io.Writer) *Socket {
	return &Socket{
		remoteAddress: "127.0.0.1",
		connectedAt:   time.Now(),
		connection:    &ConnMock{},
		reader:        in,
		writer:        out,
	}
}
