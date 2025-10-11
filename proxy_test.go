package reverseproxy

import (
	"bytes"
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

func (t testService) Transfer(buf []byte, conn *net.TCPConn) {
	t <- testData{append(make([]byte, 0, len(buf)), buf...), conn}
}

func (t testService) Active() bool {
	return false
}

type testServiceA struct {
	testService
}

func (testServiceA) MatchService(service string) bool {
	return service == aDomain
}

type testServiceB struct {
	testService
}

func (testServiceB) MatchService(service string) bool {
	return service == bDomain
}

func getUnusedPort() uint16 {
	l, err := net.ListenTCP("tcp", nil)
	if err != nil {
		return 0
	}

	p := uint16(l.Addr().(*net.TCPAddr).Port)

	l.Close()

	return p
}

func TestListener(t *testing.T) {
	sync := make(chan struct{})
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

		<-sync

		c.Write([]byte{127})
		c.Close()
	}()

	data := <-sa
	sync <- struct{}{}

	if string(data.buf) != firstSend {
		t.Errorf("test 1: expecting buf to equal %q, got %q", firstSend, data.buf)

		return
	}

	var buf [32]byte

	if n, err := data.conn.Read(buf[:]); err != nil {
		t.Errorf("test 2: unexpected error: %s", err)

		return
	} else if n != 1 {
		t.Errorf("test 2: expecting to read 1 byte, read %d", n)

		return
	} else if buf[0] != 127 {
		t.Errorf("test 2: expecting to read 127, read %d", buf[0])

		return
	} else if err = data.conn.Close(); err != nil {
		t.Errorf("test 3: unexpected error: %s", err)

		return
	}

	sb := make(testService)

	q, err := addPort(pa, testServiceB{sb})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	const secondSend = "GET / HTTP/1.1\r\nHost: " + bDomain + "\r\n\r\n"

	go func() {
		c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", pa))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		c.Write([]byte(secondSend))

		<-sync

		c.Write([]byte{255, 127})
		c.Close()
	}()

	data = <-sb
	sync <- struct{}{}

	if string(data.buf) != secondSend {
		t.Errorf("test 4: expecting buf to equal %q, got %q", secondSend, data.buf)

		return
	}

	if n, err := data.conn.Read(buf[:]); err != nil {
		t.Errorf("test 5: unexpected error: %s", err)

		return
	} else if n != 2 {
		t.Errorf("test 5: expecting to read 1 byte, read %d", n)

		return
	} else if buf[0] != 255 || buf[1] != 127 {
		t.Errorf("test 5: expecting to read 255, 127, read %v", buf[:2])

		return
	} else if err = data.conn.Close(); err != nil {
		t.Errorf("test 6: unexpected error: %s", err)
		return
	}

	go func() {
		c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", pa))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		c.Write([]byte(firstSend))

		<-sync

		c.Write([]byte{1, 2, 3})
		c.Close()
	}()

	go func() {
		<-sync

		c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", pa))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		c.Write([]byte(secondSend))

		<-sync

		c.Write([]byte{4, 5, 6, 7})
		c.Close()
	}()

	data = <-sa
	sync <- struct{}{}
	sync <- struct{}{}
	dataB := <-sb
	sync <- struct{}{}

	if string(data.buf) != firstSend {
		t.Errorf("test 7: expecting buf to equal %q, got %q", firstSend, data.buf)

		return
	} else if string(dataB.buf) != secondSend {
		t.Errorf("test 8: expecting buf to equal %q, got %q", secondSend, dataB.buf)

		return
	} else if n, err := data.conn.Read(buf[:]); err != nil {
		t.Errorf("test 9: unexpected error: %s", err)

		return
	} else if n != 3 {
		t.Errorf("test 9: expecting to read 1 byte, read %d", n)

		return
	} else if !bytes.Equal(buf[:3], []byte{1, 2, 3}) {
		t.Errorf("test 9: expecting to read 1, 2, 3, read %v", buf[:3])

		return
	} else if n, err = dataB.conn.Read(buf[:]); err != nil {
		t.Errorf("test 10: unexpected error: %s", err)

		return
	} else if n != 4 {
		t.Errorf("test 10: expecting to read 1 byte, read %d", n)

		return
	} else if !bytes.Equal(buf[:4], []byte{4, 5, 6, 7}) {
		t.Errorf("test 10: expecting to read 4, 5, 6, 7, read %v", buf[:4])

		return
	} else if err = data.conn.Close(); err != nil {
		t.Errorf("test 11: unexpected error: %s", err)

		return
	} else if err = dataB.conn.Close(); err != nil {
		t.Errorf("test 12: unexpected error: %s", err)

		return
	}

	p.Close()

	tlsData := tlsServerName(bDomain)

	go func() {
		c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", pa))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		c.Write(tlsData)

		<-sync

		c.Write([]byte{3, 2, 1, 0})
		c.Close()
	}()

	dataB = <-sb
	sync <- struct{}{}

	if !bytes.Equal(dataB.buf, tlsData) {
		t.Errorf("test 13: expected to read TLS Header, read %v", dataB.buf)

		return
	}

	if n, err := dataB.conn.Read(buf[:]); err != nil {
		t.Errorf("test 14: unexpected error: %s", err)

		return
	} else if n != 4 {
		t.Errorf("test 14: expecting to read 1 byte, read %d", n)

		return
	} else if !bytes.Equal(buf[:4], []byte{3, 2, 1, 0}) {
		t.Errorf("test 14: expecting to read 3, 2, 1, 0, read %v", buf[:4])

		return
	} else if err = dataB.conn.Close(); err != nil {
		t.Errorf("test 15: unexpected error: %s", err)

		return
	}

	q.Close()

	l, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(pa)})
	if err != nil {
		t.Errorf("test 16: unexpected error: %s", err)
		return
	}

	l.Close()
}
