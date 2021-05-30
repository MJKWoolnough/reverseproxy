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
	ports map[*port]struct{}
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
		var port *port
		mu.RLock()
		for p := range l.ports {
			if p.MatchService(name) {
				port = p
				break
			}
		}
		mu.RUnlock()
		port.transfer(&conn{
			buffer: buf,
			conn:   c,
		}, port)
	}
}

type conn struct {
	buffer []byte
	conn   net.Conn
}

type transferer interface {
	transfer(*conn, *port)
}

type service struct {
	matchServiceName
	transferer
	ports map[uint16]*port
}

func registerService(serviceName matchServiceName, transfer transferer) *service {
	return &service{
		matchServiceName: serviceName,
		transferer:       transfer,
		ports:            make(map[uint16]*port),
	}
}

func (s *service) close() {
	mu.Lock()
	defer mu.Unlock()
	for _, p := range s.ports {
		p.close()
	}
}

type port struct {
	*service
	port   uint16
	closed bool
}

func (s *service) AddPort(addPort uint16) (*port, error) {
	if addPort == 0 {
		return nil, ErrInvalidPort
	}
	mu.Lock()
	defer mu.Unlock()
	if p, ok := s.ports[addPort]; ok {
		return p, nil
	}
	l, ok := listeners[addPort]
	if !ok {
		nl, err := net.Listen("tcp", fmt.Sprintf(":%d", addPort))
		if err != nil {
			return nil, err
		}
		l = &listener{
			Listener: nl,
			ports:    make(map[*port]struct{}),
		}
		go l.listen()
	}
	p := &port{
		service: s,
		port:    addPort,
	}
	s.ports[addPort] = p
	l.ports[p] = struct{}{}
	return p, nil
}

func (p *port) close() {
	mu.Lock()
	defer mu.Unlock()
	if p.closed {
		return
	}
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

func (p *port) Closed() bool {
	return p.closed
}

// Errors
var (
	ErrInvalidPort = errors.New("cannot register on port 0")
)
