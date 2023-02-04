package tinytcp

import (
	"net"
	"sync"
)

type socketsList struct {
	head    *Socket
	tail    *Socket
	size    int
	maxSize int
	m       sync.RWMutex
	pool    sync.Pool
}

func newSocketsList(maxSize int) *socketsList {
	return &socketsList{
		maxSize: maxSize,
		pool: sync.Pool{
			New: func() any {
				return &Socket{
					meteredReader: &meteredReader{},
					meteredWriter: &meteredWriter{},
				}
			},
		},
	}
}

func (s *socketsList) New(connection net.Conn) *Socket {
	socket := s.newSocket(connection)

	if registered := s.registerSocket(socket); !registered {
		// instantly terminate the connection if it can't be added to the pool
		_ = connection.Close()
		s.recycleSocket(socket)
		return nil
	}

	return socket
}

func (s *socketsList) Len() int {
	s.m.RLock()
	defer s.m.RUnlock()

	return s.size
}

func (s *socketsList) Cleanup() {
	s.m.Lock()
	defer s.m.Unlock()

	for socket := s.head; socket != nil; {
		next := socket.next

		if socket.isRecyclable() {
			switch socket {
			case s.head:
				s.head = socket.next
			case s.tail:
				s.tail = socket.prev
				s.tail.next = nil
			default:
				socket.prev.next = socket.next
				socket.next.prev = socket.prev
			}

			s.recycleSocket(socket)
			s.size--
		}

		socket = next
	}
}

func (s *socketsList) Iterate(fn func(s *Socket)) {
	s.m.RLock()
	defer s.m.RUnlock()

	for socket := s.head; socket != nil; socket = socket.next {
		fn(socket)
	}
}

func (s *socketsList) Reset() {
	s.m.Lock()
	defer s.m.Unlock()

	for socket := s.head; socket != nil; socket = socket.next {
		_ = socket.Close()
		socket.recycle()
		s.recycleSocket(socket)
	}

	s.head = nil
	s.tail = nil
	s.size = 0
}

func (s *socketsList) newSocket(connection net.Conn) *Socket {
	socket := s.pool.Get().(*Socket)
	socket.init(connection)
	return socket
}

func (s *socketsList) registerSocket(socket *Socket) bool {
	s.m.Lock()
	defer s.m.Unlock()

	if s.maxSize >= 0 && s.size >= s.maxSize {
		return false
	}

	if s.head == nil {
		s.head = socket
		s.tail = socket
	} else {
		s.tail.next = socket
		socket.prev = s.tail
		s.tail = socket
	}

	s.size++

	return true
}

func (s *socketsList) recycleSocket(socket *Socket) {
	socket.reset()
	s.pool.Put(socket)
}
