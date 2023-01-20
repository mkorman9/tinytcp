package tinytcp

import (
	"io"
	"sync"
	"sync/atomic"
)

// SocketReference allows to hold a reference to a socket outside its designated handler.
// tinytcp performs no memory allocations on critical paths, which means it needs to pool some objects.
// These pooled objects are typically structures that exist only in the context of a specific connection (eg. Sockets).
// Objects are returned to the pool as they are no longer needed and reused by other connection in the future.
// The rule is that a socket instance is only valid inside its designated handler and storing it outside this handler
// might result in some very nasty bugs. SocketReference provides a way to safely store a reference to a socket,
// and provide a subset of its functionalities.
type SocketReference struct {
	s        *Socket
	m        sync.RWMutex
	recycled uint32
}

// NewSocketHolder creates an instance of SocketReference.
func NewSocketHolder(s *Socket) *SocketReference {
	ref := &SocketReference{
		s: s,
	}

	s.onRecycle(ref.onRecycle)
	return ref
}

// Write writes data to a socket only if it hasn't been recycled yet.
func (s *SocketReference) Write(b []byte) (int, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	if atomic.LoadUint32(&s.recycled) == 1 {
		return 0, io.EOF
	}

	return s.s.Write(b)
}

func (s *SocketReference) onRecycle() {
	s.m.Lock()
	defer s.m.Unlock()

	atomic.StoreUint32(&s.recycled, 1)
}
