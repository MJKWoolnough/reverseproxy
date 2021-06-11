package reverseproxy

import (
	"net"
	"os"
	"testing"

	"github.com/foxcpp/go-mockdns"
)

const (
	aDomain = "aaa.com"
	bDomain = "bbb.com"
)

func TestMain(m *testing.M) {
	addrs := mockdns.Zone{A: []string{"127.0.0.1"}}
	srv, _ := mockdns.NewServer(map[string]mockdns.Zone{
		aDomain: addrs,
		bDomain: addrs,
	}, false)
	srv.PatchNet(net.DefaultResolver)
	code := m.Run()
	srv.Close()
	mockdns.UnpatchNet(net.DefaultResolver)
	os.Exit(code)
}

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
	return service == "aaa.com"
}

type testServiceB struct {
	testService
}

func (testServiceB) matchService(service string) bool {
	return service == "bbb.com"
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
