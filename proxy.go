package reverseproxy // import "vimagination.zapto.org/reverseproxy"

import "net"

type Listener struct {
	net.Listener
	sockets []socket
}

type Proxy struct {
	Listeners map[net.Addr]*Listener
}

type socket struct{}

type conn struct{}
