package reverseproxy

import (
	"io"
	"net"
)

type addrService struct {
	matchServiceName
	net.Addr
}

func (a *addrService) Transfer(c *conn) {
	p, err := net.Dial(a.Network(), a.String())
	if err != nil {
		c.conn.Close()
		return
	}
	if _, err := p.Write(c.buffer); err != nil {
		c.conn.Close()
		return
	}
	io.Copy(p, c.conn)
}

func AddRedirect(serviceName matchServiceName, port uint16, to net.Addr) (*Port, error) {
	return addPort(port, &addrService{
		matchServiceName: serviceName,
		Addr:             to,
	})
}
