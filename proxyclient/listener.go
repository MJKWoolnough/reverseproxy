package proxyclient

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"vimagination.zapto.org/errors"
)

type Listener struct {
	unix *net.UnixConn

	mu     sync.Mutex
	length [4]byte
	oob    []byte

	conns sync.WaitGroup
}

func ProxyListener(srvname string) (*Listener, error) {
	sfd, ok := os.LookupEnv("rproxy_" + srvname)
	if !ok {
		return nil, ErrNoServerFD
	}
	fd, err := strconv.ParseUint(sfd, 10, 64)
	if err != nil {
		return nil, errors.WithContext(fmt.Sprintf("error getting fd from enc (%q): ", sfd), err)
	}
	return NewListener(uintptr(fd))
}

func NewListener(socketFD uintptr) (*Listener, error) {
	c, err := net.FileConn(os.NewFile(socketFD, ""))
	if err != nil {
		return nil, errors.WithContext("error creating file from descriptor: ", err)
	}
	u, ok := c.(*net.UnixConn)
	if !ok {
		return nil, ErrInvalidFD
	}

	return &Listener{
		unix: u,
		oob:  make([]byte, syscall.CmsgSpace(4)),
	}, nil
}

func (l *Listener) Accept() (net.Conn, error) {
	l.mu.Lock()
	_, _, _, _, err := l.unix.ReadMsgUnix(l.length[:], l.oob)
	if err != nil {
		l.mu.Unlock()
		return nil, errors.WithContext("error reading length and socket fd: ", err)
	}
	length := uint(l.length[0]) | uint(l.length[1])<<8 | uint(l.length[2])<<16 | uint(l.length[3])<<24
	c := new(Conn)
	c.buffer.Init()
	c.buffer.LimitedBuffer = c.buffer.LimitedBuffer[:0:length]
	_, err = c.buffer.ReadFrom(&c.buffer)
	if err != nil {
		c.buffer.Close()
		l.mu.Unlock()
		return nil, errors.WithContext("error reading buffered data: ", err)
	}
	msg, err := syscall.ParseSocketControlMessage(l.oob)
	l.mu.Unlock()
	if err != nil || len(msg) != 1 {
		c.buffer.Close()
		return nil, errors.WithContext("error parsing socket control message: ", err)
	}
	fd, err := syscall.ParseUnixRights(&msg[0])
	if err != nil || len(fd) != 1 {
		c.buffer.Close()
		return nil, errors.WithContext("error parsing rights for socket descriptor: ", err)
	}
	f := os.NewFile(uintptr(fd[0]), "")
	if f == nil {
		c.buffer.Close()
		return nil, ErrInvalidFD
	}
	c.Conn, err = net.FileConn(f)
	if err != nil {
		c.buffer.Close()
		return nil, errors.WithContext("error creating connection from descriptor: ", err)
	}
	if ka, ok := c.Conn.(keepAlive); ok {
		ka.SetKeepAlive(true)
		ka.SetKeepAlivePeriod(3 * time.Minute)
	}
	l.conns.Add(1)
	runtime.SetFinalizer(c, (*Conn).Close)
	return c, nil
}

type keepAlive interface {
	SetKeepAlive(bool) error
	SetKeepAlivePeriod(time.Duration) error
}

func (l *Listener) Addr() net.Addr {
	return l.unix.LocalAddr()
}

func (l *Listener) Close() error {
	l.conns.Done()
	return l.unix.Close()
}

func (l *Listener) Wait() {
	l.conns.Wait()
}

// Errors
const (
	ErrNoServerFD errors.Error = "service descriptor not found"
	ErrInvalidFD  errors.Error = "invalid file descriptor"
)
