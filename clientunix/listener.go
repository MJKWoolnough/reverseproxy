package clientunix

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"vimagination.zapto.org/errors"
	"vimagination.zapto.org/reverseproxy/internal/buffer"
	"vimagination.zapto.org/reverseproxy/internal/conn"
)

type listener struct {
	unix *net.UnixConn

	mu     sync.Mutex
	length buffer.BufferLength
	oob    []byte

	conns sync.WaitGroup
}

func ProxyListener(srvname string) (net.Listener, error) {
	sfd, ok := os.LookupEnv("rproxy_" + srvname)
	if !ok {
		return nil, ErrNoServerFD
	}
	fd, err := strconv.ParseUint(sfd, 10, 64)
	if err != nil {
		return nil, errors.WithContext(fmt.Sprintf("error getting fd from env (%q): ", sfd), err)
	}
	return NewListener(uintptr(fd))
}

func NewListener(socketFD uintptr) (net.Listener, error) {
	c, err := net.FileConn(os.NewFile(socketFD, ""))
	if err != nil {
		return nil, errors.WithContext("error creating file from descriptor: ", err)
	}
	u, ok := c.(*net.UnixConn)
	if !ok {
		return nil, ErrInvalidFD
	}

	l := &listener{
		unix: u,
		oob:  make([]byte, syscall.CmsgSpace(4)),
	}
	l.length.WriteUint(uint(socketFD))
	u.SetWriteDeadline(time.Now().Add(time.Second * 5))
	_, err = u.Write(l.length[:])
	if err != nil {
		return nil, err
	}
	u.SetWriteDeadline(time.Time{})
	return l, nil
}

func (l *listener) Accept() (net.Conn, error) {
	l.mu.Lock()
	_, _, _, _, err := l.unix.ReadMsgUnix(l.length[:], l.oob)
	if err != nil {
		l.mu.Unlock()
		return nil, errors.WithContext("error reading length and socket fd: ", err)
	}
	length := l.length.ReadUint()
	buf := buffer.Get()
	_, err = io.ReadFull(l.unix, buf[:length])
	if err != nil {
		buffer.Put(buf)
		l.mu.Unlock()
		return nil, errors.WithContext("error reading buffered data: ", err)
	}
	msg, err := syscall.ParseSocketControlMessage(l.oob)
	l.mu.Unlock()
	if err != nil || len(msg) != 1 {
		buffer.Put(buf)
		return nil, errors.WithContext("error parsing socket control message: ", err)
	}
	fd, err := syscall.ParseUnixRights(&msg[0])
	if err != nil || len(fd) != 1 {
		buffer.Put(buf)
		return nil, errors.WithContext("error parsing rights for socket descriptor: ", err)
	}
	cn, err := net.FileConn(os.NewFile(uintptr(fd[0]), ""))
	if err != nil {
		buffer.Put(buf)
		return nil, errors.WithContext("error creating connection from descriptor: ", err)
	}
	if ka, ok := cn.(keepAlive); ok {
		ka.SetKeepAlive(true)
		ka.SetKeepAlivePeriod(3 * time.Minute)
	}
	l.conns.Add(1)
	mc := conn.New(cn, buf, int(length))
	runtime.SetFinalizer(mc, (*conn.Conn).Close)
	return mc, nil
}

type keepAlive interface {
	SetKeepAlive(bool) error
	SetKeepAlivePeriod(time.Duration) error
}

func (l *listener) Addr() net.Addr {
	return l.unix.LocalAddr()
}

func (l *listener) Close() error {
	l.conns.Done()
	return l.unix.Close()
}

func (l *listener) Wait() {
	l.conns.Wait()
}

// Errors
const (
	ErrNoServerFD errors.Error = "service descriptor not found"
	ErrInvalidFD  errors.Error = "invalid file descriptor"
)
