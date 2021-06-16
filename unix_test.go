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
	_ = conn
}
