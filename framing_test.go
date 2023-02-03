package tinytcp

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestFramingHandlerSimple(t *testing.T) {
	// given
	in := bytes.NewBuffer(generateTestPayloadWithSeparator(128))
	socket := MockSocket(in, io.Discard)

	// when
	var receivedPackets int

	PacketFramingHandler(
		SplitBySeparator([]byte{'\n'}),
		func(providedSocket *Socket) PacketHandler {
			// then
			assert.Equal(t, socket, providedSocket, "sockets must match")

			return func(packet []byte) {
				receivedPackets++
				assert.True(t, validateTestPayload(128, packet), "packet should be valid")
			}
		},
	)(socket)

	assert.Equal(t, 1, receivedPackets, "received packets count must match")
}

func TestFramingHandlerTwoPackets(t *testing.T) {
	// given
	in := bytes.NewBuffer(bytes.Join(
		[][]byte{generateTestPayloadWithSeparator(128), generateTestPayloadWithSeparator(128)},
		nil,
	))
	socket := MockSocket(in, io.Discard)

	// when
	var receivedPackets int

	PacketFramingHandler(
		SplitBySeparator([]byte{'\n'}),
		func(providedSocket *Socket) PacketHandler {
			// then
			assert.Equal(t, socket, providedSocket, "sockets must match")

			return func(packet []byte) {
				receivedPackets++
				assert.True(t, validateTestPayload(128, packet), "packet should be valid")
			}
		},
	)(socket)

	assert.Equal(t, 2, receivedPackets, "received packets count must match")
}

func TestFramingHandlerFragmentedPacket(t *testing.T) {
	// given
	in := bytes.NewBuffer(generateTestPayloadWithSeparator(1024))
	socket := MockSocket(in, io.Discard)

	// when
	var receivedPackets int

	PacketFramingHandler(
		SplitBySeparator([]byte{'\n'}),
		func(providedSocket *Socket) PacketHandler {
			// then
			assert.Equal(t, socket, providedSocket, "sockets must match")

			return func(packet []byte) {
				receivedPackets++
				assert.True(t, validateTestPayload(1024, packet), "packet should be valid")
			}
		},
		&PacketFramingConfig{
			ReadBufferSize: 512,
			MinReadSpace:   256,
		},
	)(socket)

	assert.Equal(t, 1, receivedPackets, "received packets count must match")
}

func TestFramingHandlerTwoFragmentedPackets(t *testing.T) {
	// given
	in := bytes.NewBuffer(bytes.Join(
		[][]byte{generateTestPayloadWithSeparator(512), generateTestPayloadWithSeparator(512)},
		nil,
	))
	socket := MockSocket(in, io.Discard)

	// when
	var receivedPackets int

	PacketFramingHandler(
		SplitBySeparator([]byte{'\n'}),
		func(providedSocket *Socket) PacketHandler {
			// then
			assert.Equal(t, socket, providedSocket, "sockets must match")

			return func(packet []byte) {
				receivedPackets++
				assert.True(t, validateTestPayload(512, packet), "packet should be valid")
			}
		},
		&PacketFramingConfig{
			ReadBufferSize: 768,
			MinReadSpace:   100,
		},
	)(socket)

	assert.Equal(t, 2, receivedPackets, "received packets count must match")
}

func TestFramingHandlerDelayedWriter(t *testing.T) {
	// given
	in := newDelayedReader(
		bytes.NewBuffer(bytes.Join(
			[][]byte{generateTestPayloadWithSeparator(128), generateTestPayloadWithSeparator(128)},
			nil,
		)),
		160, 200,
	)
	socket := MockSocket(in, io.Discard)

	// when
	var receivedPackets int

	PacketFramingHandler(
		SplitBySeparator([]byte{'\n'}),
		func(providedSocket *Socket) PacketHandler {
			// then
			assert.Equal(t, socket, providedSocket, "sockets must match")

			return func(packet []byte) {
				receivedPackets++
				assert.True(t, validateTestPayload(128, packet), "packet should be valid")
			}
		},
	)(socket)

	assert.Equal(t, 2, receivedPackets, "received packets count must match")
}

func TestFramingHandlerPacketTooBig(t *testing.T) {
	// given
	in := bytes.NewBuffer(generateTestPayloadWithSeparator(1024))
	socket := MockSocket(in, io.Discard)

	// when
	var receivedPackets int

	PacketFramingHandler(
		SplitBySeparator([]byte{'\n'}),
		func(providedSocket *Socket) PacketHandler {
			// then
			assert.Equal(t, socket, providedSocket, "sockets must match")

			return func(packet []byte) {
				receivedPackets++
			}
		},
		&PacketFramingConfig{
			MaxPacketSize: 512,
		},
	)(socket)

	assert.Equal(t, 0, receivedPackets, "received packets count must match")
}

