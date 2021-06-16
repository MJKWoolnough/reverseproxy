package reverseproxy

import (
	"net"
	"os"
	"syscall"
	"testing"
)

func TestUnix(t *testing.T) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	nf := os.NewFile(uintptr(fds[0]), "")
	fconn, err := net.FileConn(nf)
	if err := nf.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	u := &UnixCmd{
		conn: fconn.(*net.UnixConn),
		open: make(map[uint16]*Port),
	}
	go u.runCmdLoop(testServiceA{make(testService)})
	nf = os.NewFile(uintptr(fds[1]), "")
	fconn, err = net.FileConn(nf)
	if err := nf.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	conn := fconn.(*net.UnixConn)
	pa := getUnusedPort()
	var (
		buf [1024]byte
		oob = make([]byte, syscall.CmsgLen(4))
	)
	buf[0] = uint8(pa)
	buf[1] = uint8(pa >> 8)
	n, _, err := conn.WriteMsgUnix(buf[:2], nil, nil)
	if err != nil {
		t.Errorf("test 1: unexpected error: %s", err)
		return
	} else if n != 2 {
		t.Errorf("test 1: expecting to write 2 bytes, wrote %d", n)
		return
	}
	n, _, _, _, err = conn.ReadMsgUnix(buf[:], oob)
	if err != nil {
		t.Errorf("test 2: unexpected error: %s", err)
		return
	} else if n != 2 {
		t.Errorf("test 2: expecting to read 2 bytes, read %d", n)
	} else if pr := uint16(buf[0]) | (uint16(buf[1]) << 8); pr != pa {
		t.Errorf("test 2: expecting to read port %d, got %d", pa, pr)
		return
	}
}
