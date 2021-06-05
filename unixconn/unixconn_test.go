package unixconn

import (
	"net"
	"os"
	"syscall"
	"testing"
)

func TestUnixConn(t *testing.T) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Errorf("unexpected error creating socket pair: %s", err)
		return
	}
	fconn, _ := net.FileConn(os.NewFile(uintptr(fds[0]), ""))
	go testServerLoop(fconn.(*net.UnixConn))
	fconn, _ = net.FileConn(os.NewFile(uintptr(fds[1]), ""))
	uc = fconn.(*net.UnixConn)
	fallback = false
	newSocket = make(chan error)
	sockets = make(map[uint16]chan net.Conn)
	go runListenLoop()
}

func testServerLoop(conn *net.UnixConn) {
}
