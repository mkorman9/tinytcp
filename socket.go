package tinytcp

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Socket represents a connected TCP socket.
// An instance of Socket is only valid inside its designated handler and cannot be stored outside (see SocketReference).
type Socket struct {
	remoteAddress      string
	connectedAt        time.Time
	connection         net.Conn
	reader             io.Reader
	writer             io.Writer
	meteredReader      *meteredReader
	meteredWriter      *meteredWriter
	recyclable         uint32
	closeOnce          *sync.Once
	closeHandlers      []SocketCloseHandler
	closeHandlersMutex sync.RWMutex
	recycleHandler     func()

	prev *Socket
	next *Socket
}

// SocketHandler represents a signature of function used by Server to handle new connections.
type SocketHandler func(*Socket)

// SocketCloseHandler represents a signature of function used by Socket to register custom close handlers.
type SocketCloseHandler func(CloseReason)

// RemoteAddress returns a remote address of the socket.
func (s *Socket) RemoteAddress() string {
	return s.remoteAddress
}

// ConnectedAt returns an exact time the socket has connected.
func (s *Socket) ConnectedAt() time.Time {
	return s.connectedAt
}

// Close closes underlying TCP connection and executes all the registered close handlers.
// This method always returns nil, but its signature is meant to stick to the io.Closer interface.
func (s *Socket) Close(reason ...CloseReason) (err error) {
	s.closeOnce.Do(func() {
		if e := s.connection.Close(); e != nil {
			err = e
		}

		r := CloseReasonServer
		if reason != nil {
			r = reason[0]
		}

		s.closeHandlersMutex.RLock()
		for i := len(s.closeHandlers) - 1; i >= 0; i-- {
			handler := s.closeHandlers[i]
			handler(r)
		}
		s.closeHandlersMutex.RUnlock()
	})

	return
}

// Read conforms to the io.Reader interface.
func (s *Socket) Read(b []byte) (int, error) {
	n, err := s.reader.Read(b)
	if err != nil {
		if isBrokenPipe(err) {
			_ = s.Close(CloseReasonClient)
			return n, io.EOF
		}

		return n, err
	}

	return n, nil
}

// Write conforms to the io.Writer interface.
func (s *Socket) Write(b []byte) (int, error) {
	n, err := s.writer.Write(b)
	if err != nil {
		if isBrokenPipe(err) {
			_ = s.Close(CloseReasonClient)
			return n, io.EOF
		}

		return n, err
	}

	return n, nil
}

// SetReadDeadline sets read deadline for underlying socket.
func (s *Socket) SetReadDeadline(deadline time.Time) error {
	err := s.connection.SetReadDeadline(deadline)
	if err != nil {
		if isBrokenPipe(err) {
			_ = s.Close(CloseReasonClient)
			return io.EOF
		}

		return err
	}

	return nil
}

// SetWriteDeadline sets read deadline for underlying socket.
func (s *Socket) SetWriteDeadline(deadline time.Time) error {
	err := s.connection.SetWriteDeadline(deadline)
	if err != nil {
		if isBrokenPipe(err) {
			_ = s.Close(CloseReasonClient)
			return io.EOF
		}

		return err
	}

	return nil
}

// OnClose registers a handler that is called when underlying TCP connection is being closed.
func (s *Socket) OnClose(handler SocketCloseHandler) {
	s.closeHandlersMutex.Lock()
	defer s.closeHandlersMutex.Unlock()

	s.closeHandlers = append(s.closeHandlers, handler)
}

// Unwrap returns underlying net.Conn instance from Socket.
func (s *Socket) Unwrap() net.Conn {
	return s.connection
}

// UnwrapTLS tries to return underlying tls.Conn instance from Socket.
func (s *Socket) UnwrapTLS() (*tls.Conn, bool) {
	if conn, ok := s.connection.(*tls.Conn); ok {
		return conn, true
	}

	return nil, false
}

// WrapReader allows to wrap reader object into user defined wrapper.
func (s *Socket) WrapReader(wrapper func(io.Reader) io.Reader) {
	s.reader = wrapper(s.reader)
}

// WrapWriter allows to wrap writer object into user defined wrapper.
func (s *Socket) WrapWriter(wrapper func(io.Writer) io.Writer) {
	s.writer = wrapper(s.writer)
}

// TotalRead returns a total number of bytes read through this socket.
func (s *Socket) TotalRead() uint64 {
	return s.meteredReader.Total()
}

// ReadLastSecond returns a total number of bytes read from this socket last second.
func (s *Socket) ReadLastSecond() uint64 {
	return s.meteredReader.PerSecond()
}

// TotalWritten returns a total number of bytes written through this socket.
func (s *Socket) TotalWritten() uint64 {
	return s.meteredWriter.Total()
}

// WrittenLastSecond returns a total number of bytes written to this socket last second.
func (s *Socket) WrittenLastSecond() uint64 {
	return s.meteredWriter.PerSecond()
}

func (s *Socket) init(conn net.Conn) {
	s.remoteAddress = parseRemoteAddress(conn)
	s.connectedAt = time.Now()
	s.connection = conn
	s.meteredReader.reader = conn
	s.meteredWriter.writer = conn
	s.reader = s.meteredReader
	s.writer = s.meteredWriter
	s.closeOnce = &sync.Once{}
	s.closeHandlersMutex = sync.RWMutex{}
}

func (s *Socket) reset() {
	s.remoteAddress = ""
	s.connection = nil
	s.meteredReader.reset()
	s.meteredWriter.reset()
	s.recyclable = 0
	s.closeOnce = nil
	s.closeHandlers = nil
	s.recycleHandler = nil

	s.prev = nil
	s.next = nil
}

func (s *Socket) recycle() {
	if s.recycleHandler != nil {
		s.recycleHandler()
	}

	atomic.StoreUint32(&s.recyclable, 1)
}

func (s *Socket) isRecyclable() bool {
	return atomic.LoadUint32(&s.recyclable) == 1
}

func (s *Socket) onRecycle(handler func()) {
	s.recycleHandler = handler
}

func (s *Socket) updateMetrics(interval time.Duration) (uint64, uint64) {
	reads := s.meteredReader.Update(interval)
	writes := s.meteredWriter.Update(interval)
	return reads, writes
}
