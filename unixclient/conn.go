package unixclient

import (
	"net"
	"runtime"

	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

type Conn struct {
	net.Conn
	buffer      *buffer.Buffer
	pos, length int
}

func (c *Conn) Read(b []byte) (int, error) {
	if c.buffer != nil {
		n := copy(b, c.buffer[c.pos:c.length])
		c.pos += n
		if c.pos >= c.length {
			buffer.Put(c.buffer)
			c.buffer = nil
		}
		if n > 0 {
			return n, nil
		}
	}
	return c.Conn.Read(b)
}

func (c *Conn) Close() error {
	runtime.SetFinalizer(c, nil)
	if c.buffer != nil {
		buffer.Put(c.buffer)
		c.buffer = nil
	}
	return c.Conn.Close()
}

func (c *Conn) LocalConn() net.Addr {
	if c.Conn == nil {
		return fakeAddr{}
	}
	r := c.Conn.LocalAddr()
	if r == nil {
		return fakeAddr{}
	}
	return r
}

func (c *Conn) RemoteConn() net.Addr {
	if c.Conn == nil {
		return fakeAddr{}
	}
	r := c.Conn.RemoteAddr()
	if r == nil {
		return fakeAddr{}
	}
	return r
}

type fakeAddr struct{}

func (fakeAddr) Network() string {
	return ""
}

func (fakeAddr) String() string {
	return ""
}
