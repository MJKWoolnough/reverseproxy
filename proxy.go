package reverseproxy

import (
	"io"
	"net"
	"sync"

	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

type Service interface {
	Handle(net.Conn, *buffer.Buffer, int)
	Stop()
}

type Proxy struct {
	l net.Listener
	p Protocol

	mu       sync.RWMutex
	services map[string]Service
}

func NewProxy(l net.Listener, p Protocol) *Proxy {
	return &Proxy{
		l:        l,
		p:        p,
		services: make(map[string]Service),
	}
}

type Protocol interface {
	GetServerName(io.Reader, []byte) (int, []byte, error)
	Name() string
}

func (p *Proxy) Start() error {
	go p.run()
	return nil
}

func (p *Proxy) run() {
	for {
		c, err := p.l.Accept()
		if err != nil {
			if eerr, ok := err.(net.Error); ok {
				if eerr.Temporary() {
					continue
				}
			}
			break // set error?
		}
		go p.handle(c)
	}
}

func (p *Proxy) Add(serverName string, server Service) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.services[serverName]; ok {
		return ErrServerRegistered
	}
	p.services[serverName] = server
	return nil
}

func (p *Proxy) handle(c net.Conn) {
	buf := buffer.Get()
	n, name, err := p.p.GetServerName(c, buf[:])
	if err != nil {
		buffer.Put(buf)
		c.Close()
		return
	}
	p.mu.RLock()
	serv, ok := p.services[string(name)]
	p.mu.RUnlock()
	if !ok {
		buffer.Put(buf)
		c.Close()
		return
	}
	serv.Handle(c, buf, n)
}

func (p *Proxy) Stop() error {
	err := p.l.Close()
	p.mu.Lock()
	for n, s := range p.services {
		s.Stop()
		delete(p.services, n)
	}
	p.mu.Unlock()
	return err
}

const (
	ErrServerRegistered errors.Error = "server already registered"
)
