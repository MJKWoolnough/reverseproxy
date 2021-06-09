package reverseproxy // import "vimagination.zapto.org/reverseproxy"

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
)

var (
	mu        sync.RWMutex // global lock
	listeners = make(map[uint16]*listener)
)

type listener struct {
	*net.TCPListener
	ports map[*Port]struct{}
}

var (
	httpPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, http.DefaultMaxHeaderBytes)
		},
	}
	tlsPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, maxTLSRead)
		},
	}
)

func (l *listener) listen() {
	for {
		c, err := l.AcceptTCP()
		if errors.Is(err, net.ErrClosed) {
			return
		} else if err != nil {
			if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
				l.Close()
				mu.Lock()
				for p := range l.ports {
					p.closed = true
					delete(l.ports, p)
				}
				mu.Unlock()
				return
			}
			continue
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
		buf := pool.Get().([]byte)[:1]
		buf[0] = tlsByte[0]
		name, buf, err = readServerName(c, buf)
		if err == nil {
			var port *Port
			mu.RLock()
			for p := range l.ports {
				if p.matchService(name) {
					port = p
					break
				}
			}
			mu.RUnlock()
			port.Transfer(buf, c)
			pool.Put(buf)
		}
	}
	c.Close()
}

type service interface {
	matchService(string) bool
	Transfer([]byte, *net.TCPConn) error
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
	mu.Lock()
	l, ok := listeners[port]
	if !ok {
		nl, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(port)})
		if err != nil {
			mu.Unlock()
			return nil, err
		}
		l = &listener{
			TCPListener: nl,
			ports:       make(map[*Port]struct{}),
		}
		go l.listen()
	}
	p := &Port{
		service: service,
		port:    port,
	}
	l.ports[p] = struct{}{}
	mu.Unlock()
	return p, nil
}

// Close closes this port connection
func (p *Port) Close() error {
	mu.Lock()
	if !p.closed {
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
	mu.Unlock()
	return nil
}

// Closed returns whether the port has been closed or not
func (p *Port) Closed() bool {
	return p.closed
}

// Errors
var (
	ErrInvalidPort = errors.New("cannot register on port 0")
)
