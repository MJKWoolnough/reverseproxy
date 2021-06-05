package unixconn

import (
	"errors"
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
	defer uc.Close()
	fallback = false
	newSocket = make(chan error)
	sockets = make(map[uint16]chan net.Conn)
	go runListenLoop()
	l, err := Listen("tcp", ":8080")
	if l != nil {
		t.Error("expecting nil listener")
		return
	} else if err == nil {
		t.Errorf("expecting 'error', got: nil")
		return
	} else if err.Error() != "error" {
		t.Errorf("expecting 'error', got: %x", err)
		return
	}
	l, err = Listen("tcp", "80")
	if l != nil {
		t.Error("expecting nil listener")
		return
	} else if err == nil {
		t.Errorf("expecting 'error', got: nil")
		return
	} else if !errors.Is(err, ErrInvalidAddress) {
		t.Errorf("expecting ErrInvalidAddress, got: %x", err)
		return
	}
	l, err = Listen("tcp", ":80")
	if l == nil {
		t.Errorf("expecting non-nil Listener")
		return
	} else if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

func testServerLoop(conn *net.UnixConn) {
	buf := [...]byte{0, 0, 'e', 'r', 'r', 'o', 'r'}
	conn.ReadMsgUnix(buf[:2], nil)
	if buf[0] != 0x90 || buf[1] != 0x1f {
		conn.WriteMsgUnix(buf[:5], nil, nil)
		return
	}
	conn.WriteMsgUnix(buf[:], nil, nil)
	conn.ReadMsgUnix(buf[:2], nil)
	conn.WriteMsgUnix(buf[:2], nil, nil)
	conn.Close()
}
