package tinytcp

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadByte(t *testing.T) {
	// given
	var buffer bytes.Buffer

	value := byte('A')

	// when then
	err := WriteByte(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadByte(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadBytes(t *testing.T) {
	// given
	var buffer bytes.Buffer

	value := []byte("AAA")

	// when then
	err := WriteBytes(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadBytes(&buffer, len(value))
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadBool(t *testing.T) {
	// given
	var buffer bytes.Buffer

	value := true

	// when then
	err := WriteBool(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadBool(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadVarInt(t *testing.T) {
	// given
	var buffer bytes.Buffer

	value := 12345

	// when then
	err := WriteVarInt(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadVarInt(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadVarLong(t *testing.T) {
	// given
	var buffer bytes.Buffer

	var value int64 = 12345

	// when then
	err := WriteVarLong(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadVarLong(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadInt16(t *testing.T) {
	// given
	var buffer bytes.Buffer

	var value int16 = 12345

	// when then
	err := WriteInt16(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadInt16(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadInt32(t *testing.T) {
	// given
	var buffer bytes.Buffer

	var value int32 = 12345

	// when then
	err := WriteInt32(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadInt32(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadInt64(t *testing.T) {
	// given
	var buffer bytes.Buffer

	var value int64 = 12345

	// when then
	err := WriteInt64(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadInt64(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadFloat32(t *testing.T) {
	// given
	var buffer bytes.Buffer

	var value float32 = 123.45

	// when then
	err := WriteFloat32(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadFloat32(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadFloat64(t *testing.T) {
	// given
	var buffer bytes.Buffer

	value := 123.45

	// when then
	err := WriteFloat64(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadFloat64(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadByteArray(t *testing.T) {
	// given
	var buffer bytes.Buffer

	value := []byte("Hello world")

	// when then
	err := WriteByteArray(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadByteArray(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}

func TestReadString(t *testing.T) {
	// given
	var buffer bytes.Buffer

	value := "Hello world"

	// when then
	err := WriteString(&buffer, value)
	if err != nil {
		assert.Nil(t, err, "write err should be nil")
	}

	readValue, err := ReadString(&buffer)
	if err != nil {
		assert.Nil(t, err, "read err should be nil")
	}

	assert.Equal(t, value, readValue, "values should match")
}
