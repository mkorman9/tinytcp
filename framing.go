package tinytcp

import (
	"bytes"
	"encoding/binary"
	"sync"
)

// PacketHandler is a function to be called after receiving packet data.
type PacketHandler func(packet []byte)

// FramingProtocol defines a strategy of extracting meaningful chunks of data out of read buffer.
type FramingProtocol interface {
	// ExtractPacket splits the source buffer into packet and "the rest".
	// Returns extracted == true if the meaningful packet has been extracted.
	ExtractPacket(source []byte) (packet []byte, rest []byte, extracted bool)
}

type separatorFramingProtocol struct {
	separator []byte
}

type lengthPrefixedFramingProtocol struct {
	prefixType   PrefixType
	prefixLength int
}

// PacketFramingConfig hold configuration for PacketFramingHandler.
type PacketFramingConfig struct {
	// ReadBufferSize sets a size of read buffer (default: 4KiB).
	ReadBufferSize int

	// MaxPacketSize sets a maximal size of a packet (default: 16KiB).
	MaxPacketSize int

	// MinReadSpace sets a minimal space in read buffer that's needed to fit another Read() into it,
	// without allocating auxiliary buffer (default: 1KiB or 1/4 of ReadBufferSize).
	MinReadSpace int
}

func mergePacketFramingConfig(provided *PacketFramingConfig) *PacketFramingConfig {
	config := &PacketFramingConfig{
		ReadBufferSize: 4 * 1024,  // 4 KiB
		MaxPacketSize:  16 * 1024, // 16 KiB
		MinReadSpace:   1024,      // 1 KiB
	}

	if provided == nil {
		return config
	}

	if provided.ReadBufferSize > 0 {
		config.ReadBufferSize = provided.ReadBufferSize
	}
	if provided.MaxPacketSize > 0 {
		config.MaxPacketSize = provided.MaxPacketSize
	}
	if provided.MinReadSpace > 0 {
		config.MinReadSpace = provided.MinReadSpace
	}

	if config.MinReadSpace > config.ReadBufferSize {
		config.MinReadSpace = config.ReadBufferSize / 4
	}

	return config
}

// PacketFramingHandler returns a SocketHandler that handles packet framing according to given FramingProtocol.
func PacketFramingHandler(
	framingProtocol FramingProtocol,
	socketHandler func(socket *Socket) PacketHandler,
	config ...*PacketFramingConfig,
) SocketHandler {
	var providedConfig *PacketFramingConfig
	if config != nil {
		providedConfig = config[0]
	}
	c := mergePacketFramingConfig(providedConfig)

	// common buffers are pooled to avoid memory allocation in hot path
	var (
		readBufferPool = sync.Pool{
			New: func() any {
				return make([]byte, c.ReadBufferSize)
			},
		}
		receiveBufferPool = sync.Pool{
			New: func() any {
				return &bytes.Buffer{}
			},
		}
	)

	return func(socket *Socket) {
		packetHandler := socketHandler(socket)

		var (
			// readBuffer is a fixed-size page, which is never reallocated. Socket pumps data straight into it.
			readBuffer = readBufferPool.Get().([]byte)

			// receiveBuffer is used to hold data between consecutive Read() calls in case a packet is fragmented.
			receiveBuffer *bytes.Buffer

			// leftOffset indicates a place in read buffer after the last, already handled packet.
			leftOffset int

			// rightOffset indicates a place in read buffer in which the next Read() will occur.
			rightOffset int
		)

		defer func() {
			readBufferPool.Put(readBuffer)

			if receiveBuffer != nil {
				receiveBuffer.Reset()
				receiveBufferPool.Put(receiveBuffer)
			}
		}()

		for {
			bytesRead, err := socket.Read(readBuffer[rightOffset:])
			if err != nil {
				if socket.IsClosed() {
					break
				}

				continue
			}

			// validate packet size
			if c.MaxPacketSize > 0 {
				memoryUsed := rightOffset + bytesRead - leftOffset
				if receiveBuffer != nil {
					memoryUsed += receiveBuffer.Len()
				}

				if memoryUsed > c.MaxPacketSize {
					// packet too big
					if receiveBuffer != nil {
						receiveBuffer.Reset()
					}

					leftOffset = 0
					rightOffset = 0
					continue
				}
			}

			// include data from past iteration if receive buffer is not empty
			source := readBuffer[leftOffset : rightOffset+bytesRead]
			if receiveBuffer != nil && receiveBuffer.Len() > 0 {
				receiveBuffer.Write(source)
				source = receiveBuffer.Bytes()
				receiveBuffer.Reset()
			}

			for {
				packet, rest, extracted := framingProtocol.ExtractPacket(source)
				if extracted {
					// fast path - packet is extracted straight from the readBuffer, without memory allocations
					excessBytes := len(source) - len(packet) - len(rest)
					leftOffset += len(packet) + excessBytes
					rightOffset += len(packet) + excessBytes
					source = rest

					packetHandler(packet)
				} else {
					if len(source) == 0 {
						leftOffset = 0
						rightOffset = 0
						break
					}

					// packet is fragmented

					if rightOffset+len(source) > len(readBuffer)-c.MinReadSpace {
						// slow path - memory allocation needed
						if receiveBuffer == nil {
							receiveBuffer = receiveBufferPool.Get().(*bytes.Buffer)
						}

						receiveBuffer.Write(source)
						leftOffset = 0
						rightOffset = 0
					} else {
						// we'll still fit another Read() into read buffer
						rightOffset += len(source)
					}

					break
				}
			}
		}
	}
}

