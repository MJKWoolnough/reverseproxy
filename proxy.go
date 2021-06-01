package reverseproxy // import "vimagination.zapto.org/reverseproxy"

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

var (
	mu        sync.RWMutex // global lock
	listeners = make(map[uint16]*listener)
)

type listener struct {
	net.Listener
	ports map[*Port]struct{}
}

func (l *listener) listen() {
	for {
		c, err := l.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		} else if err != nil {
			continue
		}
		var tlsByte [1]byte
		if n, err := io.ReadFull(c, tlsByte[:]); n != 1 || err != nil {
			c.Close()
		}
		var (
			name string
			buf  []byte
		)
		if tlsByte[0] == 22 {
			name, buf, err = readTLSServerName(c, 22)
			if err != nil {
				c.Close()
				continue
			}
		} else {
			name, buf, err = readHTTPServerName(c, tlsByte[0])
		}
		var port *Port
		mu.RLock()
		for p := range l.ports {
			if p.MatchService(name) {
				port = p
				break
			}
		}
		mu.RUnlock()
		port.Transfer(buf, c)
	}
}

type service interface {
	MatchService(string) bool
	Transfer([]byte, net.Conn)
}

type Port struct {
	service
	port   uint16
	closed bool
}

func addPort(port uint16, service service) (*Port, error) {
	if port == 0 {
		return nil, ErrInvalidPort
	}
	mu.Lock()
	defer mu.Unlock()
	l, ok := listeners[port]
	if !ok {
		nl, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return nil, err
		}
		l = &listener{
			Listener: nl,
			ports:    make(map[*Port]struct{}),
		}
		go l.listen()
	}
	p := &Port{
		service: service,
		port:    port,
	}
	l.ports[p] = struct{}{}
	return p, nil
}

func (p *Port) Close() error {
	mu.Lock()
	if !p.closed {
		l, ok := listeners[p.port]
		if ok {
			delete(l.ports, p)
			if len(l.ports) == 0 {
				delete(listeners, p.port)
				l.Close()
			}
		}
		p.closed = true
	}
	mu.Unlock()
	return nil
}

func (p *Port) Closed() bool {
	return p.closed
}

// Errors
var (
	ErrInvalidPort = errors.New("cannot register on port 0")
)
