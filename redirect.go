package reverseproxy

import (
	"errors"
	"net"
	"sync"
)

type Redirect struct {
	serviceName matchServiceName

	mu          sync.RWMutex
	socket2addr map[*socket]net.Addr
	addr2socket map[net.Addr]*socket
}

func (r *Redirect) AddRedirect(from uint16, to net.Addr) error {
	r.mu.RLock()
	if _, ok := r.addr2socket[to]; ok {
		return ErrAddressInUse
	}
	r.mu.RUnlock()
	s, err := addService(r.serviceName, from)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.socket2addr[s] = to
	r.addr2socket[to] = s
	r.mu.Unlock()
	return nil
}

func (p *Proxy) RegisterRedirecter(serviceName matchServiceName) (*Redirect, error) {
	return &Redirect{service: serviceName}, nil
}

// Errors
var (
	ErrAddressInUse = errors.New("address in use")
)
