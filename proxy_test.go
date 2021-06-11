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

func (t testService) Transfer(buf []byte, conn *net.TCPConn) {
	t <- testData{buf, conn}
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

func TestListener(t *testing.T) {

}
