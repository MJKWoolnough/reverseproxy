package local

import (
	"net"
	"sync"

	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy"
	"vimagination.zapto.org/reverseproxy/internal/addr"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
	"vimagination.zapto.org/reverseproxy/internal/conn"
)

type listener struct {
	req chan net.Conn

	mu     sync.Mutex
	closed bool
}

func Listen(p *reverseproxy.Proxy, serverName string, aliases ...string) (net.Listener, error) {
	l := &listener{
		req: make(chan net.Conn),
	}
	if err := p.Add(serverName, l); err != nil {
		return nil, err
	}
	for _, alias := range aliases {
		if err := p.Add(alias, l); err != nil {
			return nil, err
		}
	}
	return l, nil
}

func (l *listener) Handle(c net.Conn, buf *buffer.Buffer, length int) {
	l.req <- conn.New(c, buf, length)
}

func (l *listener) Stop() {
	l.mu.Lock()
	if !l.closed {
		close(l.req)
		l.closed = true
	}
	l.mu.Unlock()
}

func (l *listener) Accept() (net.Conn, error) {
	c, ok := <-l.req
	if !ok {
		return nil, ErrClosed
	}
	return c, nil
}

func (l *listener) Addr() net.Addr {
	return localAddr
}

func (l *listener) Close() error {
	l.Stop()
	return nil
}

const (
	local                  = "local"
	ErrClosed errors.Error = "connection closed"
)

var localAddr net.Addr = addr.Addr{local, local}
