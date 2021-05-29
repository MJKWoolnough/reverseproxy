package reverseproxy // import "vimagination.zapto.org/reverseproxy"

import (
	"fmt"
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
	port uint16
}

func (s *service) AddPort(port uint16) (*port, error) {
	mu.Lock()
	defer mu.Unlock()
	if p, ok := s.ports[port]; ok {
		return p, nil
	}
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
	p := &port{
		service: s,
		port:    uint16,
	}
	s.ports[port] = p
	l.ports[p] = struct{}{}
	return p, nil
}

func (p *port) close() {
	mu.Lock()
	defer mu.Unlock()
	l, ok := listeners[port]
	if ok {
		delete(l.ports, p)
		if len(l.ports) == 0 {
			delete(listeners, p.port)
			go l.close()
		}
	}
}
