package tinytcp

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// ReadBytes reads bytes from given reader.
func ReadBytes(reader io.Reader, n int) ([]byte, error) {
	buffer := make([]byte, n)
	remainingBytes := n

	for remainingBytes > 0 {
		bytesRead, err := reader.Read(buffer[n-remainingBytes:])
		if err != nil {
			return nil, err
		}

		remainingBytes -= bytesRead
	}

	return buffer, nil
}

// ReadByte reads byte from given reader.
func ReadByte(reader io.Reader) (byte, error) {
	buff, err := ReadBytes(reader, 1)
	if err != nil {
		return 0, err
	}

	return buff[0], nil
}

// ReadBool reads bool from given reader.
func ReadBool(reader io.Reader) (bool, error) {
	value, err := ReadByte(reader)
	if err != nil {
		return false, err
	}

	if value > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

// ReadInt16 reads int16 from given reader.
func ReadInt16(reader io.Reader, byteOrder ...binary.ByteOrder) (int16, error) {
	b, err := ReadBytes(reader, 2)
	if err != nil {
		return 0, err
	}

	var order binary.ByteOrder = binary.BigEndian
	if len(byteOrder) > 0 {
		order = byteOrder[0]
	}

	return int16(order.Uint16(b)), nil
}

// ReadInt32 reads int32 from given reader.
func ReadInt32(reader io.Reader, byteOrder ...binary.ByteOrder) (int32, error) {
	b, err := ReadBytes(reader, 4)
	if err != nil {
		return 0, err
	}

	var order binary.ByteOrder = binary.BigEndian
	if len(byteOrder) > 0 {
		order = byteOrder[0]
	}

	return int32(order.Uint32(b)), nil
}

// ReadInt64 reads int64 from given reader.
func ReadInt64(reader io.Reader, byteOrder ...binary.ByteOrder) (int64, error) {
	b, err := ReadBytes(reader, 8)
	if err != nil {
		return 0, err
	}

	var order binary.ByteOrder = binary.BigEndian
	if len(byteOrder) > 0 {
		order = byteOrder[0]
	}

	return int64(order.Uint64(b)), nil
}

// ReadVarInt reads var int from given reader.
func ReadVarInt(reader io.Reader) (int, error) {
	var value int
	var position int

	for {
		currentByte, err := ReadByte(reader)
		if err != nil {
			return 0, err
		}

		value |= int(currentByte) & segmentBits << position

		if (int(currentByte) & continueBit) == 0 {
			break
		}

		position += 7

		if position >= 32 {
			return 0, errors.New("invalid size of VarInt")
		}
	}

	return value, nil
}

// ReadVarLong reads var int64 from given reader.
func ReadVarLong(reader io.Reader) (int64, error) {
	var value int64
	var position int64

	for {
		currentByte, err := ReadByte(reader)
		if err != nil {
			return 0, err
		}

		value |= int64(currentByte) & int64(segmentBits) << position

		if (int(currentByte) & continueBit) == 0 {
			break
		}

		position += 7

		if position >= 64 {
			return 0, errors.New("invalid size of VarLong")
		}
	}

	return value, nil
}

// ReadFloat32 reads float32 from given reader.
func ReadFloat32(reader io.Reader, byteOrder ...binary.ByteOrder) (float32, error) {
	value, err := ReadInt32(reader, byteOrder...)
	if err != nil {
		return 0, err
	}

	return math.Float32frombits(uint32(value)), nil
}

// ReadFloat64 reads float64 from given reader.
func ReadFloat64(reader io.Reader, byteOrder ...binary.ByteOrder) (float64, error) {
	value, err := ReadInt64(reader, byteOrder...)
	if err != nil {
		return 0, err
	}

	return math.Float64frombits(uint64(value)), nil
}

// ReadByteArray reads byte array from given reader.
func ReadByteArray(reader io.Reader) ([]byte, error) {
	length, err := ReadVarInt(reader)
	if err != nil {
		return nil, err
	}

	return ReadBytes(reader, length)
}

// ReadString reads string from given reader.
func ReadString(reader io.Reader) (string, error) {
	b, err := ReadByteArray(reader)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
