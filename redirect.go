package reverseproxy

import (
	"io"
	"net"
)

type addrService struct {
	MatchServiceName
	net.Addr
}

func (a *addrService) Transfer(buf []byte, conn net.Conn) {
	p, err := net.Dial(a.Network(), a.String())
	if err != nil {
		conn.Close()
		return
	}
	if _, err := p.Write(buf); err != nil {
		conn.Close()
		return
	}
	io.Copy(p, conn)
}

func AddRedirect(serviceName MatchServiceName, port uint16, to net.Addr) (*Port, error) {
	return addPort(port, &addrService{
		MatchServiceName: serviceName,
		Addr:             to,
	})
}
