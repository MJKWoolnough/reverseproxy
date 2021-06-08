package unixconn

import (
	"errors"
	"fmt"
	"io"
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
	if lone, err = net.ListenTCP("tcp", addr); err != nil {
		m.Fatalf("unexpected error during setup (1): %q", err)
	}
	if ltwo, err = net.ListenTCP("tcp", addr); err != nil {
		m.Fatalf("unexpected error during setup (2): %q", err)
	}
	if pone = getPort(lone.Addr().String()); pone == 0 {
		m.Fatalf("invalid port number (1): %d", pone)
	}
	if ptwo = getPort(ltwo.Addr().String()); ptwo == 0 {
		m.Fatalf("invalid port number (2): %d", ptwo)
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
	listeningSockets = make(map[uint16]struct{})
	defer uc.Close()
	fallback = false
	newSocket = make(chan ns)
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
	c, err := l.Accept()
	if err != nil {
		t.Errorf("test 5: unexpected error: %s", err)
		return
	} else if c == nil {
		t.Error("test 5: conn should not be nil")
		return
	}
	var buf [32]byte
	n, err := c.Read(buf[:])
	if err != nil {
		t.Errorf("test 6: unexpected error: %s", err)
		return
	} else if n != 3 {
		t.Errorf("test 6: expecting to read 3 bytes, read %d: ", n)
		return
	} else if string(buf[:3]) != "BIG" {
		t.Errorf("test 6: expecting to read \"BIG\", read: %q", buf[:3])
	}
	n, err = c.Read(buf[:])
	if err != nil {
		t.Errorf("test 7: unexpected error: %s", err)
		return
	} else if n != 4 {
		t.Errorf("test 7: expecting to read 4 bytes, read %d: ", n)
		return
	} else if string(buf[:4]) != "data" {
		t.Errorf("test 7: expecting to read \"data\", read: %q", buf[:3])
		return
	}
	n, err = c.Read(buf[:])
	if n != 0 {
		t.Errorf("test 8: expecting to read no data, read: %q", buf[:n])
		return
	} else if !errors.Is(err, io.EOF) {
		t.Errorf("test 8: expecting to EOF, got: %s", err)
		return
	}
	var l2 net.Listener
	pstr = fmt.Sprintf(":%d", ptwo)
	l2, err = Listen("tcp", pstr)
	if err != nil {
		t.Errorf("test 9: unexpected error: %s", err)
		return
	} else if l2 == nil {
		t.Errorf("test 9: expecting non-nil Listener")
		return
	}
	if net := l2.Addr().Network(); net != "tcp" {
		t.Errorf("test 10: expecting network \"tcp\", got: %q", net)
		return
	} else if addr := l2.Addr().String(); addr != pstr {
		t.Errorf("test 10: expecting address %q, got %q", pstr, addr)
		return
	}
	c, err = l2.Accept()
	if err != nil {
		t.Errorf("test 11: unexpected error: %s", err)
		return
	} else if c == nil {
		t.Error("test 11: conn should not be nil")
		return
	}
	err = l2.Close()
	if err != nil {
		t.Errorf("test 12: expecting nil error, got: %s", err)
	}
	err = l2.Close()
	if !errors.Is(err, net.ErrClosed) {
		t.Errorf("test 13: expecting net.ErrClosed, got: %s", err)
	}
	ct, err := l2.Accept()
	if !errors.Is(err, net.ErrClosed) {
		t.Errorf("test 14: expecting net.ErrClosed, got: %s", err)
		return
	} else if ct != nil {
		t.Errorf("test 14: expecting nil conn, got: %v", ct)
	}
	ct, err = l.Accept()
	if err != nil {
		t.Errorf("test 15: unexpected error: %s", err)
	} else if ct == nil {
		t.Error("test 15: recieved nil conn when conn expected")
	}
	n, err = c.Read(buf[:])
	if err != nil {
		t.Errorf("test 16: unexpected error: %s", err)
		return
	} else if n != 5 {
		t.Errorf("test 16: expecting to read 3 bytes, read %d: ", n)
		return
	} else if string(buf[:5]) != "HELLO" {
		t.Errorf("test 16: expecting to read \"HELLO\", read: %q", buf[:5])
	}
	n, err = ct.Read(buf[:])
	if err != nil {
		t.Errorf("test 17: unexpected error: %s", err)
		return
	} else if n != 10 {
		t.Errorf("test 17: expecting to read 10 bytes, read %d: ", n)
		return
	} else if string(buf[:10]) != "1234567890" {
		t.Errorf("test 17: expecting to read \"1234567890\", read: %q", buf[:10])
	}
	n, err = c.Read(buf[:])
	if err != nil {
		t.Errorf("test 18: unexpected error: %s", err)
		return
	} else if n != 5 {
		t.Errorf("test 18: expecting to read 5 bytes, read %d: ", n)
		return
	} else if string(buf[:5]) != "world" {
		t.Errorf("test 18: expecting to read \"world\", read: %q", buf[:5])
		return
	}
	n, err = ct.Read(buf[:])
	if err != nil {
		t.Errorf("test 19: unexpected error: %s", err)
		return
	} else if n != 10 {
		t.Errorf("test 19: expecting to read 10 bytes, read %d: ", n)
		return
	} else if string(buf[:10]) != "0987654321" {
		t.Errorf("test 19: expecting to read \"0987654321\", read: %q", buf[:3])
	}
	n, err = c.Read(buf[:])
	if n != 0 {
		t.Errorf("test 20: expecting to read no data, read: %q", buf[:n])
		return
	} else if !errors.Is(err, io.EOF) {
		t.Errorf("test 20: expecting to EOF, got: %s", err)
		return
	}
	err = ct.Close()
	if err != nil {
		t.Errorf("test 21: expecting nil error, got: %s", err)
	}
	n, err = ct.Read(buf[:])
	if n != 0 {
		t.Errorf("test 22: expecting to read no data, read: %q", buf[:n])
		return
	} else if !errors.Is(err, net.ErrClosed) {
		t.Errorf("test 22: expecting to EOF, got: %s", err)
		return
	}
}

func testServerLoop(conn *net.UnixConn) {
	defer conn.Close()
	buf := [...]byte{0, 0, 'e', 'r', 'r', 'o', 'r'}

	// test 1
	conn.ReadMsgUnix(buf[:2], nil)
	if buf[0] != 0x90 || buf[1] != 0x1f {
		conn.WriteMsgUnix(buf[:5], nil, nil)
		return
	}
	conn.WriteMsgUnix(buf[:], nil, nil)

	// test 3
	conn.ReadMsgUnix(buf[:2], nil)
	p := uint16(buf[1])<<8 | uint16(buf[0])
	if p != pone {
		conn.WriteMsgUnix(buf[:5], nil, nil)
		return
	}
	conn.WriteMsgUnix(buf[:2], nil, nil)

	go func() {
		c, _ := net.DialTCP("tcp", nil, &net.TCPAddr{Port: int(pone)})
		c.Write([]byte("data")) // test 7
		c.Close()
	}()

	c, _ := lone.AcceptTCP()         // test 5
	transfer(conn, c, []byte("BIG")) // test 6

	// test 9
	conn.ReadMsgUnix(buf[:2], nil)
	p = uint16(buf[1])<<8 | uint16(buf[0])
	if p != ptwo {
		conn.WriteMsgUnix(buf[:5], nil, nil)
		return
	}
	conn.WriteMsgUnix(buf[:2], nil, nil)

	// test 11
	go func() {
		c, _ := net.DialTCP("tcp", nil, &net.TCPAddr{Port: int(ptwo)})
		c.Write([]byte("world")) // test 18
		c.Close()                // test 20
	}()
	c, _ = ltwo.AcceptTCP()
	transfer(conn, c, []byte("HELLO")) // test 16

	// test 12
	conn.ReadMsgUnix(buf[:2], nil)
	conn.WriteMsgUnix(buf[:2], nil, nil)

	// test 15
	go func() {
		c, _ := net.DialTCP("tcp", nil, &net.TCPAddr{Port: int(pone)})
		c.Write([]byte("0987654321")) // test 19
		c.Close()
	}()
	c, _ = lone.AcceptTCP()
	transfer(conn, c, []byte("1234567890")) // test 17
}

func transfer(conn *net.UnixConn, c *net.TCPConn, data []byte) {
	f, _ := c.File()
	conn.WriteMsgUnix(data, syscall.UnixRights(int(f.Fd())), nil)
}
