package proxyclient

import (
	"net"
	"runtime"

	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

type Conn struct {
	net.Conn
	buffer buffer.Buffer
}

func (c *Conn) Read(b []byte) (int, error) {
	if len(c.buffer.LimitedBuffer) > 0 {
		n, err := c.buffer.Read(b)
		if len(c.buffer.LimitedBuffer) == 0 {
			c.buffer.Close()
		}
		return n, err
	}
	return c.Conn.Read(b)
}

func (c *Conn) Close() error {
	runtime.SetFinalizer(c, nil)
	c.buffer.Close()
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
