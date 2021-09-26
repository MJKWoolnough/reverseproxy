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
	"sync/atomic"
	"syscall"
	"time"
)

type buffer [http.DefaultMaxHeaderBytes]byte

type ns struct {
	c   chan net.Conn
	err error
}

var (
	fallback         = uint32(1)
	ucMu             sync.Mutex
	uc               *net.UnixConn
	listeningSockets map[uint16]struct{}
	newSocket        chan ns
	bufPool          = sync.Pool{
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
			fallback = 0
			newSocket = make(chan ns)
			listeningSockets = make(map[uint16]struct{})
			go runListenLoop()
		}
	}
}

func runListenLoop() {
	buf := bufPool.Get().(*buffer)
	oob := make([]byte, syscall.CmsgLen(4))
	sockets := make(map[uint16]chan net.Conn)
	for {
		n, oobn, _, _, err := uc.ReadMsgUnix(buf[:], oob[:])
		if err != nil {
			for _, c := range sockets {
				close(c)
			}
			atomic.StoreUint32(&fallback, 1)
			break
		}
		if oobn == 0 {
			if n == 2 {
				port := uint16(buf[1])<<8 | uint16(buf[0])
				if s, ok := sockets[port]; ok {
					close(s)
					delete(sockets, port)
					delete(listeningSockets, port)
				} else {
					listeningSockets[port] = struct{}{}
					c := make(chan net.Conn)
					sockets[port] = c
					newSocket <- ns{c: c}
				}
			} else if n > 2 {
				newSocket <- ns{err: errors.New(string(buf[2:n]))}
			}
		} else if msg, err := syscall.ParseSocketControlMessage(oob[:oobn]); err == nil && len(msg) == 1 {
			if fd, err := syscall.ParseUnixRights(&msg[0]); err == nil && len(fd) == 1 {
				nf := os.NewFile(uintptr(fd[0]), "")
				if cn, err := net.FileConn(nf); err == nil {
					var port uint16
					if tcpaddr, ok := cn.LocalAddr().(*net.TCPAddr); ok {
						port = uint16(tcpaddr.Port)
					} else {
						port = getPort(cn.LocalAddr().String())
					}
					c, ok := sockets[port]
					if ok {
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
						continue
					} else {
						cn.Close()
					}
				}
				nf.Close()
			}
		}
		for n := range buf[:n] {
			buf[n] = 0
		}
	}
}

func sendConn(c chan net.Conn, conn *conn) {
	t := time.NewTimer(time.Minute * 3)
	select {
	case <-t.C:
		conn.Close()
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
			c.clearBuffer()
		}
		return n, nil
	}
	return c.Conn.Read(b)
}

func (c *conn) clearBuffer() {
	for n := range c.buf[:c.length] {
		c.buf[n] = 0
	}
	bufPool.Put(c.buf)
	c.buf = nil
}

func (c *conn) Close() error {
	if c.buf != nil {
		c.clearBuffer()
	}
	runtime.SetFinalizer(c, nil)
	return c.Conn.Close()
}

type listener struct {
	socket uint16
	c      chan net.Conn
	addr
}

func (l *listener) Accept() (net.Conn, error) {
	c, ok := <-l.c
	if !ok {
		return nil, net.ErrClosed
	}
	return c, nil
}

func (l *listener) Close() error {
	if l.socket == 0 {
		return net.ErrClosed
	}
	var buf [2]byte
	buf[0] = byte(l.socket)
	buf[1] = byte(l.socket >> 8)
	l.socket = 0
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
	if atomic.LoadUint32(&fallback) == 1 {
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
	if _, ok := listeningSockets[port]; ok {
		ucMu.Unlock()
		return nil, ErrAlreadyListening
	}
	_, _, err := uc.WriteMsgUnix(buf[:], nil, nil)
	if err != nil {
		ucMu.Unlock()
		return nil, err
	}
	ns := <-newSocket
	ucMu.Unlock()
	if ns.err != nil {
		return nil, ns.err
	}
	l := &listener{
		socket: port,
		c:      ns.c,
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
	ErrInvalidAddress   = errors.New("port must be 0 < port < 2^16")
	ErrAlreadyListening = errors.New("port already being listened on")
)
