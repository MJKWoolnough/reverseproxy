package reverseproxy

import (
	"fmt"
	"net"
	"os"
	"testing"
)

const (
	aDomain = "aaa.com"
	bDomain = "bbb.com"
)

type testData struct {
	buf  []byte
	conn *net.TCPConn
}

type testService chan testData

func (t testService) Transfer(buf []byte, conn *net.TCPConn) error {
	f, _ := conn.File()
	fc, _ := net.FileConn(f)
	f.Close()
	t <- testData{append(make([]byte, 0, len(buf)), buf...), fc.(*net.TCPConn)}
	return nil
}

type testServiceA struct {
	testService
}

func (testServiceA) matchService(service string) bool {
	return service == aDomain
}

type testServiceB struct {
	testService
}

func (testServiceB) matchService(service string) bool {
	return service == bDomain
}

func getUnusedPort() uint16 {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{})
	if err != nil {
		return 0
	}
	p := uint16(l.Addr().(*net.TCPAddr).Port)
	l.Close()
	return p
}

func TestListener(t *testing.T) {
	pa := getUnusedPort()
	sa := make(testService)
	p, err := addPort(pa, testServiceA{sa})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	const firstSend = "GET / HTTP/1.1\r\nHost: " + aDomain + "\r\n\r\n"
	go func() {
		c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", pa))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		c.Write([]byte(firstSend))
		c.Close()
	}()
	data := <-sa
	if string(data.buf) != firstSend {
		t.Errorf("test 1: expecting buf to equal %q, got %q", firstSend, data.buf)
		return
	}
	p.Close()
}
