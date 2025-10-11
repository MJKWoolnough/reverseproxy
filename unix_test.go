package reverseproxy

import (
	"bytes"
	"net"
	"os"
	"syscall"
	"testing"
	"time"
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
	} else if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	u := &UnixCmd{
		conn: fconn.(*net.UnixConn),
		open: make(map[uint16]*Port),
	}

	go u.runCmdLoop(testServiceA{make(testService)})

	nf = os.NewFile(uintptr(fds[1]), "")

	fconn, err = net.FileConn(nf)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	} else if err = nf.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	conn := fconn.(*net.UnixConn)

	var (
		buf [1024]byte
		oob = make([]byte, syscall.CmsgLen(4))
	)

	if n, _, err := conn.WriteMsgUnix(buf[:2], nil, nil); err != nil {
		t.Errorf("test 1: unexpected error: %s", err)

		return
	} else if n != 2 {
		t.Errorf("test 1: expecting to write 2 bytes, wrote %d", n)

		return
	}

	if n, _, _, _, err := conn.ReadMsgUnix(buf[:], oob); err != nil {
		t.Errorf("test 2: unexpected error: %s", err)

		return
	} else if n <= 2 {
		t.Errorf("test 2: expecting to read more than 2 bytes, read %d", n)

		return
	} else if pr := uint16(buf[0]) | (uint16(buf[1]) << 8); pr != 0 {
		t.Errorf("test 2: expecting to read port 0, got %d", pr)

		return
	} else if string(buf[2:n]) != "cannot register on port 0" {
		t.Errorf("test 2: expecting ErrInvalidPort, got %q", buf[2:n])

		return
	}

	pa := getUnusedPort()
	buf[0] = uint8(pa)
	buf[1] = uint8(pa >> 8)

	if n, _, err := conn.WriteMsgUnix(buf[:2], nil, nil); err != nil {
		t.Errorf("test 3: unexpected error: %s", err)

		return
	} else if n != 2 {
		t.Errorf("test 3: expecting to write 2 bytes, wrote %d", n)

		return
	}

	if n, _, _, _, err := conn.ReadMsgUnix(buf[:], oob); err != nil {
		t.Errorf("test 4: unexpected error: %s", err)

		return
	} else if n != 2 {
		t.Errorf("test 4: expecting to read 2 bytes, read %d", n)

		return
	} else if pr := uint16(buf[0]) | (uint16(buf[1]) << 8); pr != pa {
		t.Errorf("test 4: expecting to read port %d, got %d", pa, pr)

		return
	}

	nc, err := net.DialTCP("tcp", nil, &net.TCPAddr{Port: int(pa)})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	data := tlsServerName(aDomain)
	nc.Write(data)

	n, oobn, _, _, err := conn.ReadMsgUnix(buf[:], oob)
	if err != nil {
		t.Errorf("test 5: unexpected error: %s", err)
		return
	} else if !bytes.Equal(buf[:n], data) {
		t.Errorf("test 5: expecting to read TLS header %v, got %v", data, buf[:n])
		return
	}

	msg, _ := syscall.ParseSocketControlMessage(oob[:oobn])
	fd, _ := syscall.ParseUnixRights(&msg[0])
	nf = os.NewFile(uintptr(fd[0]), "")

	cn, err := net.FileConn(nf)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	nf.Close()

	addr := cn.LocalAddr().(*net.TCPAddr)
	if addr.Port != int(pa) {
		t.Errorf("test 6: expecting port %d, got %d", pa, addr.Port)

		return
	}

	nc.Write([]byte("TEST"))
	nc.Close()

	if n, err := cn.Read(buf[:]); err != nil {
		t.Errorf("test 7: unexpected error: %s", err)

		return
	} else if string(buf[:n]) != "TEST" {
		t.Errorf("test 7: expecting to read \"TEST\", read %q", buf[:n])

		return
	}

	buf[0] = uint8(pa)
	buf[1] = uint8(pa >> 8)

	n, _, err = conn.WriteMsgUnix(buf[:2], nil, nil)
	if err != nil {
		t.Errorf("test 8: unexpected error: %s", err)

		return
	} else if n != 2 {
		t.Errorf("test 8: expecting to write 2 bytes, wrote %d", n)

		return
	}

	time.Sleep(time.Second)

	l, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(pa)})
	if err != nil {
		t.Errorf("test 9: unexpected error: %s", err)
		return
	}

	l.Close()
}
