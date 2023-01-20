package tinytcp

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestSocketInput(t *testing.T) {
	// given
	payload := []byte("Hello world!")
	payloadSize := len(payload)

	in := bytes.NewBuffer(payload)
	socket := MockSocket(in, io.Discard)

	// when
	buffer := make([]byte, payloadSize)
	n, err := socket.Read(buffer)

	// then
	assert.Nil(t, err, "err should be nil")
	assert.Equal(t, payloadSize, n, "n should equal to bytes read")
	assert.Equal(t, payload, buffer, "payloads should match")
}

func TestSocketInputEOF(t *testing.T) {
	// given
	socket := MockSocket(&eofReader{}, io.Discard)

	var closeHandlerCalled bool
	socket.OnClose(func(reason CloseReason) {
		closeHandlerCalled = true
		assert.Equal(t, CloseReasonClient, reason, "close reason should be correct")
	})

	// when
	_, err := socket.Read(nil)

	// then
	assert.Error(t, io.EOF, err, "err should be equal to io.EOF")
	assert.Truef(t, closeHandlerCalled, "close handler should be called")
}

func TestSocketOutput(t *testing.T) {
	// given
	payload := []byte("Hello world")
	payloadSize := len(payload)

	var out bytes.Buffer
	socket := MockSocket(nil, &out)

	// when
	n, err := socket.Write(payload)

	// then
	assert.Nil(t, err, "err should be nil")
	assert.Equal(t, payloadSize, n, "n should equal to bytes read")
	assert.Equal(t, payload, out.Bytes(), "payloads should match")
}

func TestSocketOutputEOF(t *testing.T) {
	// given
	socket := MockSocket(nil, &eofWriter{})

	var closeHandlerCalled bool
	socket.OnClose(func(reason CloseReason) {
		closeHandlerCalled = true
		assert.Equal(t, CloseReasonClient, reason, "close reason should be correct")
	})

	// when
	_, err := socket.Write(nil)

	// then
	assert.Error(t, io.EOF, err, "err should be equal to io.EOF")
	assert.Truef(t, closeHandlerCalled, "close handler should be called")
}

type eofReader struct {
}

func (er *eofReader) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

type eofWriter struct {
}

func (ew *eofWriter) Write(_ []byte) (int, error) {
	return 0, io.EOF
}
