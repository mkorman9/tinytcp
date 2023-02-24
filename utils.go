package tinytcp

import (
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// PrefixType denotes the type of the prefix used to specify packet length.
type PrefixType int

const (
	// PrefixVarInt represents a VarInt prefix.
	PrefixVarInt PrefixType = iota

	// PrefixVarLong represents a VarLong prefix.
	PrefixVarLong

	// PrefixInt16_BE 16-bit prefix (Big Endian).
	PrefixInt16_BE

	// PrefixInt16_LE 16-bit prefix (Little Endian).
	PrefixInt16_LE

	// PrefixInt32_BE 32-bit prefix (Big Endian).
	PrefixInt32_BE

	// PrefixInt32_LE 32-bit prefix (Little Endian).
	PrefixInt32_LE

	// PrefixInt64_BE 64-bit prefix (Big Endian).
	PrefixInt64_BE

	// PrefixInt64_LE 64-bit prefix (Little Endian).
	PrefixInt64_LE
)

// Size returns a size of given prefix in bytes, and -1 in case of types with undefined size.
func (p PrefixType) Size() int {
	switch p {
	case PrefixVarInt:
		fallthrough
	case PrefixVarLong:
		return -1
	case PrefixInt16_BE:
		fallthrough
	case PrefixInt16_LE:
		return 2
	case PrefixInt32_BE:
		fallthrough
	case PrefixInt32_LE:
		return 4
	case PrefixInt64_BE:
		fallthrough
	case PrefixInt64_LE:
		return 8
	}

	return -1
}

// CloseReason denotes a reason that Close() function has been called for.
// Close() can be triggered either by server, or by client (connection reset by peer).
type CloseReason int

const (
	// CloseReasonServer means the connection has been closed intentionally on the server side.
	CloseReasonServer CloseReason = iota

	// CloseReasonClient means the connection has been either closed by client or has been lost for other reasons.
	CloseReasonClient
)

const (
	segmentBits = 0x7F
	continueBit = 0x80
)

func isBrokenPipe(err error) bool {
	return err == io.EOF ||
		errors.Is(err, syscall.ECONNRESET) ||
		strings.Contains(err.Error(), "use of closed network connection") ||
		strings.Contains(err.Error(), "wsarecv: An existing connection was forcibly closed by the remote host.") ||
		strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "reset by peer") ||
		strings.Contains(err.Error(), "unexpected EOF") ||
		strings.Contains(err.Error(), "read/write on closed pipe")
}

func isTimeout(err error) bool {
	return errors.Is(err, os.ErrDeadlineExceeded)
}

func parseRemoteAddress(connection net.Conn) string {
	address := connection.RemoteAddr().String()
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}

	return host
}

func resolveNetworkPort(address net.Addr) int {
	a := address.String()
	_, portRaw, err := net.SplitHostPort(a)
	if err != nil {
		return 0
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil {
		return 0
	}

	return port
}
