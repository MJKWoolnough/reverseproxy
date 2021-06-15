package reverseproxy

import (
	"io"
	"testing"
)

type delayReader []byte

func (d *delayReader) Read(p []byte) (int, error) {
	if len(*d) == 0 {
		return 0, io.EOF
	}
	p[0] = (*d)[0]
	*d = (*d)[1:]
	return 1, nil
}

func TestHTTP(t *testing.T) {
	stra := "0000000000000000000000000000000000000\r\n111111111111111111111111111111111111\r\nHost\r\nHost: example.com\r\n"
	data := delayReader(stra[1:])
	buf := make([]byte, 1024)
	buf[0] = '0'
	name, b, err := readHTTPServerName(&data, buf)
	if err != nil {
		t.Errorf("test 1: unexpected error: %s", err)
		return
	} else if len(b) != len(stra) {
		t.Errorf("test 1: expected to read %d bytes, read %d", len(stra), len(b))
		return
	} else if name != "example.com" {
		t.Errorf("test 1: expected hostname \"example.com\", got %q", name)
		return
	}
}
