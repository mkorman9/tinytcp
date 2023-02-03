package tinytcp

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// ReadByte reads byte from given reader.
func ReadByte(reader io.Reader) (byte, error) {
	var buff [1]byte
	_, err := reader.Read(buff[:])
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
	var buff [2]byte
	_, err := reader.Read(buff[:])
	if err != nil {
		return 0, err
	}

	var order binary.ByteOrder = binary.BigEndian
	if len(byteOrder) > 0 {
		order = byteOrder[0]
	}

	return int16(order.Uint16(buff[:])), nil
}

// ReadInt32 reads int32 from given reader.
func ReadInt32(reader io.Reader, byteOrder ...binary.ByteOrder) (int32, error) {
	var buff [4]byte
	_, err := reader.Read(buff[:])
	if err != nil {
		return 0, err
	}

	var order binary.ByteOrder = binary.BigEndian
	if len(byteOrder) > 0 {
		order = byteOrder[0]
	}

	return int32(order.Uint32(buff[:])), nil
}

// ReadInt64 reads int64 from given reader.
func ReadInt64(reader io.Reader, byteOrder ...binary.ByteOrder) (int64, error) {
	var buff [8]byte
	_, err := reader.Read(buff[:])
	if err != nil {
		return 0, err
	}

	var order binary.ByteOrder = binary.BigEndian
	if len(byteOrder) > 0 {
		order = byteOrder[0]
	}

	return int64(order.Uint64(buff[:])), nil
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
