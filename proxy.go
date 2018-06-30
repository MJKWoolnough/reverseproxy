package reverseproxy

import (
	"io"
	"net"
	"sync"

	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
)

type service interface {
	Handle(io.Reader, []buf) bool
}

type Proxy struct {
	l net.Listener
	p Protocol

	mu       sync.RWMutex
	services map[string]service
}

func NewProxy(l net.Listener, p Protocol) *Proxy {
	return &Proxy{
		l:        l,
		p:        p,
		services: make(map[string]service),
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
	if serv.handle(c, buf[:n]) {
		buffer.Put(buf)
		c.Close()
	}
}

func (p *Proxy) Stop() error {
	return p.l.Close()
}

const (
	ErrServerRegistered errors.Error = "server already registered"
)
