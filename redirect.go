package reverseproxy

import (
	"errors"
	"io"
	"net"
	"sync"
)

type Redirect struct {
	serviceName matchServiceName

	mu          sync.RWMutex
	socket2addr map[*socket]net.Addr
	addr2socket map[net.Addr]*socket
}

func NewRedirecter(serviceName matchServiceName) *Redirect {
	return &Redirect{
		serviceName: serviceName,
		socket2addr: make(map[*socket]net.Addr),
		addr2socket: make(map[net.Addr]*socket),
	}
}

type addrService struct {
	matchServiceName
	net.Addr
}

func (a *addrService) Transfer(c *conn) {
	p, err := net.Dial(a.Network(), a.String())
	if err != nil {
		c.conn.Close()
		return
	}
	if _, err := p.Write(c.buffer); err != nil {
		c.conn.Close()
		return
	}
	io.Copy(p, c.conn)
}

func (r *Redirect) Add(from uint16, to net.Addr) error {
	r.mu.RLock()
	if _, ok := r.addr2socket[to]; ok {
		return ErrAddressInUse
	}
	r.mu.RUnlock()
	s, err := addService(from, &addrService{
		matchServiceName: r.serviceName,
		Addr:             to,
	})
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
