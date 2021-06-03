package reverseproxy

import (
	"io"
	"net"
)

type addrService struct {
	MatchServiceName
	net.Addr
}

func (a *addrService) Transfer(buf []byte, conn net.Conn) error {
	p, err := net.Dial(a.Network(), a.String())
	if err == nil {
		if _, err = p.Write(buf); err == nil {
			_, err = io.Copy(p, conn)
		}
	}
	return err
}

// AddRedirect sets a port to be redirected to an external service
func AddRedirect(serviceName MatchServiceName, port uint16, to net.Addr) (*Port, error) {
	return addPort(port, &addrService{
		MatchServiceName: serviceName,
		Addr:             to,
	})
}
