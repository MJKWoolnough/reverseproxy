package reverseproxy

import (
	"io"
	"net"
	"sync"

	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

type Service interface {
	Handle(net.Conn, *buffer.Buffer, uint)
	Stop()
}

type Proxy struct {
	l net.Listener
	p Protocol

	mu       sync.RWMutex
	started  bool
	services map[string]Service
	err      error
}

func NewProxy(l net.Listener, p Protocol) *Proxy {
	return &Proxy{
		l:        l,
		p:        p,
		services: make(map[string]Service),
	}
}

type Protocol interface {
	GetServerName(io.Reader, []byte) (uint, []byte, error)
	Name() string
}

func (p *Proxy) Start() {
	go func() {
		err := p.Run()
		if err != nil && err != ErrRunning {
			p.mu.Lock()
			p.err = err
			p.mu.Unlock()
		}
	}()
}

func (p *Proxy) Name() string {
	return p.p.Name()
}

func (p *Proxy) Run() error {
	p.mu.Lock()
	s := p.started
	p.started = true
	p.mu.Unlock()
	if s {
		return ErrRunning
	}
	for {
		c, err := p.l.Accept()
		if err != nil {
			if eerr, ok := err.(net.Error); ok {
				if eerr.Temporary() {
					continue
				}
			}
			return err
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

func (p *Proxy) Remove(serverName string) {
	p.mu.Lock()
	delete(p.services, serverName)
	p.mu.Unlock()
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
	if err == nil {
		err = p.err
	}
	p.mu.Unlock()
	return err
}

const (
	ErrRunning          errors.Error = "already running"
	ErrServerRegistered errors.Error = "server already registered"
)
