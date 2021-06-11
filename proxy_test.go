package reverseproxy

import (
	"net"
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
	t <- testData{buf, conn}
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

}
