package unixclient

import (
	"bytes"
	"io"
	"testing"

	"vimagination.zapto.org/ioconn"
	"vimagination.zapto.org/memio"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

func TestConn(t *testing.T) {
	var connData memio.Buffer
	conn := Conn{
		Conn: &ioconn.Conn{
			Reader: &connData,
			Writer: &memio.Buffer{},
			Closer: ioconn.CloserFunc(func() error { return nil }),
			Local:  ioconn.Addr{},
			Remote: ioconn.Addr{},
		},
	}
	for n, test := range [...]struct {
		BufferData, ConnData, ReadBuf []byte
		Output                        [][]byte
	}{
		{
			[]byte("123"),
			[]byte("ABC"),
			make([]byte, 3),
			[][]byte{
				[]byte("123"),
				[]byte("ABC"),
			},
		},
		{
			[]byte("123"),
			[]byte("ABC"),
			make([]byte, 4),
			[][]byte{
				[]byte("123"),
				[]byte("ABC"),
			},
		},
		{
			[]byte("123"),
			[]byte("ABC"),
			make([]byte, 2),
			[][]byte{
				[]byte("12"),
				[]byte("3"),
				[]byte("AB"),
				[]byte("C"),
			},
		},
		{
			[]byte{},
			[]byte("A"),
			make([]byte, 2),
			[][]byte{
				[]byte("A"),
			},
		},
		{
			[]byte{},
			[]byte{},
			make([]byte, 2),
			nil,
		},
	} {
		conn.buffer = buffer.Get()
		copy(conn.buffer[:len(test.BufferData)], test.BufferData)
		conn.pos = 0
		conn.length = len(test.BufferData)
		connData = test.ConnData
		for m, result := range test.Output {
			l, err := conn.Read(test.ReadBuf)
			if err != nil {
				t.Errorf("test %d.%d: got unexpected error: %s", n+1, m+1, err)
			} else if l != len(result) {
				t.Errorf("test %d.%d: expecting read length %d, got %d", n+1, m+1, len(result), l)
			} else if !bytes.Equal(test.ReadBuf[:l], result) {
				t.Errorf("test %d.%d: expecting to read %v, read %v", n+1, m+1, result, test.ReadBuf[:l])
			}
		}
		if len(test.Output) == 0 {
			conn.Close()
		}
		if conn.buffer != nil {
			t.Errorf("test %d: expecting buffer to be returned", n+1)
		} else if _, err := conn.Read(test.ReadBuf); err != io.EOF {
			t.Errorf("test %d: expecting EOF, got %s", n+1, err)
		}
	}
}
