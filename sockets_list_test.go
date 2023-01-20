package tinytcp

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestSocketsListSimple(t *testing.T) {
	// given
	list := newSocketsList(-1)
	connections := []net.Conn{&ConnMock{}, &ConnMock{}, &ConnMock{}}
	sockets := make([]*Socket, len(connections))

	// when
	for i, conn := range connections {
		sockets[i] = list.New(conn)
	}

	list.Cleanup()

	// then
	assert.Equal(t, len(sockets), list.Len(), "sockets count should match")
}

func TestSocketsListCleanup(t *testing.T) {
	// given
	list := newSocketsList(-1)
	connections := []net.Conn{&ConnMock{}, &ConnMock{}, &ConnMock{}}
	sockets := make([]*Socket, len(connections))

	// when
	for i, conn := range connections {
		sockets[i] = list.New(conn)
	}

	_ = sockets[0].Close()

	list.Cleanup()

	// then
	assert.Equal(t, len(sockets)-1, list.Len(), "sockets count should match")
}

func TestSocketsListLimit(t *testing.T) {
	// given
	list := newSocketsList(0)
	connection := &ConnMock{}

	// when
	socket := list.New(connection)

	// then
	assert.Nil(t, socket, "socket should not be returned")
}
