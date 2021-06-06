package unixconn

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
)

var (
	lone, ltwo *net.TCPListener
	pone, ptwo uint16
)

func TestMain(m *testing.T) {
	addr := new(net.TCPAddr)
	var err error
	lone, err = net.ListenTCP("tcp", addr)
	if err != nil {
		m.Fatalf("unexpected error during setup: %q", err)
	}
	ltwo, err = net.ListenTCP("tcp", addr)
	if err != nil {
		m.Fatalf("unexpected error during setup: %q", err)
	}
	if pone = getPort(lone.Addr().String()); pone == 0 {
		m.Fatalf("invalid port number: %d", pone)
	}
	if ptwo = getPort(ltwo.Addr().String()); ptwo == 0 {
		m.Fatalf("invalid port number: %d", ptwo)
	}
}

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
	if err == nil {
		t.Errorf("test 1: expecting \"error\", got: nil")
		return
	} else if err.Error() != "error" {
		t.Errorf("test 1: expecting \"error\", got: %q", err)
		return
	} else if l != nil {
		t.Error("test 1: expecting nil listener")
		return
	}
	l, err = Listen("tcp", "80")
	if err == nil {
		t.Errorf("test 2: expecting \"error\", got: nil")
		return
	} else if !errors.Is(err, ErrInvalidAddress) {
		t.Errorf("test 2: expecting ErrInvalidAddress, got: %q", err)
		return
	} else if l != nil {
		t.Error("test 2: expecting nil listener")
		return
	}
	pstr := fmt.Sprintf(":%d", pone)
	l, err = Listen("tcp", pstr)
	if err != nil {
		t.Errorf("test 3: unexpected error: %s", err)
		return
	} else if l == nil {
		t.Errorf("test 3: expecting non-nil Listener")
		return
	}
	if net := l.Addr().Network(); net != "tcp" {
		t.Errorf("test 4: expecting network \"tcp\", got: %q", net)
		return
	} else if addr := l.Addr().String(); addr != pstr {
		t.Errorf("test 4: expecting address %q, got %q", pstr, addr)
		return
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
	p := uint16(buf[1])<<8 | uint16(buf[0])
	if p != pone {
		conn.WriteMsgUnix(buf[:5], nil, nil)
		return
	}
	conn.WriteMsgUnix(buf[:2], nil, nil)
	conn.Close()
}
