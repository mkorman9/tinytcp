package singleclient

import (
	"net"
)

type mockListener struct {
	acceptQueue chan net.Conn
}

func (l *mockListener) Listen() error {
	return nil
}

func (l *mockListener) Accept() (net.Conn, error) {
	return <-l.acceptQueue, nil
}

func (l *mockListener) Port() int {
	return 0
}

func (l *mockListener) Close() error {
	return nil
}

func (l *mockListener) Connect() net.Conn {
	c1, c2 := net.Pipe()
	l.acceptQueue <- c1
	return c2
}

func newMockListener() *mockListener {
	return &mockListener{
		acceptQueue: make(chan net.Conn),
	}
}
