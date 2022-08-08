// Package reverseproxy implements a basic HTTP/TLS connection forwarder based either the passed Host header or SNI extension
package reverseproxy // import "vimagination.zapto.org/reverseproxy"

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
)

var (
	lMu       sync.RWMutex
	listeners = make(map[uint16]*listener)
)

type listener struct {
	*net.TCPListener

	mu    sync.RWMutex
	ports map[*Port]struct{}
}

var (
	httpPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, http.DefaultMaxHeaderBytes)
			return &b
		},
	}
	tlsPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, maxTLSRead)
			return &b
		},
	}
)

func (l *listener) listen() {
	for {
		c, err := l.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			l.Close()
			l.mu.Lock()
			for p := range l.ports {
				p.closed = true
				delete(l.ports, p)
			}
			l.mu.Unlock()
			return
		}
		go l.transfer(c)
	}
}

func (l *listener) transfer(c *net.TCPConn) {
	var tlsByte [1]byte
	if n, err := io.ReadFull(c, tlsByte[:]); n == 1 && err == nil {
		var (
			name           string
			pool           *sync.Pool
			readServerName func(io.Reader, []byte) (string, []byte, error)
		)
		if tlsByte[0] == 22 {
			pool = &tlsPool
			readServerName = readTLSServerName
		} else {
			pool = &httpPool
			readServerName = readHTTPServerName
		}
		b := pool.Get().(*[]byte)
		buf := *b
		buf[0] = tlsByte[0]
		name, buf, err = readServerName(c, buf)
		if err == nil {
			if host, _, err := net.SplitHostPort(name); err == nil {
				name = host
			}
			var port *Port
			l.mu.RLock()
			for p := range l.ports {
				if p.MatchService(name) {
					port = p
					break
				}
			}
			l.mu.RUnlock()
			if port != nil {
				port.Transfer(buf, c)
			}
		} else {
			c.Close()
		}
		for n := range buf {
			buf[n] = 0
		}
		pool.Put(b)
	} else {
		c.Close()
	}
}

type service interface {
	MatchServiceName
	Transfer([]byte, *net.TCPConn)
	Active() bool
}

// Port represents a service waiting on a port
type Port struct {
	service
	port   uint16
	closed bool
}

func addPort(port uint16, service service) (*Port, error) {
	if port == 0 {
		return nil, ErrInvalidPort
	}
	lMu.Lock()
	l, ok := listeners[port]
	if !ok {
		nl, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(port)})
		if err != nil {
			return nil, err
		}
		l = &listener{
			TCPListener: nl,
			ports:       make(map[*Port]struct{}),
		}
		go l.listen()
		listeners[port] = l
	}
	lMu.Unlock()
	p := &Port{
		service: service,
		port:    port,
	}
	l.mu.Lock()
	l.ports[p] = struct{}{}
	l.mu.Unlock()
	return p, nil
}

// Close closes this port connection
func (p *Port) Close() error {
	lMu.Lock()
	if !p.closed {
		l, ok := listeners[p.port]
		if ok {
			l.mu.Lock()
			delete(l.ports, p)
			if len(l.ports) == 0 {
				delete(listeners, p.port)
				l.Close()
			}
			l.mu.Unlock()
		}
		p.closed = true
	}
	lMu.Unlock()
	return nil
}

// Closed returns whether the port has been closed or not
func (p *Port) Closed() bool {
	return p.closed
}

// Status constains the status of a Port
type Status struct {
	Ports           []uint16
	Closing, Active bool
}

// Status retrieves the status of a Port
func (p *Port) Status() Status {
	lMu.RLock()
	closed := p.closed
	lMu.RUnlock()
	return Status{
		Ports:   []uint16{p.port},
		Closing: closed,
		Active:  p.service.Active(),
	}
}

// Errors
var (
	ErrInvalidPort = errors.New("cannot register on port 0")
)