// SplitBySeparator is a FramingProtocol strategy that expects each packet to end with a sequence of bytes given as
// separator. It is a good strategy for tasks like handling Telnet sessions (packets are separated by a newline).
func SplitBySeparator(separator []byte) FramingProtocol {
	return &separatorFramingProtocol{
		separator: separator,
	}
}

func (s *separatorFramingProtocol) ExtractPacket(buffer []byte) ([]byte, []byte, bool) {
	return bytes.Cut(buffer, s.separator)
}

// LengthPrefixedFraming is a FramingProtocol that expects each packet to be prefixed with its length in bytes.
// Length is expected to be provided as binary encoded number with size and endianness specified by value provided
// as prefixType argument.
func LengthPrefixedFraming(prefixType PrefixType) FramingProtocol {
	var prefixLength int

	switch prefixType {
	case PrefixInt16_BE:
		fallthrough
	case PrefixInt16_LE:
		prefixLength = 2
	case PrefixInt32_BE:
		fallthrough
	case PrefixInt32_LE:
		prefixLength = 4
	case PrefixInt64_BE:
		fallthrough
	case PrefixInt64_LE:
		prefixLength = 8
	}

	return &lengthPrefixedFramingProtocol{
		prefixType:   prefixType,
		prefixLength: prefixLength,
	}
}

func (l *lengthPrefixedFramingProtocol) ExtractPacket(buffer []byte) ([]byte, []byte, bool) {
	var (
		prefixLength = l.prefixLength
		packetSize   int64
	)

	if len(buffer) >= prefixLength {
		switch l.prefixType {
		case PrefixVarInt:
			valueRead := false
			prefixLength, packetSize, valueRead = readVarIntPacketSize(buffer)
			if !valueRead {
				return nil, buffer, false
			}
		case PrefixVarLong:
			valueRead := false
			prefixLength, packetSize, valueRead = readVarLongPacketSize(buffer)
			if !valueRead {
				return nil, buffer, false
			}
		case PrefixInt16_BE:
			packetSize = int64(binary.BigEndian.Uint16(buffer[:prefixLength]))
		case PrefixInt16_LE:
			packetSize = int64(binary.LittleEndian.Uint16(buffer[:prefixLength]))
		case PrefixInt32_BE:
			packetSize = int64(binary.BigEndian.Uint32(buffer[:prefixLength]))
		case PrefixInt32_LE:
			packetSize = int64(binary.LittleEndian.Uint32(buffer[:prefixLength]))
		case PrefixInt64_BE:
			packetSize = int64(binary.BigEndian.Uint64(buffer[:prefixLength]))
		case PrefixInt64_LE:
			packetSize = int64(binary.LittleEndian.Uint64(buffer[:prefixLength]))
		}
	} else {
		return nil, buffer, false
	}

	if int64(len(buffer[prefixLength:])) >= packetSize {
		buffer = buffer[prefixLength:]
		return buffer[:packetSize], buffer[packetSize:], true
	} else {
		return nil, buffer, false
	}
}

func readVarIntPacketSize(buffer []byte) (int, int64, bool) {
	var (
		value    int
		position int
		i        int
	)

	for {
		if i >= len(buffer) {
			return 0, 0, false
		}
		currentByte := buffer[i]

		value |= int(currentByte) & segmentBits << position
		if (int(currentByte) & continueBit) == 0 {
			break
		}

		position += 7
		if position >= 32 {
			return 0, 0, false
		}

		i++
	}

	return i + 1, int64(value), true
}

func readVarLongPacketSize(buffer []byte) (int, int64, bool) {
	var (
		value    int64
		position int
		i        int
	)

	for {
		if i >= len(buffer) {
			return 0, 0, false
		}
		currentByte := buffer[i]

		value |= int64(currentByte) & int64(segmentBits) << position
		if (int(currentByte) & continueBit) == 0 {
			break
		}

		position += 7
		if position >= 64 {
			return 0, 0, false
		}

		i++
	}

	return i + 1, value, true
}
