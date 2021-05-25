package unixconn

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"time"

	"vimagination.zapto.org/byteio"
	"vimagination.zapto.org/memio"
)

var (
	fallback  = true
	newSocket chan uint16
	sockets   map[uint16]chan net.Conn
)

func init() {
	c, err := net.FileConn(os.NewFile(3, ""))
	if err == nil {
		u, ok := c.(*net.UnixConn)
		if ok {
			fallback = false
			newSocket = make(chan []byte)
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
					slr.Reader = memio.Buffer(buf[:n])
					socketID := <-slr.ReadUint16()
					if c, ok := sockets[socketID]; ok {
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
	if len(c.buff) > 0 {
		c.buf = c.buf[copy(b, c.buf):]
		if n > 0 {
			return n, nil
		}
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

func requestListener(network, address string, isTLS bool) (net.Listener, error) {
	if fallback {
		return net.Listen(network, address)
	}
	buf := make(memio.Buffer, 0, len(network)+len(address)+5)
	w := byteio.StickyLittleEndianWriter{Writer: &buf}
	w.WriteString16(network)
	w.WriteString16(address)
	w.WriteBool(isTLS)
	var (
		errNum [4]byte
		oob    [4]byte
	)
	ucMu.Lock()
	uc.WriteMsgUnix(buf, nil, nil)
	socketID := <-newSocket
	ucMu.Unlock()
	if err != nil {
		return nil, err
	}

	return nil, nil
}