func TestSeparatorFraming(t *testing.T) {
	// given
	protocol := SplitBySeparator([]byte{'\n'})
	payload := generateTestPayloadWithSeparator(128)

	// when
	packet, rest, extracted := protocol.ExtractPacket(payload)

	// then
	assert.True(t, extracted, "packet should be extracted")
	assert.True(t, validateTestPayload(128, packet), "packet should be valid")
	assert.Len(t, rest, 0, "packet should be only data in buffer")
}

func TestVarIntPrefixFraming(t *testing.T) {
	// given
	protocol := LengthPrefixedFraming(PrefixVarInt)
	payload := generateVarIntTestPayload(128)

	// when
	packet, rest, extracted := protocol.ExtractPacket(payload)

	// then
	assert.True(t, extracted, "packet should be extracted")
	assert.True(t, validateTestPayload(128, packet), "packet should be valid")
	assert.Len(t, rest, 0, "packet should be only data in buffer")
}

func TestVarLongPrefixFraming(t *testing.T) {
	// given
	protocol := LengthPrefixedFraming(PrefixVarLong)
	payload := generateVarIntTestPayload(128)

	// when
	packet, rest, extracted := protocol.ExtractPacket(payload)

	// then
	assert.True(t, extracted, "packet should be extracted")
	assert.True(t, validateTestPayload(128, packet), "packet should be valid")
	assert.Len(t, rest, 0, "packet should be only data in buffer")
}

func TestInt16PrefixFraming(t *testing.T) {
	// given
	protocol := LengthPrefixedFraming(PrefixInt16_BE)
	payload := generateInt16TestPayload(128)

	// when
	packet, rest, extracted := protocol.ExtractPacket(payload)

	// then
	assert.True(t, extracted, "packet should be extracted")
	assert.True(t, validateTestPayload(128, packet), "packet should be valid")
	assert.Len(t, rest, 0, "packet should be only data in buffer")
}

func TestInt32PrefixFraming(t *testing.T) {
	// given
	protocol := LengthPrefixedFraming(PrefixInt32_BE)
	payload := generateInt32TestPayload(128)

	// when
	packet, rest, extracted := protocol.ExtractPacket(payload)

	// then
	assert.True(t, extracted, "packet should be extracted")
	assert.True(t, validateTestPayload(128, packet), "packet should be valid")
	assert.Len(t, rest, 0, "packet should be only data in buffer")
}

func TestInt64PrefixFraming(t *testing.T) {
	// given
	protocol := LengthPrefixedFraming(PrefixInt64_BE)
	payload := generateInt64TestPayload(128)

	// when
	packet, rest, extracted := protocol.ExtractPacket(payload)

	// then
	assert.True(t, extracted, "packet should be extracted")
	assert.True(t, validateTestPayload(128, packet), "packet should be valid")
	assert.Len(t, rest, 0, "packet should be only data in buffer")
}

func generateTestPayloadWithSeparator(n int) []byte {
	var buff bytes.Buffer
	_ = WriteBytes(&buff, generateTestPayload(n))
	_ = WriteByte(&buff, '\n')
	return buff.Bytes()
}

func generateVarIntTestPayload(n int) []byte {
	var buff bytes.Buffer
	_ = WriteVarInt(&buff, n)
	_ = WriteBytes(&buff, generateTestPayload(n))
	return buff.Bytes()
}

func generateInt16TestPayload(n int) []byte {
	var buff bytes.Buffer
	_ = WriteInt16(&buff, int16(n))
	_ = WriteBytes(&buff, generateTestPayload(n))
	return buff.Bytes()
}

func generateInt32TestPayload(n int) []byte {
	var buff bytes.Buffer
	_ = WriteInt32(&buff, int32(n))
	_ = WriteBytes(&buff, generateTestPayload(n))
	return buff.Bytes()
}

func generateInt64TestPayload(n int) []byte {
	var buff bytes.Buffer
	_ = WriteInt64(&buff, int64(n))
	_ = WriteBytes(&buff, generateTestPayload(n))
	return buff.Bytes()
}

func generateTestPayload(n int) []byte {
	buff := make([]byte, n)
	for i := range buff {
		buff[i] = 'A'
	}
	return buff
}

func validateTestPayload(n int, payload []byte) bool {
	if len(payload) != n {
		return false
	}

	for _, v := range payload {
		if v != 'A' {
			return false
		}
	}

	return true
}

type delayedReader struct {
	reader   io.Reader
	schedule []int
	index    int
}

func newDelayedReader(reader io.Reader, schedule ...int) *delayedReader {
	return &delayedReader{
		reader:   reader,
		schedule: append(schedule, 0),
		index:    0,
	}
}

func (dr *delayedReader) Read(b []byte) (int, error) {
	chunkSize := dr.schedule[dr.index]
	if chunkSize == 0 {
		return 0, io.EOF
	}

	readBuffer := make([]byte, chunkSize)
	dr.index += 1

	n, err := dr.reader.Read(readBuffer)
	if err != nil {
		return n, err
	}

	copy(b, readBuffer[:n])
	return n, err
}
