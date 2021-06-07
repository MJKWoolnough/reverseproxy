// Package unixconn facilitates creating reverse proxy connections
package unixconn // import "vimagination.zapto.org/reverseproxy/unixconn"

import (
	"errors"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type buffer [http.DefaultMaxHeaderBytes]byte

var (
	fallback  = true
	ucMu      sync.Mutex
	uc        *net.UnixConn
	newSocket chan error
	sockets   map[uint16]chan net.Conn
	bufPool   = sync.Pool{
		New: func() interface{} {
			return new(buffer)
		},
	}
)

func init() {
	c, err := net.FileConn(os.NewFile(3, ""))
	if err == nil {
		u, ok := c.(*net.UnixConn)
		uc = u
		if ok {
			fallback = false
			newSocket = make(chan error)
			sockets = make(map[uint16]chan net.Conn)
			go runListenLoop()
		}
	}
}

func runListenLoop() {
	buf := bufPool.Get().(*buffer)
	oob := make([]byte, syscall.CmsgLen(4))
	for {
		n, oobn, _, _, err := uc.ReadMsgUnix(buf[:], oob[:])
		if err != nil {
			if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
				break
			}
		}
		if oobn == 0 {
			if n >= 2 {
				port := uint16(buf[1])<<8 | uint16(buf[0])
				sockets[port] = make(chan net.Conn)
				if n == 2 {
					newSocket <- nil
				} else {
					newSocket <- errors.New(string(buf[2:n]))
				}
			}
		} else if msg, err := syscall.ParseSocketControlMessage(oob[:oobn]); err == nil && len(msg) == 1 {
			if fd, err := syscall.ParseUnixRights(&msg[0]); err == nil && len(fd) == 1 {
				if cn, err := net.FileConn(os.NewFile(uintptr(fd[0]), "")); err == nil {
					var port uint16
					if tcpaddr, ok := cn.LocalAddr().(*net.TCPAddr); ok {
						port = uint16(tcpaddr.Port)
					} else {
						port = getPort(cn.LocalAddr().String())
					}
					if c, ok := sockets[port]; ok {
						if ka, ok := cn.(keepAlive); ok {
							if err := ka.SetKeepAlive(true); err != nil {
								ka.SetKeepAlivePeriod(3 * time.Minute)
							}
						}
						cc := &conn{
							Conn:   cn,
							buf:    buf,
							length: n,
						}
						buf = bufPool.Get().(*buffer)
						runtime.SetFinalizer(cc, (*conn).Close)
						go sendConn(c, cc)
					} else {
						cn.Close()
					}
				}
			}
		}
	}
}

func sendConn(c chan net.Conn, conn *conn) {
	t := time.NewTimer(time.Minute * 3)
	select {
	case <-t.C:
	case c <- conn:
	}
	t.Stop()
}

type keepAlive interface {
	SetKeepAlive(bool) error
	SetKeepAlivePeriod(time.Duration) error
}

type conn struct {
	net.Conn
	buf    *buffer
	pos    int
	length int
}

func (c *conn) Read(b []byte) (int, error) {
	if c.buf != nil {
		n := copy(b, c.buf[c.pos:c.length])
		c.pos += n
		if c.pos == c.length {
			bufPool.Put(c.buf)
			c.buf = nil
		}
		return n, nil
	}
	return c.Conn.Read(b)
}

func (c *conn) Close() error {
	if c.buf != nil {
		bufPool.Put(c.buf)
		c.buf = nil
	}
	return c.Conn.Close()
}

type listener struct {
	socket uint16
	addr
}

func (l *listener) Accept() (net.Conn, error) {
	c, ok := <-sockets[l.socket]
	if !ok {
		return nil, net.ErrClosed
	}
	return c, nil
}

func (l *listener) Close() error {
	if l.socket == 0 {
		return net.ErrClosed
	}
	l.socket = 0
	var buf [2]byte
	buf[0] = byte(l.socket)
	buf[1] = byte(l.socket >> 8)
	ucMu.Lock()
	_, _, err := uc.WriteMsgUnix(buf[:], nil, nil)
	ucMu.Unlock()
	return err
}

func (l *listener) Addr() net.Addr {
	return l.addr
}

type addr struct {
	network, address string
}

func (a addr) Network() string {
	return a.network
}

func (a addr) String() string {
	return a.address
}

// Listen creates a reverse proxy connection, falling back to the net package if
// the reverse proxy is not available
func Listen(network, address string) (net.Listener, error) {
	if fallback {
		return net.Listen(network, address)
	}
	port := getPort(address)
	if port == 0 {
		return nil, ErrInvalidAddress
	}
	var buf [2]byte
	buf[0] = byte(port)
	buf[1] = byte(port >> 8)
	ucMu.Lock()
	_, _, err := uc.WriteMsgUnix(buf[:], nil, nil)
	if err != nil {
		ucMu.Unlock()
		return nil, err
	}
	err = <-newSocket
	ucMu.Unlock()
	if err != nil {
		return nil, err
	}
	l := &listener{
		socket: port,
		addr: addr{
			network: network,
			address: address,
		},
	}
	runtime.SetFinalizer(l, (*listener).Close)
	return l, nil
}

func getPort(address string) uint16 {
	_, portStr, _ := net.SplitHostPort(address)
	port, _ := strconv.ParseUint(portStr, 10, 16)
	return uint16(port)
}

// Errors
var (
	ErrInvalidAddress = errors.New("port must be 0 < port < 2^16")
)
