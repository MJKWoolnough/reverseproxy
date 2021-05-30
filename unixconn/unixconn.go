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

var (
	fallback  = true
	ucMu      sync.Mutex
	uc        *net.UnixConn
	newSocket chan uint16
	sockets   map[uint16]chan net.Conn
)

func init() {
	c, err := net.FileConn(os.NewFile(3, ""))
	if err == nil {
		u, ok := c.(*net.UnixConn)
		uc = u
		if ok {
			fallback = false
			newSocket = make(chan uint16)
			sockets = make(map[uint16]chan net.Conn)
			go func() {
				var (
					buf [http.DefaultMaxHeaderBytes]byte
					oob [4]byte
				)
				for {
					n, oobn, _, _, err := u.ReadMsgUnix(buf[:], oob[:])
					if err != nil {
						if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
							break
						}
					} else if n < 2 {
						break
					}
					socketID := uint16(buf[1]<<8) | uint16(buf[0])
					if c, ok := sockets[socketID]; ok {
						if n == 2 {
							close(c)
							delete(sockets, socketID)
						} else {
							data := buf[2:]
							msg, err := syscall.ParseSocketControlMessage(oob[:oobn])
							if err != nil || len(msg) != 1 {
								continue
							}
							fd, err := syscall.ParseUnixRights(&msg[0])
							if err != nil || len(fd) != 1 {
								continue
							}
							cn, err := net.FileConn(os.NewFile(uintptr(fd[0]), ""))
							if err != nil {
								continue
							}
							if ka, ok := cn.(keepAlive); ok {
								ka.SetKeepAlive(true)
								ka.SetKeepAlivePeriod(3 * time.Minute)
							}
							conn := &conn{
								Conn: cn,
								buf:  data,
							}
							runtime.SetFinalizer(conn, (*conn).Close)
							go func() {
								t := time.NewTimer(time.Minute * 3)
								select {
								case <-t.C:
									conn.Close()
								case c <- conn:
								}
								t.Stop()
							}()
						}
					} else if n > 2 {
						newSocket <- errors.New(string(buf[2:]))
					} else {
						sockets[socketID] = make(chan net.Conn)
						newSocket <- nil
					}
				}
			}()
		}
	}
}

type keepAlive interface {
	SetKeepAlive(bool) error
	SetKeepAlivePeriod(time.Duration) error
}

type conn struct {
	net.Conn
	buf []byte
}

func (c *conn) Read(b []byte) (int, error) {
	if len(c.buf) > 0 {
		n := copy(b, c.buf)
		c.buf = c.buf[n:]
		return n, nil
	}
	return c.Conn.Read(b)
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
	var buf [2]byte
	buf[0] = byte(l.socket)
	but[1] = byte(l.socket >> 8)
	ucMu.Lock()
	_, _, err := uc.WriteMsgUnix(buf, nil, nil)
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
	_, portStr, _ := net.SplitHostPort(address)
	port, _ := strconv.ParseUint(portStr, 10, 16)
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
	err := <-newSocket
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

// Errors
var (
	ErrInvalidAddress = errors.New("port must be 0 < port < 2^16")
)
