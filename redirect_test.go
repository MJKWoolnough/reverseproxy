package reverseproxy

import (
	"io"
	"net"
	"testing"
)

func TestRedirect(t *testing.T) {
	la, err := net.ListenTCP("tcp", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	lb, err := net.ListenTCP("tcp", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	pna := getUnusedPort()
	pa, err := AddRedirect(HostName(aDomain), pna, la.Addr())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	pb, err := AddRedirect(HostName(bDomain), pna, lb.Addr())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	go func() {
		c, _ := net.DialTCP("tcp", nil, &net.TCPAddr{Port: int(pna)})
		c.Write([]byte("GET / HTTP/1.1\r\nHost: " + aDomain + "\r\n\r\nDATA"))
		c.Close()
	}()
	go func() {
		c, _ := net.DialTCP("tcp", nil, &net.TCPAddr{Port: int(pna)})
		c.Write(tlsServerName(bDomain))
		c.Write([]byte("ATAD"))
		c.Close()
	}()
	c, err := la.Accept()
	if err != nil {
		t.Errorf("test 1: unexpected error: %s", err)
		return
	}
	d, err := lb.Accept()
	if err != nil {
		t.Errorf("test 2: unexpected error: %s", err)
		return
	}
	buf, err := io.ReadAll(c)
	n := len(buf)
	if err != nil {
		t.Errorf("test 3: unexpected error: %s", err)
		return
	} else if string(buf[n-4:n]) != "DATA" {
		t.Errorf("test 3: expecting \"DATA\", got %q", buf[n-4:n])
		return
	}
	c.Close()
	buf, err = io.ReadAll(d)
	n = len(buf)
	if err != nil {
		t.Errorf("test 4: unexpected error: %s", err)
		return
	} else if string(buf[n-4:n]) != "ATAD" {
		t.Errorf("test 4: expecting \"ATAD\", got %q", buf[n-4:n])
		return
	}
	d.Close()
	pa.Close()
	pb.Close()
}
