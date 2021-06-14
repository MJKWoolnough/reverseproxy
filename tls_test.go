package reverseproxy

import (
	"bytes"
	"testing"

	"vimagination.zapto.org/memio"
)

var tlsBase = [...]byte{
	22,   // TLS Handshake
	3, 3, // Version
	0, 0, // Length of TLS Fragment
	1,       // CLIENT_HELLO
	0, 0, 0, // Length of body
	3, 3, // Version
	0, 0, 0, 0, // Unix Time
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, // Random
	0,    // Session ID
	0, 2, // Cipher Suite Length
	0, 0, // Cipher Suite
	1, 0, // Compression (Length + ) Method
}

func tlsServerName(name string) []byte {
	l := len(name)
	buf := make([]byte, len(tlsBase), len(tlsBase)+11+l)
	copy(buf, tlsBase[:])
	buf[3] = byte((56 + l) >> 8)
	buf[4] = byte(56 + l)
	buf[7] = byte((52 + l) >> 8)
	buf[8] = byte(52 + l)
	return append(append(buf, byte((l+9)>>8), byte(l+9), 0, 0, byte((l+5)>>8), byte(l+5), byte((l+3)>>8), byte(l+3), 0, byte(l>>8), byte(l)), name...)
}

func TestTLS(t *testing.T) {
	buf := tlsServerName("aaa.com")
	rBuf := make([]byte, 100)
	rBuf[0] = buf[0]
	aBuf := memio.Buffer(buf[1:])
	name, b, err := readTLSServerName(&aBuf, rBuf)
	if err != nil {
		t.Errorf("test 1: unexpected error, %s", err)
		return
	}
	if name != "aaa.com" {
		t.Errorf("test 1: expecting name \"aaa.com\", got %q", name)
		return
	}
	if !bytes.Equal(buf, b) {
		t.Errorf("test 1: expecting bytes %v, got %v", buf, b)
		return
	}
	buf = tlsServerName("example.com")
	rBuf[0] = buf[0]
	aBuf = memio.Buffer(buf[1:])
	name, b, err = readTLSServerName(&aBuf, rBuf)
	if err != nil {
		t.Errorf("test 2: unexpected error, %s", err)
		return
	}
	if name != "example.com" {
		t.Errorf("test 2: expecting name \"example.com\", got %q", name)
		return
	}
	if !bytes.Equal(buf, b) {
		t.Errorf("test 2: expecting bytes %v, got %v", buf, b)
		return
	}
}
