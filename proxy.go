package reverseproxy // import "vimagination.zapto.org/reverseproxy"

import (
	"fmt"
	"net"
	"sync"
)

var listeners = make(map[uint16]*listener)

type listener struct {
	net.Listener

	mu    sync.RWMutex
	ports map[*port]struct{}
}

func (l *listener) listen() {

}

func (l *listener) close() {

}

type conn struct {
	buffer []byte
	conn   net.Conn
}

type transfer interface {
	transfer(*conn, *port)
}

type service struct {
	matchServiceName
	transferer
}

func registerService(serviceName matchServiceName, transfer transferer) *service {
	return &service{
		matchServiceName: serviceName,
		transferer:       transfer,
	}
}

func (s *service) close() {
}

type port struct {
	*service
	port uint16
}

func (s *service) AddPort(port uint16) (*port, error) {
	l, ok := listeners[port]
	if !ok {
		nl, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return nil, err
		}
		l = &listener{
			Listener: nl,
			ports:    make(map[*port]struct{}),
		}
		go l.listen()
	}
	port := &port{
		service: s,
		port:    uint16,
	}
	l.mu.Lock()
	l.ports[port] = struct{}{}
	l.mu.Unlock()
	return port, nil
}

func (p *port) close() {
	l, ok := listeners[port]
	if !ok {
		return
	}
	l.mu.Lock()
	delete(l.ports, p)
	if len(l.ports) == 0 {
		delete(listeners, p.port)
		go l.close()
	}
	l.mu.Unlock()
}
