package reverseproxy

import (
	"io"
	"net"
)

type addrService struct {
	MatchServiceName
	net.Addr
}

func (a *addrService) Transfer(buf []byte, conn *net.TCPConn) error {
	p, err := net.Dial(a.Network(), a.String())
	if err == nil {
		if _, err = p.Write(buf); err == nil {
			go func() {
				c := make(chan struct{})
				go copyConn(p, conn, c)
				go copyConn(conn, p, c)
				<-c
				p.Close()
				conn.Close()
				<-c
			}()
		}
	}
	return err
}

func copyConn(a, b net.Conn, c chan struct{}) {
	io.Copy(a, b)
	c <- struct{}{}
}

// AddRedirect sets a port to be redirected to an external service
func AddRedirect(serviceName MatchServiceName, port uint16, to net.Addr) (*Port, error) {
	return addPort(port, &addrService{
		MatchServiceName: serviceName,
		Addr:             to,
	})
}
