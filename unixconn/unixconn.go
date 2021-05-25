package unixconn

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"vimagination.zapto.org/byteio"
	"vimagination.zapto.org/memio"
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
					slr byteio.StickyLittleEndianReader
				)
				for {
					n, oobn, _, _, err := u.ReadMsgUnix(buf[:], oob[:])
					if err != nil {
						if nerr, ok := err.(net.Error); !ok || !nerr.Temporary() {
							break
						}
					}
					b := memio.Buffer(buf[:n])
					slr.Reader = &b
					socketID := slr.ReadUint16()
					if c, ok := sockets[socketID]; ok {
						if n == 2 {
							close(c)
							delete(sockets, socketID)
						} else {
							data := slr.ReadBytes(n - 2)
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
							c <- conn
						}
					} else if n == 2 {
						sockets[socketID] = make(chan net.Conn)
						newSocket <- socketID
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

func ListenHTTP(network, address string) (net.Listener, error) {
	return requestListener(network, address, false)
}

func ListenTLS(network, address string, config *tls.Config) (net.Listener, error) {
	if config == nil || len(config.Certificates) == 0 && config.GetCertificate == nil && config.GetConfigForClient == nil {
		return nil, errors.New("need valid tls.Config")
	}
	l, err := requestListener(network, address, true)
	if err != nil {
		return nil, err
	}
	return tls.NewListener(l, config), nil
}

type listener struct {
	socket uint16
	addr
}

func (l *listener) Accept() (net.Conn, error) {
	c, ok := <-sockets[l.socket]
	if !ok {
		return nil, errors.New("closed")
	}
	return c, nil
}

func (l *listener) Close() error {
	buf := make(memio.Buffer, 0, 2)
	w := &byteio.StickyLittleEndianWriter{
		Writer: &buf,
	}
	w.WriteUint16(l.socket)
	ucMu.Lock()
	uc.WriteMsgUnix(buf, nil, nil)
	ucMu.Unlock()
	return nil
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

func requestListener(network, address string, isTLS bool) (net.Listener, error) {
	if fallback {
		return net.Listen(network, address)
	}
	buf := make(memio.Buffer, 0, len(network)+len(address)+5)
	w := byteio.StickyLittleEndianWriter{Writer: &buf}
	w.WriteString16(network)
	w.WriteString16(address)
	w.WriteBool(isTLS)
	ucMu.Lock()
	uc.WriteMsgUnix(buf, nil, nil)
	socketID := <-newSocket
	ucMu.Unlock()
	return &listener{
		socket: socketID,
		addr: addr{
			network: network,
			address: address,
		},
	}, nil
}
