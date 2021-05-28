package reverseproxy

import "net"

type Redirect struct {
	from, to net.Addr
}

func (p *Proxy) Redirect(from, to net.Addr) (*Redirect, error) {
	return Redirect{from, to}, nil
}
