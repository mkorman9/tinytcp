package tinytcp

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

func TestGoroutinePerConnection(t *testing.T) {
	// given
	socket := MockSocket(nil, io.Discard)
	parentGoroutineID := getGoroutineID()
	childGoroutineID := parentGoroutineID

	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(s *Socket) {
		assert.Equal(t, socket, s, "socket should be passed to handler")
		childGoroutineID = getGoroutineID()
		wg.Done()
	}

	// when
	GoroutinePerConnection(handler).OnAccept(socket)
	wg.Wait()
	
	// then
	assert.NotEqual(t, parentGoroutineID, childGoroutineID, "handler should be run on different goroutine")
}

func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
