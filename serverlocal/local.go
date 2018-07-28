package local

import (
	"net"

	"vimagination.zapto.org/reverseproxy"
	"vimagination.zapto.org/reverseproxy/internal/addr"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
	"vimagination.zapto.org/reverseproxy/internal/conn"
)

type listener struct {
	req chan net.Conn
}

func Listen(serverName string, p *reverseproxy.Proxy) (net.Listener, error) {
	l := &listener{
		req: make(chan net.Conn),
	}
	if err := p.Add(serverName, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *listener) Handle(c net.Conn, buf *buffer.Buffer, length int) {
	l.req <- conn.New(c, buf, length)
}

func (l *listener) Stop() {
	close(l.req)
}

func (l *listener) Accept() (net.Conn, error) {
	c, ok := <-l.req
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (l *listener) Addr() net.Addr {
	return localAddr
}

func (l *listener) Close() error {
	close(l.req)
	return nil
}

var localAddr net.Addr = addr.Addr{"local", "local"}
