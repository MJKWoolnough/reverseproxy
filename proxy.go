package reverseproxy // import "vimagination.zapto.org/reverseproxy"

import (
	"fmt"
	"net"
	"sync"
)

type Listener struct {
	net.Listener

	mu      sync.RWMutex
	sockets []*socket
}

type Proxy struct {
	Listeners map[uint16]*Listener
}

func (p *Proxy) addService(s service, port uint16) (*socket, error) {
	l, ok := p.Listeners[port]
	if !ok {
		nl, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return nil, err
		}
		l = &Listener{
			Listener: nl,
		}
	}
	socket := &socket{
		listener: l,
		service:  service,
	}
	l.mu.Lock()
	l.sockets = append(l.sockets, socket)
	l.mu.Unlock()
	return socket, nil
}

type socket struct {
	listener *listener
	service  service
}

type conn struct{}
