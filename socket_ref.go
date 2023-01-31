package tinytcp

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"
)

// SocketRef allows to hold a reference to a socket outside its designated handler.
// tinytcp performs no memory allocations on critical paths, which means it needs to pool some objects.
// These pooled objects are typically structures that exist only in the context of a specific connection (eg. Sockets).
// Objects are returned to the pool as they are no longer needed and reused by other connection in the future.
// The rule is that a socket instance is only valid inside its designated handler and storing it outside this handler
// might result in some very nasty bugs. SocketRef provides a way to safely store a reference to a socket,
// and provide an interface to all of its functionalities.
type SocketRef struct {
	s *Socket
	m sync.RWMutex
}

// NewSocketRef creates an instance of SocketReference.
func NewSocketRef(s *Socket) *SocketRef {
	ref := &SocketRef{
		s: s,
	}

	s.onRecycle(ref.onRecycle)
	return ref
}

// Read reads data from socket only if it hasn't been recycled yet.
func (r *SocketRef) Read(b []byte) (int, error) {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return 0, io.EOF
	}

	return r.s.Read(b)
}

// Write writes data to a socket only if it hasn't been recycled yet.
func (r *SocketRef) Write(b []byte) (int, error) {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return 0, io.EOF
	}

	return r.s.Write(b)
}

// Close closes a socket only if it hasn't been recycled yet.
func (r *SocketRef) Close(reason ...CloseReason) error {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return io.EOF
	}

	return r.s.Close(reason...)
}

// SetDeadline sets deadline of a socket only if it hasn't been recycled yet.
func (r *SocketRef) SetDeadline(deadline time.Time) error {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return io.EOF
	}

	return r.s.SetDeadline(deadline)
}

// SetReadDeadline sets read deadline of a socket only if it hasn't been recycled yet.
func (r *SocketRef) SetReadDeadline(deadline time.Time) error {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return io.EOF
	}

	return r.s.SetReadDeadline(deadline)
}

// SetWriteDeadline sets write deadline of a socket only if it hasn't been recycled yet.
func (r *SocketRef) SetWriteDeadline(deadline time.Time) error {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return io.EOF
	}

	return r.s.SetWriteDeadline(deadline)
}

// RemoteAddress returns a remote address of the socket.
func (r *SocketRef) RemoteAddress() string {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return ""
	}

	return r.s.RemoteAddress()
}

// ConnectedAt returns an exact time the socket has connected.
func (r *SocketRef) ConnectedAt() time.Time {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return time.UnixMilli(0)
	}

	return r.s.ConnectedAt()
}

// OnClose registers a handler that is called when underlying TCP connection is being closed.
func (r *SocketRef) OnClose(handler SocketCloseHandler) {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return
	}

}

// Unwrap returns underlying net.Conn instance from Socket.
func (r *SocketRef) Unwrap() net.Conn {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return nil
	}

	return r.s.Unwrap()
}

// UnwrapTLS tries to return underlying tls.Conn instance from Socket.
func (r *SocketRef) UnwrapTLS() (*tls.Conn, bool) {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return nil, false
	}

	return r.s.UnwrapTLS()
}

// WrapReader allows to wrap reader object into user defined wrapper.
func (r *SocketRef) WrapReader(wrapper func(io.Reader) io.Reader) {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return
	}

	r.s.WrapReader(wrapper)
}

// WrapWriter allows to wrap writer object into user defined wrapper.
func (r *SocketRef) WrapWriter(wrapper func(io.Writer) io.Writer) {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return
	}

	r.s.WrapWriter(wrapper)
}

// TotalRead returns a total number of bytes read through this socket.
func (r *SocketRef) TotalRead() uint64 {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return 0
	}

	return r.s.TotalRead()
}

// ReadLastSecond returns a total number of bytes read from this socket last second.
func (r *SocketRef) ReadLastSecond() uint64 {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return 0
	}

	return r.s.ReadLastSecond()
}

// TotalWritten returns a total number of bytes written through this socket.
func (r *SocketRef) TotalWritten() uint64 {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return 0
	}

	return r.s.TotalWritten()
}

// WrittenLastSecond returns a total number of bytes written to this socket last second.
func (r *SocketRef) WrittenLastSecond() uint64 {
	r.m.RLock()
	defer r.m.RUnlock()

	if r.s == nil {
		return 0
	}

	return r.s.WrittenLastSecond()
}

func (r *SocketRef) onRecycle() {
	r.m.Lock()
	defer r.m.Unlock()

	r.s = nil
}
