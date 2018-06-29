package clienttcp

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"vimagination.zapto.org/byteio"
	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy/internal/addr"
)

func GetPort(srvname string) (uint16, error) {
	sp, ok := os.LookupEnv("rproxy_" + srvname)
	if !ok {
		return 0, ErrNoServerPort
	}
	port, err := strconv.ParseUint(sp, 10, 16)
	if err != nil {
		return 0, errors.WithContext(fmt.Sprintf("error getting port from env (%q): ", sp), err)
	}
	return uint16(port), nil
}

func ProxyListener(srvname string) (net.Listener, error) {
	port, err := GetPort(srvname)
	if err != nil {
		return nil, err
	}
	return NewListener("", port)
}

type listener struct {
	*net.TCPListener
}

func NewListener(host string, port uint16) (net.Listener, error) {
	a, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, errors.WithContext("error resolving listening address: ", err)
	}
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return nil, errors.WithContext("error starting TCP listener: ", err)
	}
	return listener{l}, nil
}

func (l listener) Accept() (net.Conn, error) {
	c, err := l.AcceptTCP()
	if err != nil {
		return nil, err
	}
	b := byteio.LittleEndianReader{Reader: c}
	a, _, err := b.ReadString8()
	if err != nil {
		return nil, errors.WithContext("error reading remote address: ", err)
	}
	return &conn{
		TCPConn: c,
		addr: addr.Addr{
			Net:  "tcp",
			Addr: a,
		},
	}, nil
}

type conn struct {
	*net.TCPConn
	addr addr.Addr
}

func (c *conn) RemoteAddr() net.Addr {
	return &c.addr
}

// Errors
const (
	ErrNoServerPort errors.Error = "no server port"
)
