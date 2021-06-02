package reverseproxy // import "vimagination.zapto.org/reverseproxy"

import (
	"errors"
	"fmt"
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
	net.Listener
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
		c, err := l.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		} else if err != nil {
			continue
		}
		var tlsByte [1]byte
		if n, err := io.ReadFull(c, tlsByte[:]); n != 1 || err != nil {
			c.Close()
		}
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
		name, buf, err = readServerName(c, buf)
		if err != nil {
			c.Close()
			continue
		}
		var port *Port
		mu.RLock()
		for p := range l.ports {
			if p.MatchService(name) {
				port = p
				break
			}
		}
		mu.RUnlock()
		go transfer(port, buf, c, pool)
	}
}

func transfer(port *Port, buf []byte, c net.Conn, pool *sync.Pool) {
	port.Transfer(buf, c)
	pool.Put(buf)
}

type service interface {
	MatchService(string) bool
	Transfer([]byte, net.Conn)
}

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
	defer mu.Unlock()
	l, ok := listeners[port]
	if !ok {
		nl, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return nil, err
		}
		l = &listener{
			Listener: nl,
			ports:    make(map[*Port]struct{}),
		}
		go l.listen()
	}
	p := &Port{
		service: service,
		port:    port,
	}
	l.ports[p] = struct{}{}
	return p, nil
}

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

func (p *Port) Closed() bool {
	return p.closed
}

// Errors
var (
	ErrInvalidPort = errors.New("cannot register on port 0")
)
