package reverseproxy

import (
	"io"
	"net"
	"sync/atomic"
)

type addrService struct {
	MatchServiceName
	net.Addr
	copying uint64
}

func (a *addrService) Transfer(buf []byte, conn *net.TCPConn) error {
	p, err := net.Dial(a.Network(), a.String())
	if err == nil {
		if _, err = p.Write(buf); err == nil {
			atomic.AddUint64(&a.copying, 2)
			go copyConn(p, conn, &a.copying)
			go copyConn(conn, p, &a.copying)
		}
	}
	return err
}

func (a *addrService) Active() bool {
	return atomic.LoadUint64(*a.copying) > 0
}

func copyConn(a, b net.Conn, c *uint64) {
	io.Copy(a, b)
	a.Close()
	b.Close()
	atomic.AddUint64(c, ^uint64(0))
}

// AddRedirect sets a port to be redirected to an external service
func AddRedirect(serviceName MatchServiceName, port uint16, to net.Addr) (*Port, error) {
	return addPort(port, &addrService{
		MatchServiceName: serviceName,
		Addr:             to,
	})
}
