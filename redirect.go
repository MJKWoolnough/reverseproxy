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
			c := make(chan error)
			go copyConn(p, conn, c)
			go copyConn(conn, p, c)
			err = <-c
			p.Close()
			if err == nil {
				err = <-c
			} else {
				<-c
			}
		}
	}
	return err
}

func copyConn(a, b net.Conn, c chan error) {
	_, err := io.Copy(a, b)
	c <- err
}

// AddRedirect sets a port to be redirected to an external service
func AddRedirect(serviceName MatchServiceName, port uint16, to net.Addr) (*Port, error) {
	return addPort(port, &addrService{
		MatchServiceName: serviceName,
		Addr:             to,
	})
}
