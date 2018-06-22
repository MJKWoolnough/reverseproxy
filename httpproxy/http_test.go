package httpproxy

import (
	"bytes"
	"testing"

	"vimagination.zapto.org/memio"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

func TestHTTPRead(t *testing.T) {
	buf := buffer.Get()
	for n, test := range [...]struct {
		Input  memio.Buffer
		Output []byte
		Error  error
	}{
		{
			memio.Buffer("GET / HTTP/1.1\r\nHost: host.com\r\n\r\n"),
			[]byte("host.com"),
			nil,
		},
		{
			memio.Buffer("GET / HTTP/1.1\r\nHeader1: data1\r\nHost: host2.com\r\nAnother: header\r\n\r\n"),
			[]byte("host2.com"),
			nil,
		},
	} {
		_, name, err := Service.GetServerName(&test.Input, buf[:0])
		if err != test.Error {
			t.Errorf("test %d: expecting error %v, got %v", n+1, test.Error, err)
		} else if !bytes.Equal(name, test.Output) {
			t.Errorf("test %d: expecting name %q, got %q", n+1, test.Output, name)
		}
	}
	buffer.Put(buf)
}
